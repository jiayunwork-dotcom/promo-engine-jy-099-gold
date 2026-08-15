package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"promo-engine/internal/cache"
	"promo-engine/internal/db"
	"promo-engine/internal/engine"
	"promo-engine/internal/models"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

type PromoHandler struct{}

func NewPromoHandler() *PromoHandler {
	return &PromoHandler{}
}

type CalculatePriceRequest struct {
	models.Cart
	CouponCode string `json:"coupon_code"`
}

func (h *PromoHandler) CalculatePrice(c echo.Context) error {
	var req CalculatePriceRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	result, err := engine.Engine.Calculate(&req.Cart)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if req.CouponCode != "" {
		couponDiscount, _ := h.calculateCouponDiscount(&req.Cart, req.CouponCode, result)
		if couponDiscount > 0 {
			result.CouponDiscount = couponDiscount
			result.CouponCode = req.CouponCode
			result.TotalDiscount = round2(result.TotalDiscount + couponDiscount)
			result.TotalPay = round2(result.TotalPay - couponDiscount)
			if result.TotalPay < 0 {
				result.TotalPay = 0
			}
		}
	}

	return c.JSON(http.StatusOK, result)
}

func round2(value float64) float64 {
	return float64(int64(value*100+0.5)) / 100
}

func (h *PromoHandler) calculateCouponDiscount(cart *models.Cart, couponCode string, promoResult *models.CalculationResult) (float64, error) {
	ctx := context.Background()

	var batch models.CouponBatch
	var scopeJSON []byte
	var couponStatus string
	var couponUserID *string

	err := db.Pool.QueryRow(ctx, `
		SELECT cb.id, cb.name, cb.coupon_type, cb.discount_amount, cb.discount_rate,
		       cb.max_discount_amount, cb.threshold_amount, cb.scope,
		       cb.valid_from, cb.valid_to, c.status, c.user_id
		FROM coupons c
		JOIN coupon_batches cb ON c.batch_id = cb.id
		WHERE c.code = $1
	`, couponCode).Scan(
		&batch.ID, &batch.Name, &batch.CouponType, &batch.DiscountAmount,
		&batch.DiscountRate, &batch.MaxDiscountAmount, &batch.ThresholdAmount,
		&scopeJSON, &batch.ValidFrom, &batch.ValidTo, &couponStatus, &couponUserID,
	)
	if err != nil {
		return 0, err
	}

	json.Unmarshal(scopeJSON, &batch.Scope)

	now := time.Now()
	if now.Before(batch.ValidFrom) || now.After(batch.ValidTo) {
		return 0, nil
	}

	if couponStatus != string(models.CouponStatusClaimed) {
		return 0, nil
	}

	if couponUserID == nil || *couponUserID != cart.UserID {
		return 0, nil
	}

	scope := models.PromoScope{
		Type:        batch.Scope.Type,
		CategoryIDs: batch.Scope.CategoryIDs,
		StoreIDs:    batch.Scope.StoreIDs,
	}

	applicableTotal := 0.0
	for _, item := range promoResult.Items {
		if engine.Engine.IsItemInScope(&item, scope) {
			applicableTotal += item.PayPrice * float64(item.Quantity)
		}
	}

	if applicableTotal < batch.ThresholdAmount {
		return 0, nil
	}

	var discount float64
	switch batch.CouponType {
	case models.CouponTypeFullReduction:
		discount = batch.DiscountAmount
	case models.CouponTypeDiscount:
		discount = applicableTotal * (1 - batch.DiscountRate)
		if batch.MaxDiscountAmount > 0 && discount > batch.MaxDiscountAmount {
			discount = batch.MaxDiscountAmount
		}
	case models.CouponTypeNoThreshold:
		discount = batch.DiscountAmount
	}

	discount = round2(discount)

	if discount > applicableTotal {
		discount = applicableTotal
	}

	return discount, nil
}

func (h *PromoHandler) ListRules(c echo.Context) error {
	ctx := context.Background()
	status := c.QueryParam("status")
	promoType := c.QueryParam("type")

	query := `
		SELECT id, name, description, promo_type, status, version, 
		       config, scope, time_condition, usage_limit, priority, 
		       created_by, created_at, updated_at
		FROM promo_rules
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if status != "" {
		query += " AND status = $" + strconv.Itoa(argIndex)
		args = append(args, status)
		argIndex++
	}

	if promoType != "" {
		query += " AND promo_type = $" + strconv.Itoa(argIndex)
		args = append(args, promoType)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer rows.Close()

	var rules []models.PromoRule
	for rows.Next() {
		var rule models.PromoRule
		var configJSON, scopeJSON, timeCondJSON, usageLimitJSON []byte

		err := rows.Scan(
			&rule.ID, &rule.Name, &rule.Description, &rule.PromoType,
			&rule.Status, &rule.Version, &configJSON, &scopeJSON, &timeCondJSON,
			&usageLimitJSON, &rule.Priority, &rule.CreatedBy, &rule.CreatedAt, &rule.UpdatedAt,
		)
		if err != nil {
			continue
		}

		rule.Config = make(models.JSONB)
		json.Unmarshal(configJSON, &rule.Config)
		json.Unmarshal(scopeJSON, &rule.Scope)
		json.Unmarshal(timeCondJSON, &rule.TimeCondition)
		if usageLimitJSON != nil {
			json.Unmarshal(usageLimitJSON, &rule.UsageLimit)
		}

		rules = append(rules, rule)
	}

	return c.JSON(http.StatusOK, rules)
}

func (h *PromoHandler) GetRule(c echo.Context) error {
	ctx := context.Background()
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var rule models.PromoRule
	var scopeJSON, timeCondJSON, usageLimitJSON []byte

	query := `
		SELECT id, name, description, promo_type, status, version, 
		       config, scope, time_condition, usage_limit, priority, 
		       created_by, created_at, updated_at
		FROM promo_rules WHERE id = $1
	`

	err := db.Pool.QueryRow(ctx, query, id).Scan(
		&rule.ID, &rule.Name, &rule.Description, &rule.PromoType,
		&rule.Status, &rule.Version, &rule.Config, &scopeJSON, &timeCondJSON,
		&usageLimitJSON, &rule.Priority, &rule.CreatedBy, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "rule not found"})
	}

	json.Unmarshal(scopeJSON, &rule.Scope)
	json.Unmarshal(timeCondJSON, &rule.TimeCondition)
	if usageLimitJSON != nil {
		json.Unmarshal(usageLimitJSON, &rule.UsageLimit)
	}

	return c.JSON(http.StatusOK, rule)
}

func (h *PromoHandler) CreateRule(c echo.Context) error {
	ctx := context.Background()
	var rule models.PromoRule
	if err := c.Bind(&rule); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	conflictResult, err := engine.Engine.DetectConflicts(&rule)
	if err == nil && len(conflictResult.Errors) > 0 {
		return c.JSON(http.StatusBadRequest, conflictResult)
	}

	scopeJSON, _ := json.Marshal(rule.Scope)
	timeCondJSON, _ := json.Marshal(rule.TimeCondition)
	usageLimitJSON, _ := json.Marshal(rule.UsageLimit)
	configJSON, _ := json.Marshal(rule.Config)

	query := `
		INSERT INTO promo_rules (name, description, promo_type, status, version, config, scope, time_condition, usage_limit, priority, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`

	err = db.Pool.QueryRow(ctx, query,
		rule.Name, rule.Description, rule.PromoType, models.PromoStatusDraft, 1,
		configJSON, scopeJSON, timeCondJSON, usageLimitJSON, rule.Priority, rule.CreatedBy,
	).Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	h.saveRuleVersion(ctx, &rule)
	h.refreshCache()

	return c.JSON(http.StatusCreated, rule)
}

func (h *PromoHandler) UpdateRule(c echo.Context) error {
	ctx := context.Background()
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var rule models.PromoRule
	if err := c.Bind(&rule); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	rule.ID = id

	var currentVersion int
	db.Pool.QueryRow(ctx, "SELECT version FROM promo_rules WHERE id = $1", id).Scan(&currentVersion)
	rule.Version = currentVersion + 1

	scopeJSON, _ := json.Marshal(rule.Scope)
	timeCondJSON, _ := json.Marshal(rule.TimeCondition)
	usageLimitJSON, _ := json.Marshal(rule.UsageLimit)
	configJSON, _ := json.Marshal(rule.Config)

	query := `
		UPDATE promo_rules 
		SET name = $1, description = $2, promo_type = $3, config = $4, scope = $5, 
		    time_condition = $6, usage_limit = $7, priority = $8, version = $9, updated_at = CURRENT_TIMESTAMP
		WHERE id = $10
		RETURNING status, created_at, updated_at
	`

	err := db.Pool.QueryRow(ctx, query,
		rule.Name, rule.Description, rule.PromoType, configJSON, scopeJSON,
		timeCondJSON, usageLimitJSON, rule.Priority, rule.Version, id,
	).Scan(&rule.Status, &rule.CreatedAt, &rule.UpdatedAt)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	h.saveRuleVersion(ctx, &rule)
	h.refreshCache()

	return c.JSON(http.StatusOK, rule)
}

func (h *PromoHandler) DeleteRule(c echo.Context) error {
	ctx := context.Background()
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	_, err := db.Pool.Exec(ctx, "DELETE FROM promo_rules WHERE id = $1", id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	h.refreshCache()

	return c.NoContent(http.StatusNoContent)
}

func (h *PromoHandler) UpdateRuleStatus(c echo.Context) error {
	ctx := context.Background()
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req struct {
		Status models.PromoStatus `json:"status"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	_, err := db.Pool.Exec(ctx, "UPDATE promo_rules SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2", req.Status, id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	h.refreshCache()

	return c.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

func (h *PromoHandler) DetectConflicts(c echo.Context) error {
	var rule models.PromoRule
	if err := c.Bind(&rule); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	result, err := engine.Engine.DetectConflicts(&rule)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

func (h *PromoHandler) EstimateEffect(c echo.Context) error {
	var rule models.PromoRule
	if err := c.Bind(&rule); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	result, err := engine.Engine.EstimateEffect(&rule)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

func (h *PromoHandler) GetRuleVersions(c echo.Context) error {
	ctx := context.Background()
	ruleID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	query := `
		SELECT id, rule_id, version, name, config, scope, time_condition, usage_limit, priority, created_by, created_at
		FROM promo_rule_versions
		WHERE rule_id = $1
		ORDER BY version DESC
	`

	rows, err := db.Pool.Query(ctx, query, ruleID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer rows.Close()

	var versions []map[string]interface{}
	for rows.Next() {
		var version struct {
			ID            int64           `json:"id"`
			RuleID        int64           `json:"rule_id"`
			Version       int             `json:"version"`
			Name          string          `json:"name"`
			Config        json.RawMessage `json:"config"`
			Scope         json.RawMessage `json:"scope"`
			TimeCondition json.RawMessage `json:"time_condition"`
			UsageLimit    json.RawMessage `json:"usage_limit,omitempty"`
			Priority      int             `json:"priority"`
			CreatedBy     string          `json:"created_by"`
			CreatedAt     time.Time       `json:"created_at"`
		}

		err := rows.Scan(&version.ID, &version.RuleID, &version.Version, &version.Name,
			&version.Config, &version.Scope, &version.TimeCondition, &version.UsageLimit,
			&version.Priority, &version.CreatedBy, &version.CreatedAt)
		if err != nil {
			continue
		}

		versions = append(versions, map[string]interface{}{
			"id":              version.ID,
			"rule_id":         version.RuleID,
			"version":         version.Version,
			"name":            version.Name,
			"config":          version.Config,
			"scope":           version.Scope,
			"time_condition":  version.TimeCondition,
			"usage_limit":     version.UsageLimit,
			"priority":        version.Priority,
			"created_by":      version.CreatedBy,
			"created_at":      version.CreatedAt,
		})
	}

	return c.JSON(http.StatusOK, versions)
}

func (h *PromoHandler) RollbackVersion(c echo.Context) error {
	ctx := context.Background()
	ruleID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	version, _ := strconv.Atoi(c.Param("version"))

	query := `
		SELECT name, config, scope, time_condition, usage_limit, priority
		FROM promo_rule_versions
		WHERE rule_id = $1 AND version = $2
	`

	var name string
	var configJSON, scopeJSON, timeCondJSON, usageLimitJSON []byte
	var priority int

	err := db.Pool.QueryRow(ctx, query, ruleID, version).Scan(
		&name, &configJSON, &scopeJSON, &timeCondJSON, &usageLimitJSON, &priority,
	)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "version not found"})
	}

	var currentVersion int
	db.Pool.QueryRow(ctx, "SELECT version FROM promo_rules WHERE id = $1", ruleID).Scan(&currentVersion)

	updateQuery := `
		UPDATE promo_rules
		SET name = $1, config = $2, scope = $3, time_condition = $4, usage_limit = $5, priority = $6, version = $7, updated_at = CURRENT_TIMESTAMP
		WHERE id = $8
	`

	_, err = db.Pool.Exec(ctx, updateQuery, name, configJSON, scopeJSON, timeCondJSON, usageLimitJSON, priority, currentVersion+1, ruleID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	h.refreshCache()

	return c.JSON(http.StatusOK, map[string]string{"status": "rolled back"})
}

func (h *PromoHandler) saveRuleVersion(ctx context.Context, rule *models.PromoRule) {
	scopeJSON, _ := json.Marshal(rule.Scope)
	timeCondJSON, _ := json.Marshal(rule.TimeCondition)
	usageLimitJSON, _ := json.Marshal(rule.UsageLimit)
	configJSON, _ := json.Marshal(rule.Config)

	query := `
		INSERT INTO promo_rule_versions (rule_id, version, name, config, scope, time_condition, usage_limit, priority, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	db.Pool.Exec(ctx, query, rule.ID, rule.Version, rule.Name, configJSON, scopeJSON, timeCondJSON, usageLimitJSON, rule.Priority, rule.CreatedBy)
}

func (h *PromoHandler) refreshCache() {
	cache.Delete(context.Background(), engine.CacheKeyActiveRules)
	cache.Publish(context.Background(), engine.RuleChangeChannel, map[string]string{"action": "refresh"})
}
