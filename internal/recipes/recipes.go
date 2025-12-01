package recipes

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Recipe struct {
	ID          int        `json:"id"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	Instructions *string   `json:"instructions,omitempty"`
	YoutubeLink  *string   `json:"youtube_link,omitempty"`
	Servings    *int       `json:"servings,omitempty"`
	PrepTime    *int       `json:"prep_time,omitempty"`
	CookTime    *int       `json:"cook_time,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

type CreateRecipeRequest struct {
	Title       string  `json:"title" binding:"required"`
	Description *string `json:"description"`
	Instructions *string `json:"instructions"`
	YoutubeLink  *string `json:"youtube_link"`
	Servings    *int    `json:"servings"`
	PrepTime    *int    `json:"prep_time"`
	CookTime    *int    `json:"cook_time"`
}

func ListRecipesHandler(c *gin.Context, db *sql.DB, sugar *zap.SugaredLogger) {
	rows, err := db.Query(`
SELECT id, title, description, instructions, youtube_link, servings, prep_time, cook_time, created_at, updated_at
FROM recipes
ORDER BY created_at DESC
LIMIT 100
	`)

	if err != nil {
		if sugar != nil {
			sugar.Errorw("failed to query recipes", "error", err)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query recipes"})
		return
	}

	defer rows.Close()

	var recipes []Recipe
	for rows.Next() {
		var r Recipe
		if err := rows.Scan(
			&r.ID, &r.Title, &r.Description, &r.Instructions, &r.YoutubeLink, &r.Servings,
			&r.PrepTime, &r.CookTime, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}

		recipes = append(recipes, r)
	}

	c.JSON(http.StatusOK, recipes)
}

func GetRecipeHandler(c *gin.Context, db *sql.DB, sugar *zap.SugaredLogger) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var r Recipe
	err = db.QueryRow(`
SELECT id, title, description, instructions, youtube_link, servings, prep_time, cook_time, created_at, updated_at
FROM recipes
WHERE id = ?
	`, id).Scan(
		&r.ID, &r.Title, &r.Description, &r.Instructions, &r.YoutubeLink, &r.Servings,
		&r.PrepTime, &r.CookTime, &r.CreatedAt, &r.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		if sugar != nil {
			sugar.Errorw("db error getting recipe", "error", err)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	c.JSON(http.StatusOK, r)
}

func CreateRecipeHandler(c *gin.Context, db *sql.DB, sugar *zap.SugaredLogger) {
	var req CreateRecipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	res, err := db.Exec(`
		INSERT INTO recipes (title, description, instructions, youtube_link, servings, prep_time, cook_time)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, req.Title, req.Description, req.Instructions, req.YoutubeLink, req.Servings, req.PrepTime, req.CookTime)
	if err != nil {
		if sugar != nil {
			sugar.Errorw("failed to insert recipe", "error", err)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert"})
		return
	}

	id64, err := res.LastInsertId()
	if err != nil {
		if sugar != nil {
			sugar.Errorw("failed to get last insert id", "error", err)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get id"})
		return
	}
	id := int(id64)

	// Return the created recipe (fetch by id)
	var r Recipe
	err = db.QueryRow(`
		SELECT id, title, description, instructions, youtube_link, servings, prep_time, cook_time, created_at, updated_at
		FROM recipes WHERE id = ?
	`, id).Scan(
		&r.ID, &r.Title, &r.Description, &r.Instructions, &r.YoutubeLink, &r.Servings,
		&r.PrepTime, &r.CookTime, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		if sugar != nil {
			sugar.Errorw("created but failed to reload recipe", "error", err)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "created but failed to reload"})
		return
	}

	c.JSON(http.StatusCreated, r)
}

// ImportMealRequest accepts meal data and ingredients (with raw measures) and creates
// a recipe and its ingredients in SQLite. Ingredients' measures are parsed into
// quantity and unit according to simple rules.
type ImportIngredient struct {
	Name    string `json:"name" binding:"required"`
	Measure string `json:"measure"`
}

type ImportMealRequest struct {
	Title       string             `json:"title" binding:"required"`
	Description *string            `json:"description"`
	Servings    *int               `json:"servings"`
	PrepTime    *int               `json:"prep_time"`
	CookTime    *int               `json:"cook_time"`
	Instructions *string           `json:"instructions"`
	YoutubeLink  *string           `json:"youtube_link"`
	Ingredients []ImportIngredient `json:"ingredients"`
}

func ImportMealHandler(c *gin.Context, db *sql.DB, sugar *zap.SugaredLogger) {
	var req ImportMealRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to begin tx"})
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	res, err := tx.Exec(`
		INSERT INTO recipes (title, description, instructions, youtube_link, servings, prep_time, cook_time)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, req.Title, req.Description, req.Instructions, req.YoutubeLink, req.Servings, req.PrepTime, req.CookTime)
	if err != nil {
		if sugar != nil {
			sugar.Errorw("failed to insert recipe (import)", "error", err)
		}
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert recipe"})
		return
	}

	id64, err := res.LastInsertId()
	if err != nil {
		if sugar != nil {
			sugar.Errorw("failed to lastinsertid (import)", "error", err)
		}
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get id"})
		return
	}
	recipeID := int(id64)

	// helper: parse measure into quantity and unit per rules
	parseMeasure := func(m string) (quantity *string, unit *string) {
		s := strings.TrimSpace(m)
		if s == "" {
			return nil, nil
		}
		// if contains '(' or ' or ' then map everything to quantity
		if strings.Contains(s, "(") || strings.Contains(strings.ToLower(s), " or ") {
			q := s
			return &q, nil
		}

		// pattern: number (integer/decimal/fraction) followed by space + text
		// e.g. "1 1/2 cup", "1.5cups", "1/2 tsp"
		// try to find leading numeric
		re := regexp.MustCompile(`^([0-9]+(?:\s+[0-9]+/[0-9]+)?|[0-9]+/[0-9]+|[0-9]+(?:\.[0-9]+)?)(?:\s*)(.*)$`)
		mparts := re.FindStringSubmatch(s)
		if len(mparts) == 3 {
			q := strings.TrimSpace(mparts[1])
			u := strings.TrimSpace(mparts[2])
			if u == "" {
				// nothing left -> all quantity
				return &q, nil
			}
			return &q, &u
		}

		// fallback: map entire measure to quantity
		qq := s
		return &qq, nil
	}

	// insert ingredients
	for _, ing := range req.Ingredients {
		q, u := parseMeasure(ing.Measure)
		_, err := tx.Exec(`INSERT INTO recipe_ingredients (recipe_id, name, quantity, unit) VALUES (?, ?, ?, ?)`, recipeID, ing.Name, q, u)
		if err != nil {
			if sugar != nil {
				sugar.Errorw("failed to insert ingredient (import)", "error", err, "name", ing.Name)
			}
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert ingredient"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		if sugar != nil {
			sugar.Errorw("failed to commit import tx", "error", err)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"recipe_id": recipeID})
}

func DeleteRecipeHandler(c *gin.Context, db *sql.DB, sugar *zap.SugaredLogger) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipe id"})
		return
	}

	// Execute delete
	res, err := db.Exec(`DELETE FROM recipes WHERE id = ?`, id)
	if err != nil {
		if sugar != nil {
			sugar.Errorw("failed to delete recipe", "error", err)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rows, err := res.RowsAffected()
	if err != nil {
		if sugar != nil {
			sugar.Errorw("failed rows affected for delete", "error", err)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "recipe not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "deleted",
		"recipeId": id,
	})
}
