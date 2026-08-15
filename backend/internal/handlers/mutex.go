package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"promo-engine/internal/db"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

type MutexHandler struct{}

func NewMutexHandler() *MutexHandler {
	return &MutexHandler{}
}

func (h *MutexHandler) ListGroups(c echo.Context) error {
	ctx := context.Background()

	rows, err := db.Pool.Query(ctx, `
		SELECT mg.id, mg.name, mg.description, mg.created_at,
		       COALESCE(json_agg(json_build_object('id', pr.id, 'name', pr.name)) FILTER (WHERE pr.id IS NOT NULL), '[]') as rules
		FROM mutex_groups mg
		LEFT JOIN promo_mutex_relations pmr ON mg.id = pmr.group_id
		LEFT JOIN promo_rules pr ON pmr.rule_id = pr.id
		GROUP BY mg.id
		ORDER BY mg.created_at DESC
	`)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer rows.Close()

	var groups []map[string]interface{}
	for rows.Next() {
		var id int64
		var name, description string
		var createdAt time.Time
		var rulesJSON []byte

		err := rows.Scan(&id, &name, &description, &createdAt, &rulesJSON)
		if err != nil {
			continue
		}

		var rules interface{}
		json.Unmarshal(rulesJSON, &rules)

		groups = append(groups, map[string]interface{}{
			"id":          id,
			"name":        name,
			"description": description,
			"created_at":  createdAt,
			"rules":       rules,
		})
	}

	return c.JSON(http.StatusOK, groups)
}

func (h *MutexHandler) GetGroup(c echo.Context) error {
	ctx := context.Background()
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var name, description string
	var createdAt time.Time

	err := db.Pool.QueryRow(ctx, `
		SELECT name, description, created_at
		FROM mutex_groups WHERE id = $1
	`, id).Scan(&name, &description, &createdAt)

	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "group not found"})
	}

	rows, _ := db.Pool.Query(ctx, `
		SELECT pr.id, pr.name
		FROM promo_mutex_relations pmr
		JOIN promo_rules pr ON pmr.rule_id = pr.id
		WHERE pmr.group_id = $1
	`, id)
	defer rows.Close()

	var rules []map[string]interface{}
	for rows.Next() {
		var ruleID int64
		var ruleName string
		rows.Scan(&ruleID, &ruleName)
		rules = append(rules, map[string]interface{}{
			"id":   ruleID,
			"name": ruleName,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":          id,
		"name":        name,
		"description": description,
		"created_at":  createdAt,
		"rules":       rules,
	})
}

func (h *MutexHandler) CreateGroup(c echo.Context) error {
	ctx := context.Background()
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	var id int64
	var createdAt time.Time
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO mutex_groups (name, description)
		VALUES ($1, $2)
		RETURNING id, created_at
	`, req.Name, req.Description).Scan(&id, &createdAt)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"id":          id,
		"name":        req.Name,
		"description": req.Description,
		"created_at":  createdAt,
	})
}

func (h *MutexHandler) UpdateGroup(c echo.Context) error {
	ctx := context.Background()
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	_, err := db.Pool.Exec(ctx, `
		UPDATE mutex_groups
		SET name = $1, description = $2
		WHERE id = $3
	`, req.Name, req.Description, id)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

func (h *MutexHandler) DeleteGroup(c echo.Context) error {
	ctx := context.Background()
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	_, err := db.Pool.Exec(ctx, "DELETE FROM mutex_groups WHERE id = $1", id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *MutexHandler) AddRuleToGroup(c echo.Context) error {
	ctx := context.Background()
	groupId, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	ruleId, _ := strconv.ParseInt(c.Param("ruleId"), 10, 64)

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO promo_mutex_relations (group_id, rule_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, groupId, ruleId)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "added"})
}

func (h *MutexHandler) RemoveRuleFromGroup(c echo.Context) error {
	ctx := context.Background()
	groupId, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	ruleId, _ := strconv.ParseInt(c.Param("ruleId"), 10, 64)

	_, err := db.Pool.Exec(ctx, `
		DELETE FROM promo_mutex_relations
		WHERE group_id = $1 AND rule_id = $2
	`, groupId, ruleId)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}
