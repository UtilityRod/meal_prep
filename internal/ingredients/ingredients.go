package ingredients

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Ingredient struct {
	ID       int     `json:"id"`
	RecipeID int     `json:"recipe_id"`
	Name     string  `json:"name"`
	Quantity *string `json:"quantity,omitempty"`
	Unit     *string `json:"unit,omitempty"`
}

type CreateIngredientRequest struct {
	Name     string  `json:"name" binding:"required"`
	Quantity *string `json:"quantity"`
	Unit     *string `json:"unit"`
}

type UpdateIngredientRequest struct {
	Name     *string `json:"name"`     // optional
	Quantity *string `json:"quantity"` // optional
	Unit     *string `json:"unit"`     // optional
}

func ListIngredientsForRecipeHandler(c *gin.Context, db *sql.DB) {
	recipeIDStr := c.Param("id")
	recipeID, err := strconv.Atoi(recipeIDStr)

	if err != nil || recipeID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipe id"})
		return
	}

	rows, err := db.Query(`
		SELECT id, recipe_id, name, quantity, unit
		FROM recipe_ingredients
		WHERE recipe_id = ?
		ORDER BY id ASC
	`, recipeID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query ingredients"})
		return
	}

	defer rows.Close()

	var list []Ingredient
	for rows.Next() {
		var ing Ingredient
		if err := rows.Scan(&ing.ID, &ing.RecipeID, &ing.Name, &ing.Quantity, &ing.Unit); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}

		list = append(list, ing)
	}

	c.JSON(http.StatusOK, list)
}

func CreateIngredientForRecipeHandler(c *gin.Context, db *sql.DB) {
	recipeIDStr := c.Param("id")
	recipeID, err := strconv.Atoi(recipeIDStr)
	if err != nil || recipeID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipe id"})
		return
	}

	// Ensure recipe exists (otherwise FK will fail with a vague error)
	var exists int
	if err := db.QueryRow(`SELECT 1 FROM recipes WHERE id = ?`, recipeID).Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "recipe not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate recipe"})
		return
	}

	var req CreateIngredientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	res, err := db.Exec(`
		INSERT INTO recipe_ingredients (recipe_id, name, quantity, unit)
		VALUES (?, ?, ?, ?)
	`, recipeID, req.Name, req.Quantity, req.Unit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert ingredient"})
		return
	}

	id64, err := res.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get new ingredient id"})
		return
	}
	id := int(id64)

	var ing Ingredient
	err = db.QueryRow(`
		SELECT id, recipe_id, name, quantity, unit
		FROM recipe_ingredients
		WHERE id = ?
	`, id).Scan(&ing.ID, &ing.RecipeID, &ing.Name, &ing.Quantity, &ing.Unit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "created but failed to reload"})
		return
	}

	c.JSON(http.StatusCreated, ing)
}

func GetIngredientHandler(c *gin.Context, db *sql.DB) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var ing Ingredient
	err = db.QueryRow(`
		SELECT id, recipe_id, name, quantity, unit
		FROM recipe_ingredients
		WHERE id = ?
	`, id).Scan(&ing.ID, &ing.RecipeID, &ing.Name, &ing.Quantity, &ing.Unit)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	c.JSON(http.StatusOK, ing)
}

func UpdateIngredientHandler(c *gin.Context, db *sql.DB) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req UpdateIngredientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	// Load existing
	var current Ingredient
	err = db.QueryRow(`
		SELECT id, recipe_id, name, quantity, unit
		FROM recipe_ingredients
		WHERE id = ?
	`, id).Scan(&current.ID, &current.RecipeID, &current.Name, &current.Quantity, &current.Unit)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	// Apply patch
	if req.Name != nil {
		current.Name = *req.Name
	}
	if req.Quantity != nil {
		current.Quantity = req.Quantity
	}
	if req.Unit != nil {
		current.Unit = req.Unit
	}

	_, err = db.Exec(`
		UPDATE recipe_ingredients
		SET name = ?, quantity = ?, unit = ?
		WHERE id = ?
	`, current.Name, current.Quantity, current.Unit, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update"})
		return
	}

	c.JSON(http.StatusOK, current)
}

func DeleteIngredientHandler(c *gin.Context, db *sql.DB) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	res, err := db.Exec(`DELETE FROM recipe_ingredients WHERE id = ?`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete"})
		return
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	c.Status(http.StatusNoContent)
}
