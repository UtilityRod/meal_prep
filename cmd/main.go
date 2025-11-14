package main

import (
	"fmt"
	"log"
	"meal_prep/internal/db"
	"meal_prep/internal/ingredients"
	mealplan "meal_prep/internal/meal_plan"
	"meal_prep/internal/recipes"

	"github.com/gin-gonic/gin"
)

const (
	DBPath = "/tmp/meal_prep.db"
)

func main() {
	mealDB, err := db.Open(DBPath)

	if err != nil {
		log.Fatal(err)
	}

	err = db.Init(mealDB)

	if err != nil {
		log.Fatal(err)
	}

	r := gin.Default()

	v1 := r.Group("/v1")
	{
		// v1.GET("/health", func(c *gin.Context) {
		// 	c.JSON(http.StatusOK, gin.H{"status": "ok"})
		// })
		v1.GET("/recipes", func(c *gin.Context) { recipes.ListRecipesHandler(c, mealDB) })
		v1.POST("/recipes", func(c *gin.Context) { recipes.CreateRecipeHandler(c, mealDB) })
		v1.GET("/recipes/:id", func(c *gin.Context) { recipes.GetRecipeHandler(c, mealDB) })
		v1.GET("/recipes/:id/ingredients", func(c *gin.Context) { ingredients.ListIngredientsForRecipeHandler(c, mealDB) })
		v1.POST("/recipes/:id/ingredients", func(c *gin.Context) { ingredients.CreateIngredientForRecipeHandler(c, mealDB) })

		v1.GET("/ingredients/:id", func(c *gin.Context) { ingredients.GetIngredientHandler(c, mealDB) })
		v1.PUT("/ingredients/:id", func(c *gin.Context) { ingredients.UpdateIngredientHandler(c, mealDB) })
		v1.DELETE("/ingredients/:id", func(c *gin.Context) { ingredients.DeleteIngredientHandler(c, mealDB) })

		v1.GET("/meal-plans", func(c *gin.Context) { mealplan.ListMealPlansHandler(c, mealDB) })
		v1.POST("/meal-plans", func(c *gin.Context) { mealplan.CreateMealPlanHandler(c, mealDB) })
		v1.GET("/meal-plans/:id", func(c *gin.Context) { mealplan.GetMealPlanHandler(c, mealDB) })
		v1.PUT("/meal-plans/:id", func(c *gin.Context) { mealplan.UpdateMealPlanHandler(c, mealDB) })
		v1.DELETE("/meal-plans/:id", func(c *gin.Context) { mealplan.DeleteMealPlanHandler(c, mealDB) })

		// Recipes inside a meal plan
		v1.GET("/meal-plans/:id/recipes", func(c *gin.Context) { mealplan.ListMealPlanRecipesHandler(c, mealDB) })
		v1.POST("/meal-plans/:id/recipes", func(c *gin.Context) { mealplan.CreateMealPlanRecipeHandler(c, mealDB) })

		// Single meal_plan_recipes entries
		v1.GET("/plan-recipes/:id", func(c *gin.Context) { mealplan.GetMealPlanRecipeHandler(c, mealDB) })
		v1.PUT("/plan-recipes/:id", func(c *gin.Context) { mealplan.UpdateMealPlanRecipeHandler(c, mealDB) })
		v1.DELETE("/plan-recipes/:id", func(c *gin.Context) { mealplan.DeleteMealPlanRecipeHandler(c, mealDB) })
	}

	r.Static("/app", "./public")

	log.Println("listening on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
	fmt.Println("exiting...")

	if err = mealDB.Close(); err != nil {
		log.Fatal("failed to gracefully close server, force quiting...")
	}

}
