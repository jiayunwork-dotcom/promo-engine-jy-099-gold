package handlers

import (
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"promo-engine/internal/db"
	"promo-engine/internal/engine"
	"promo-engine/internal/models"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

type CouponHandler struct{}

func NewCouponHandler() *CouponHandler {
	return &CouponHandler{}
}

const couponChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateCouponCode() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	code := make([]byte, 12)
	for i := range code {
		code[i] = couponChars[r.Intn(len(couponChars))]
	}
	return string(code)
}

func (h *CouponHandler) ListBatches(c echo.Context) error {
	ctx := context.Background()

	query := `
		SELECT id, name, description, coupon_type, discount_amount, discount_rate,
		       max_discount_amount, threshold_amount, scope, valid_from, valid_to,
		       total_quantity, claimed_quantity, used_quantity, per_user_limit,
		       created_by, created_at, updated_at
		FROM coupon_batches
		ORDER BY created_at DESC
	`

	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer rows.Close()

	var batches []models.CouponBatch
	for rows.Next() {
		var batch models.CouponBatch
		var scopeJSON []byte

		err := rows.Scan(
			&batch.ID, &batch.Name, &batch.Description, &batch.CouponType,
			&batch.DiscountAmount, &batch.DiscountRate, &batch.MaxDiscountAmount,
			&batch.ThresholdAmount, &scopeJSON, &batch.ValidFrom, &batch.ValidTo,
			&batch.TotalQuantity, &batch.ClaimedQuantity, &batch.UsedQuantity,
			&batch.PerUserLimit, &batch.CreatedBy, &batch.CreatedAt, &batch.UpdatedAt,
		)
		if err != nil {
			continue
		}

		json.Unmarshal(scopeJSON, &batch.Scope)
		batches = append(batches, batch)
	}

	return c.JSON(http.StatusOK, batches)
}

func (h *CouponHandler) GetBatch(c echo.Context) error {
	ctx := context.Background()
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var batch models.CouponBatch
	var scopeJSON []byte

	query := `
		SELECT id, name, description, coupon_type, discount_amount, discount_rate,
		       max_discount_amount, threshold_amount, scope, valid_from, valid_to,
		       total_quantity, claimed_quantity, used_quantity, per_user_limit,
		       created_by, created_at, updated_at
		FROM coupon_batches
		WHERE id = $1
	`

	err := db.Pool.QueryRow(ctx, query, id).Scan(
		&batch.ID, &batch.Name, &batch.Description, &batch.CouponType,
		&batch.DiscountAmount, &batch.DiscountRate, &batch.MaxDiscountAmount,
		&batch.ThresholdAmount, &scopeJSON, &batch.ValidFrom, &batch.ValidTo,
		&batch.TotalQuantity, &batch.ClaimedQuantity, &batch.UsedQuantity,
		&batch.PerUserLimit, &batch.CreatedBy, &batch.CreatedAt, &batch.UpdatedAt,
	)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "batch not found"})
	}

	json.Unmarshal(scopeJSON, &batch.Scope)

	return c.JSON(http.StatusOK, batch)
}

func (h *CouponHandler) CreateBatch(c echo.Context) error {
	ctx := context.Background()
	var batch models.CouponBatch
	if err := c.Bind(&batch); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if batch.TotalQuantity <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "total quantity must be positive"})
	}

	scopeJSON, _ := json.Marshal(batch.Scope)

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO coupon_batches (name, description, coupon_type, discount_amount,
			discount_rate, max_discount_amount, threshold_amount, scope, valid_from,
			valid_to, total_quantity, per_user_limit, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at, updated_at
	`

	err = tx.QueryRow(ctx, query,
		batch.Name, batch.Description, batch.CouponType, batch.DiscountAmount,
		batch.DiscountRate, batch.MaxDiscountAmount, batch.ThresholdAmount,
		scopeJSON, batch.ValidFrom, batch.ValidTo, batch.TotalQuantity,
		batch.PerUserLimit, batch.CreatedBy,
	).Scan(&batch.ID, &batch.CreatedAt, &batch.UpdatedAt)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	for i := 0; i < batch.TotalQuantity; i++ {
		code := generateCouponCode()
		_, err = tx.Exec(ctx,
			"INSERT INTO coupons (batch_id, code, status) VALUES ($1, $2, $3)",
			batch.ID, code, models.CouponStatusAvailable,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate coupon codes"})
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, batch)
}

func (h *CouponHandler) ListBatchCoupons(c echo.Context) error {
	ctx := context.Background()
	batchID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	status := c.QueryParam("status")

	query := `
		SELECT id, batch_id, code, user_id, status, order_id, claimed_at, used_at, created_at
		FROM coupons
		WHERE batch_id = $1
	`
	args := []interface{}{batchID}
	argIndex := 2

	if status != "" {
		query += " AND status = $" + strconv.Itoa(argIndex)
		args = append(args, status)
		argIndex++
	}

	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer rows.Close()

	var coupons []models.Coupon
	for rows.Next() {
		var coupon models.Coupon
		err := rows.Scan(
			&coupon.ID, &coupon.BatchID, &coupon.Code, &coupon.UserID,
			&coupon.Status, &coupon.OrderID, &coupon.ClaimedAt, &coupon.UsedAt,
			&coupon.CreatedAt,
		)
		if err != nil {
			continue
		}
		coupons = append(coupons, coupon)
	}

	return c.JSON(http.StatusOK, coupons)
}

func (h *CouponHandler) ClaimCoupon(c echo.Context) error {
	ctx := context.Background()

	var req struct {
		UserID   string `json:"user_id"`
		BatchID  int64  `json:"batch_id,omitempty"`
		CouponCode string `json:"coupon_code,omitempty"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.UserID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "user_id is required"})
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer tx.Rollback(ctx)

	var coupon models.Coupon
	var batch models.CouponBatch
	var scopeJSON []byte

	if req.CouponCode != "" {
		err = tx.QueryRow(ctx, `
			SELECT c.id, c.batch_id, c.code, c.status, cb.id, cb.name, cb.coupon_type,
			       cb.discount_amount, cb.discount_rate, cb.max_discount_amount,
			       cb.threshold_amount, cb.scope, cb.valid_from, cb.valid_to,
			       cb.total_quantity, cb.claimed_quantity, cb.per_user_limit
			FROM coupons c
			JOIN coupon_batches cb ON c.batch_id = cb.id
			WHERE c.code = $1 AND c.status = $2
			FOR UPDATE
		`, strings.ToUpper(req.CouponCode), models.CouponStatusAvailable).Scan(
			&coupon.ID, &coupon.BatchID, &coupon.Code, &coupon.Status,
			&batch.ID, &batch.Name, &batch.CouponType, &batch.DiscountAmount,
			&batch.DiscountRate, &batch.MaxDiscountAmount, &batch.ThresholdAmount,
			&scopeJSON, &batch.ValidFrom, &batch.ValidTo, &batch.TotalQuantity,
			&batch.ClaimedQuantity, &batch.PerUserLimit,
		)
	} else if req.BatchID > 0 {
		err = tx.QueryRow(ctx, `
			SELECT c.id, c.batch_id, c.code, c.status, cb.id, cb.name, cb.coupon_type,
			       cb.discount_amount, cb.discount_rate, cb.max_discount_amount,
			       cb.threshold_amount, cb.scope, cb.valid_from, cb.valid_to,
			       cb.total_quantity, cb.claimed_quantity, cb.per_user_limit
			FROM coupons c
			JOIN coupon_batches cb ON c.batch_id = cb.id
			WHERE c.batch_id = $1 AND c.status = $2
			ORDER BY c.id ASC
			LIMIT 1
			FOR UPDATE
		`, req.BatchID, models.CouponStatusAvailable).Scan(
			&coupon.ID, &coupon.BatchID, &coupon.Code, &coupon.Status,
			&batch.ID, &batch.Name, &batch.CouponType, &batch.DiscountAmount,
			&batch.DiscountRate, &batch.MaxDiscountAmount, &batch.ThresholdAmount,
			&scopeJSON, &batch.ValidFrom, &batch.ValidTo, &batch.TotalQuantity,
			&batch.ClaimedQuantity, &batch.PerUserLimit,
		)
	} else {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "batch_id or coupon_code is required"})
	}

	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "coupon not available"})
	}

	now := time.Now()
	if now.Before(batch.ValidFrom) || now.After(batch.ValidTo) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "coupon batch not in valid period"})
	}

	var userClaimedCount int
	tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM coupons
		WHERE batch_id = $1 AND user_id = $2 AND status != $3
	`, batch.ID, req.UserID, models.CouponStatusAvailable).Scan(&userClaimedCount)

	if userClaimedCount >= batch.PerUserLimit {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "user has reached claim limit"})
	}

	_, err = tx.Exec(ctx, `
		UPDATE coupons
		SET user_id = $1, status = $2, claimed_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`, req.UserID, models.CouponStatusClaimed, coupon.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	_, err = tx.Exec(ctx, `
		UPDATE coupon_batches
		SET claimed_quantity = claimed_quantity + 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, batch.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if err = tx.Commit(ctx); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	coupon.UserID = req.UserID
	coupon.Status = models.CouponStatusClaimed
	json.Unmarshal(scopeJSON, &batch.Scope)
	coupon.Batch = &batch

	return c.JSON(http.StatusOK, coupon)
}

func (h *CouponHandler) ListUserCoupons(c echo.Context) error {
	ctx := context.Background()
	userID := c.Param("user_id")
	status := c.QueryParam("status")

	query := `
		SELECT c.id, c.batch_id, c.code, c.user_id, c.status, c.order_id,
		       c.claimed_at, c.used_at, c.created_at,
		       cb.name, cb.coupon_type, cb.discount_amount, cb.discount_rate,
		       cb.max_discount_amount, cb.threshold_amount, cb.scope,
		       cb.valid_from, cb.valid_to
		FROM coupons c
		JOIN coupon_batches cb ON c.batch_id = cb.id
		WHERE c.user_id = $1
	`
	args := []interface{}{userID}
	argIndex := 2

	if status != "" {
		query += " AND c.status = $" + strconv.Itoa(argIndex)
		args = append(args, status)
		argIndex++
	}

	query += " ORDER BY c.claimed_at DESC"

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer rows.Close()

	var coupons []models.Coupon
	for rows.Next() {
		var coupon models.Coupon
		var batch models.CouponBatch
		var scopeJSON []byte

		err := rows.Scan(
			&coupon.ID, &coupon.BatchID, &coupon.Code, &coupon.UserID,
			&coupon.Status, &coupon.OrderID, &coupon.ClaimedAt, &coupon.UsedAt,
			&coupon.CreatedAt, &batch.Name, &batch.CouponType, &batch.DiscountAmount,
			&batch.DiscountRate, &batch.MaxDiscountAmount, &batch.ThresholdAmount,
			&scopeJSON, &batch.ValidFrom, &batch.ValidTo,
		)
		if err != nil {
			continue
		}

		json.Unmarshal(scopeJSON, &batch.Scope)
		coupon.Batch = &batch
		coupons = append(coupons, coupon)
	}

	return c.JSON(http.StatusOK, coupons)
}

func (h *CouponHandler) ValidateAndCalculateCoupon(c echo.Context, cart *models.Cart, couponCode string) (float64, *models.CouponBatch, error) {
	ctx := context.Background()

	if couponCode == "" {
		return 0, nil, nil
	}

	var batch models.CouponBatch
	var scopeJSON []byte

	err := db.Pool.QueryRow(ctx, `
		SELECT cb.id, cb.name, cb.coupon_type, cb.discount_amount, cb.discount_rate,
		       cb.max_discount_amount, cb.threshold_amount, cb.scope,
		       cb.valid_from, cb.valid_to, c.status
		FROM coupons c
		JOIN coupon_batches cb ON c.batch_id = cb.id
		WHERE c.code = $1 AND c.user_id = $2
	`, strings.ToUpper(couponCode), cart.UserID).Scan(
		&batch.ID, &batch.Name, &batch.CouponType, &batch.DiscountAmount,
		&batch.DiscountRate, &batch.MaxDiscountAmount, &batch.ThresholdAmount,
		&scopeJSON, &batch.ValidFrom, &batch.ValidTo, &batch.CouponType,
	)
	if err != nil {
		return 0, nil, nil
	}

	json.Unmarshal(scopeJSON, &batch.Scope)

	now := time.Now()
	if now.Before(batch.ValidFrom) || now.After(batch.ValidTo) {
		return 0, nil, nil
	}

	promoResult, err := engine.Engine.Calculate(cart)
	if err != nil {
		return 0, nil, nil
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
		return 0, nil, nil
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

	if discount > applicableTotal {
		discount = applicableTotal
	}

	return discount, &batch, nil
}

func (h *CouponHandler) UseCoupon(c echo.Context) error {
	ctx := context.Background()

	var req struct {
		CouponCode string `json:"coupon_code"`
		UserID     string `json:"user_id"`
		OrderID    string `json:"order_id"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer tx.Rollback(ctx)

	var batchID int64
	err = tx.QueryRow(ctx, `
		UPDATE coupons
		SET status = $1, order_id = $2, used_at = CURRENT_TIMESTAMP
		WHERE code = $3 AND user_id = $4 AND status = $5
		RETURNING batch_id
	`, models.CouponStatusUsed, req.OrderID, strings.ToUpper(req.CouponCode), req.UserID, models.CouponStatusClaimed).Scan(&batchID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "coupon not available"})
	}

	_, err = tx.Exec(ctx, `
		UPDATE coupon_batches
		SET used_quantity = used_quantity + 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, batchID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if err = tx.Commit(ctx); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "coupon used"})
}

func (h *CouponHandler) ReturnCoupon(c echo.Context) error {
	ctx := context.Background()

	var req struct {
		CouponCode string `json:"coupon_code"`
		OrderID    string `json:"order_id"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer tx.Rollback(ctx)

	var batchID int64
	err = tx.QueryRow(ctx, `
		UPDATE coupons
		SET status = $1, order_id = NULL, used_at = NULL
		WHERE code = $2 AND order_id = $3 AND status = $4
		RETURNING batch_id
	`, models.CouponStatusClaimed, strings.ToUpper(req.CouponCode), req.OrderID, models.CouponStatusUsed).Scan(&batchID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "coupon not found or already returned"})
	}

	_, err = tx.Exec(ctx, `
		UPDATE coupon_batches
		SET used_quantity = used_quantity - 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, batchID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if err = tx.Commit(ctx); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "coupon returned"})
}
