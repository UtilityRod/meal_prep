package main

import (
	"fmt"
	"log"
	"meal_prep/internal/db"
	"meal_prep/internal/ingredients"
	"meal_prep/internal/recipes"
	"net/http"

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
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := r.Group("/v1")
	{
		v1.GET("/recipes", func(c *gin.Context) { recipes.ListRecipesHandler(c, mealDB) })
		v1.POST("/recipes", func(c *gin.Context) { recipes.CreateRecipeHandler(c, mealDB) })
		v1.GET("/recipes/:id", func(c *gin.Context) { recipes.GetRecipeHandler(c, mealDB) })
		v1.GET("/recipes/:id/ingredients", func(c *gin.Context) { ingredients.ListIngredientsForRecipeHandler(c, mealDB) })
		v1.POST("/recipes/:id/ingredients", func(c *gin.Context) { ingredients.CreateIngredientForRecipeHandler(c, mealDB) })

		v1.GET("/ingredients/:id", func(c *gin.Context) { ingredients.GetIngredientHandler(c, mealDB) })
		v1.PUT("/ingredients/:id", func(c *gin.Context) { ingredients.UpdateIngredientHandler(c, mealDB) })
		v1.DELETE("/ingredients/:id", func(c *gin.Context) { ingredients.DeleteIngredientHandler(c, mealDB) })
	}

	log.Println("listening on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
	fmt.Println("exiting...")

	if err = mealDB.Close(); err != nil {
		log.Fatal("failed to gracefully close server, force quiting...")
	}

}
