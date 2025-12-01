package main

import (
	"fmt"
	"os"
	"time"

	"meal_prep/internal/db"
	"meal_prep/internal/ingredients"
	mealplan "meal_prep/internal/meal_plan"
	"meal_prep/internal/logging"
	"meal_prep/internal/recipes"
	mongostore "meal_prep/internal/mongo"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
)

const (
	DBPath = "/tmp/meal_prep.db"
)

func main() {
	// initialize structured logger
	logger, sugar, err := logging.InitLogger()
	if err != nil {
		fmt.Printf("failed to init logger: %v\n", err)
		return
	}
	defer logger.Sync()

	mealDB, err := db.Open(DBPath)

	if err != nil {
		sugar.Fatalw("failed to open DB", "error", err)
	}

	err = db.Init(mealDB)

	if err != nil {
		sugar.Fatalw("failed to init DB", "error", err)
	}

	// create router and attach zap middleware
	r := gin.New()
	r.Use(ginzap.Ginzap(logger, time.RFC3339, true))
	r.Use(ginzap.RecoveryWithZap(logger, true))

	r.StaticFile("/", "./public/index.html")
    
	// Serve the Meal Search placeholder page directly at /meal_search.html
	r.StaticFile("/meal_search.html", "./public/meal_search.html")

	// Serve the meal detail page
	r.StaticFile("/meal_detail.html", "./public/meal_detail.html")

	// Initialize Mongo + Redis store if configured via env
	var store *mongostore.Store
	mongoURI := os.Getenv("MONGO_URI")
	redisAddr := os.Getenv("REDIS_ADDR")
	mongoDBName := os.Getenv("MONGO_DB")
	if mongoDBName == "" {
		mongoDBName = "meal_prep"
	}
	if mongoURI != "" {
		s, err := mongostore.New(mongoURI, mongoDBName, redisAddr, sugar)
		if err != nil {
			sugar.Warnw("failed to init mongo store", "error", err)
		} else {
			store = s
			if os.Getenv("IMPORT_MEALS_ON_START") == "true" {
				mealsPath := resolveMealsFile()
				if mealsPath == "" {
					sugar.Infow("no meals file found; skipping import", "tried", "MEALS_FILE and common locations")
				} else {
					sugar.Infow("importing meals", "path", mealsPath)
					if err := store.ImportFromFile(mealsPath); err != nil {
						sugar.Errorw("failed to import meals", "error", err)
					} else {
						sugar.Infow("import complete")
					}
				}
			}
		}
	}

    

	v1 := r.Group("/v1")
	{
		v1.GET("/recipes", func(c *gin.Context) { recipes.ListRecipesHandler(c, mealDB, sugar) })
		v1.POST("/recipes", func(c *gin.Context) { recipes.CreateRecipeHandler(c, mealDB, sugar) })
		v1.POST("/recipes/import-from-meal", func(c *gin.Context) { recipes.ImportMealHandler(c, mealDB, sugar) })
		v1.GET("/recipes/:id", func(c *gin.Context) { recipes.GetRecipeHandler(c, mealDB, sugar) })
		v1.DELETE("/recipes/:id", func(c *gin.Context) { recipes.DeleteRecipeHandler(c, mealDB, sugar) })
		v1.GET("/recipes/:id/ingredients", func(c *gin.Context) { ingredients.ListIngredientsForRecipeHandler(c, mealDB, sugar) })
		v1.POST("/recipes/:id/ingredients", func(c *gin.Context) { ingredients.CreateIngredientForRecipeHandler(c, mealDB, sugar) })

		v1.GET("/ingredients/:id", func(c *gin.Context) { ingredients.GetIngredientHandler(c, mealDB, sugar) })
		v1.PUT("/ingredients/:id", func(c *gin.Context) { ingredients.UpdateIngredientHandler(c, mealDB, sugar) })
		v1.DELETE("/ingredients/:id", func(c *gin.Context) { ingredients.DeleteIngredientHandler(c, mealDB, sugar) })

		v1.GET("/meal-plans", func(c *gin.Context) { mealplan.ListMealPlansHandler(c, mealDB, sugar) })
		v1.POST("/meal-plans", func(c *gin.Context) { mealplan.CreateMealPlanHandler(c, mealDB, sugar) })
		v1.GET("/meal-plans/:id", func(c *gin.Context) { mealplan.GetMealPlanHandler(c, mealDB, sugar) })
		v1.PUT("/meal-plans/:id", func(c *gin.Context) { mealplan.UpdateMealPlanHandler(c, mealDB, sugar) })
		v1.DELETE("/meal-plans/:id", func(c *gin.Context) { mealplan.DeleteMealPlanHandler(c, mealDB, sugar) })

		// Recipes inside a meal plan
		v1.GET("/meal-plans/:id/recipes", func(c *gin.Context) { mealplan.ListMealPlanRecipesHandler(c, mealDB, sugar) })
		v1.POST("/meal-plans/:id/recipes", func(c *gin.Context) { mealplan.CreateMealPlanRecipeHandler(c, mealDB, sugar) })

		// Single meal_plan_recipes entries
		v1.GET("/plan-recipes/:id", func(c *gin.Context) { mealplan.GetMealPlanRecipeHandler(c, mealDB, sugar) })
		v1.PUT("/plan-recipes/:id", func(c *gin.Context) { mealplan.UpdateMealPlanRecipeHandler(c, mealDB, sugar) })
		v1.DELETE("/plan-recipes/:id", func(c *gin.Context) { mealplan.DeleteMealPlanRecipeHandler(c, mealDB, sugar) })
	}

	// Meals API using Mongo+Redis store (register route regardless of store presence)
	v1.GET("/meals/:id", func(c *gin.Context) {
		if store == nil {
			c.JSON(503, gin.H{"error": "meals store not configured"})
			return
		}
		id := c.Param("id")
		m, err := store.GetMealByID(id)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		if m == nil {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}
		c.JSON(200, m)
	})
	r.Static("/app", "./public")

	sugar.Infow("listening", "addr", ":80")
	if err := r.Run(":80"); err != nil {
		sugar.Fatalw("server failed", "error", err)
	}
	sugar.Infow("exiting")

	if err = mealDB.Close(); err != nil {
		sugar.Fatalw("failed to gracefully close DB", "error", err)
	}

}

// resolveMealsFile returns a path to the meals JSON file. It checks the MEALS_FILE env var
// first, then a set of sensible locations commonly used in this project's containers.
func resolveMealsFile() string {
	// allow explicit override
	if p := os.Getenv("MEALS_FILE"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// common locations to try (relative and absolute)
	candidates := []string{
		"./all_meals.json",
		"/root/all_meals.json",
		"/root/public/all_meals.json",
		"/app/all_meals.json",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}
