package mealplan

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type MealPlan struct {
	ID        int        `json:"id"`
	Name      string     `json:"name"`
	StartDate string     `json:"start_date"` // "YYYY-MM-DD"
	EndDate   string     `json:"end_date"`   // "YYYY-MM-DD"
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

type CreateMealPlanRequest struct {
	Name      string `json:"name" binding:"required"`
	StartDate string `json:"start_date" binding:"required"`
	EndDate   string `json:"end_date" binding:"required"`
}

type UpdateMealPlanRequest struct {
	Name      *string `json:"name"`
	StartDate *string `json:"start_date"`
	EndDate   *string `json:"end_date"`
}

type MealPlanRecipe struct {
	ID          int     `json:"id"`
	MealPlanID  int     `json:"meal_plan_id"`
	RecipeID    *int    `json:"recipe_id,omitempty"`
	MealType    *string `json:"meal_type,omitempty"`    // breakfast/lunch/dinner/snack
	PlannedDate *string `json:"planned_date,omitempty"` // "YYYY-MM-DD"
}

type CreateMealPlanRecipeRequest struct {
	RecipeID    int     `json:"recipe_id" binding:"required"`
	MealType    *string `json:"meal_type"`
	PlannedDate *string `json:"planned_date"`
}

type UpdateMealPlanRecipeRequest struct {
	RecipeID    *int    `json:"recipe_id"`
	MealType    *string `json:"meal_type"`
	PlannedDate *string `json:"planned_date"`
}

func ListMealPlansHandler(c *gin.Context, db *sql.DB) {
	rows, err := db.Query(`
		SELECT id, name, start_date, end_date, created_at
		FROM meal_plans
		ORDER BY start_date ASC, id ASC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query meal plans"})
		return
	}
	defer rows.Close()

	var list []MealPlan
	for rows.Next() {
		var mp MealPlan
		if err := rows.Scan(&mp.ID, &mp.Name, &mp.StartDate, &mp.EndDate, &mp.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		list = append(list, mp)
	}

	c.JSON(http.StatusOK, list)
}

func CreateMealPlanHandler(c *gin.Context, db *sql.DB) {
	var req CreateMealPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	res, err := db.Exec(`
		INSERT INTO meal_plans (name, start_date, end_date)
		VALUES (?, ?, ?)
	`, req.Name, req.StartDate, req.EndDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert meal plan"})
		return
	}

	id64, err := res.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get id"})
		return
	}
	id := int(id64)

	var mp MealPlan
	err = db.QueryRow(`
		SELECT id, name, start_date, end_date, created_at
		FROM meal_plans
		WHERE id = ?
	`, id).Scan(&mp.ID, &mp.Name, &mp.StartDate, &mp.EndDate, &mp.CreatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "created but failed to reload"})
		return
	}

	c.JSON(http.StatusCreated, mp)
}

func GetMealPlanHandler(c *gin.Context, db *sql.DB) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var mp MealPlan
	err = db.QueryRow(`
		SELECT id, name, start_date, end_date, created_at
		FROM meal_plans
		WHERE id = ?
	`, id).Scan(&mp.ID, &mp.Name, &mp.StartDate, &mp.EndDate, &mp.CreatedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	c.JSON(http.StatusOK, mp)
}

func UpdateMealPlanHandler(c *gin.Context, db *sql.DB) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req UpdateMealPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	// Load existing
	var mp MealPlan
	err = db.QueryRow(`
		SELECT id, name, start_date, end_date, created_at
		FROM meal_plans
		WHERE id = ?
	`, id).Scan(&mp.ID, &mp.Name, &mp.StartDate, &mp.EndDate, &mp.CreatedAt)
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
		mp.Name = *req.Name
	}
	if req.StartDate != nil {
		mp.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		mp.EndDate = *req.EndDate
	}

	_, err = db.Exec(`
		UPDATE meal_plans
		SET name = ?, start_date = ?, end_date = ?
		WHERE id = ?
	`, mp.Name, mp.StartDate, mp.EndDate, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update"})
		return
	}

	c.JSON(http.StatusOK, mp)
}

func DeleteMealPlanHandler(c *gin.Context, db *sql.DB) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	res, err := db.Exec(`DELETE FROM meal_plans WHERE id = ?`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete"})
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	// Cascade will remove meal_plan_recipes automatically
	c.Status(http.StatusNoContent)
}

func ListMealPlanRecipesHandler(c *gin.Context, db *sql.DB) {
	mpIDStr := c.Param("id")
	mpID, err := strconv.Atoi(mpIDStr)
	if err != nil || mpID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid meal plan id"})
		return
	}

	rows, err := db.Query(`
		SELECT id, meal_plan_id, recipe_id, meal_type, planned_date
		FROM meal_plan_recipes
		WHERE meal_plan_id = ?
		ORDER BY planned_date ASC, id ASC
	`, mpID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query meal plan recipes"})
		return
	}
	defer rows.Close()

	var list []MealPlanRecipe
	for rows.Next() {
		var mpr MealPlanRecipe
		if err := rows.Scan(&mpr.ID, &mpr.MealPlanID, &mpr.RecipeID, &mpr.MealType, &mpr.PlannedDate); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		list = append(list, mpr)
	}

	c.JSON(http.StatusOK, list)
}

func CreateMealPlanRecipeHandler(c *gin.Context, db *sql.DB) {
	mpIDStr := c.Param("id")
	mpID, err := strconv.Atoi(mpIDStr)
	if err != nil || mpID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid meal plan id"})
		return
	}

	// Ensure meal plan exists
	var exists int
	if err := db.QueryRow(`SELECT 1 FROM meal_plans WHERE id = ?`, mpID).Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "meal plan not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate meal plan"})
		return
	}

	var req CreateMealPlanRecipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	// Ensure recipe exists (optional but nicer error than FK)
	if err := db.QueryRow(`SELECT 1 FROM recipes WHERE id = ?`, req.RecipeID).Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "recipe not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate recipe"})
		return
	}

	res, err := db.Exec(`
		INSERT INTO meal_plan_recipes (meal_plan_id, recipe_id, meal_type, planned_date)
		VALUES (?, ?, ?, ?)
	`, mpID, req.RecipeID, req.MealType, req.PlannedDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert meal plan recipe"})
		return
	}

	id64, err := res.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get id"})
		return
	}
	id := int(id64)

	var mpr MealPlanRecipe
	err = db.QueryRow(`
		SELECT id, meal_plan_id, recipe_id, meal_type, planned_date
		FROM meal_plan_recipes
		WHERE id = ?
	`, id).Scan(&mpr.ID, &mpr.MealPlanID, &mpr.RecipeID, &mpr.MealType, &mpr.PlannedDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "created but failed to reload"})
		return
	}

	c.JSON(http.StatusCreated, mpr)
}

func GetMealPlanRecipeHandler(c *gin.Context, db *sql.DB) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var mpr MealPlanRecipe
	err = db.QueryRow(`
		SELECT id, meal_plan_id, recipe_id, meal_type, planned_date
		FROM meal_plan_recipes
		WHERE id = ?
	`, id).Scan(&mpr.ID, &mpr.MealPlanID, &mpr.RecipeID, &mpr.MealType, &mpr.PlannedDate)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	c.JSON(http.StatusOK, mpr)
}

func UpdateMealPlanRecipeHandler(c *gin.Context, db *sql.DB) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req UpdateMealPlanRecipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	// Load existing
	var current MealPlanRecipe
	err = db.QueryRow(`
		SELECT id, meal_plan_id, recipe_id, meal_type, planned_date
		FROM meal_plan_recipes
		WHERE id = ?
	`, id).Scan(&current.ID, &current.MealPlanID, &current.RecipeID, &current.MealType, &current.PlannedDate)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	// Apply patch
	if req.RecipeID != nil {
		// optionally check recipe exists
		current.RecipeID = req.RecipeID
	}
	if req.MealType != nil {
		current.MealType = req.MealType
	}
	if req.PlannedDate != nil {
		current.PlannedDate = req.PlannedDate
	}

	_, err = db.Exec(`
		UPDATE meal_plan_recipes
		SET recipe_id = ?, meal_type = ?, planned_date = ?
		WHERE id = ?
	`, current.RecipeID, current.MealType, current.PlannedDate, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update"})
		return
	}

	c.JSON(http.StatusOK, current)
}

func DeleteMealPlanRecipeHandler(c *gin.Context, db *sql.DB) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	res, err := db.Exec(`DELETE FROM meal_plan_recipes WHERE id = ?`, id)
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
