package engine

import (
	"context"
	"encoding/json"
	"promo-engine/internal/db"
	"promo-engine/internal/models"
	"time"
)

func (e *PromoEngine) DetectConflicts(newRule *models.PromoRule) (*models.ConflictDetectionResult, error) {
	result := &models.ConflictDetectionResult{
		Warnings: []string{},
		Errors:   []string{},
	}

	ctx := context.Background()
	query := `
		SELECT id, name, promo_type, config, scope, time_condition
		FROM promo_rules
		WHERE status IN ($1, $2) AND id != $3
	`
	rows, err := db.Pool.Query(ctx, query, models.PromoStatusActive, models.PromoStatusDraft, newRule.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var existingRules []models.PromoRule
	for rows.Next() {
		var rule models.PromoRule
		var configJSON, scopeJSON, timeCondJSON []byte
		
		err := rows.Scan(&rule.ID, &rule.Name, &rule.PromoType, &configJSON, &scopeJSON, &timeCondJSON)
		if err != nil {
			continue
		}
		
		json.Unmarshal(scopeJSON, &rule.Scope)
		json.Unmarshal(timeCondJSON, &rule.TimeCondition)
		rule.Config = make(map[string]interface{})
		json.Unmarshal(configJSON, &rule.Config)
		
		existingRules = append(existingRules, rule)
	}

	for _, existingRule := range existingRules {
		if e.hasTimeOverlap(&existingRule.TimeCondition, &newRule.TimeCondition) {
			if e.hasScopeOverlap(&existingRule.Scope, &newRule.Scope) {
				if existingRule.PromoType == newRule.PromoType {
					result.Warnings = append(result.Warnings, 
						"与规则 "+existingRule.Name+" 存在时间和范围重叠，建议合并")
					result.HasConflict = true
					result.ConflictingRuleIDs = append(result.ConflictingRuleIDs, existingRule.ID)
				}
			}
		}
	}

	if e.willCausePriceBelowCost(newRule) {
		result.Errors = append(result.Errors, "该规则可能导致商品价格低于成本价，请调整")
		result.HasConflict = true
	}

	return result, nil
}

func (e *PromoEngine) hasTimeOverlap(t1, t2 *models.TimeCondition) bool {
	if t1.EndTime.Before(t2.StartTime) || t1.EndTime.Equal(t2.StartTime) || t1.StartTime.After(t2.EndTime) || t1.StartTime.Equal(t2.EndTime) {
		return false
	}

	if len(t1.Weekdays) > 0 && len(t2.Weekdays) > 0 {
		hasOverlap := false
		for _, wd1 := range t1.Weekdays {
			for _, wd2 := range t2.Weekdays {
				if wd1 == wd2 {
					hasOverlap = true
					break
				}
			}
		}
		if !hasOverlap {
			return false
		}
	}

	if len(t1.TimeRanges) > 0 && len(t2.TimeRanges) > 0 {
		hasOverlap := false
		for _, tr1 := range t1.TimeRanges {
			for _, tr2 := range t2.TimeRanges {
				if !(tr1.End <= tr2.Start || tr1.Start >= tr2.End) {
					hasOverlap = true
					break
				}
			}
		}
		if !hasOverlap {
			return false
		}
	}

	return true
}

func (e *PromoEngine) hasScopeOverlap(s1, s2 *models.PromoScope) bool {
	if s1.Type == "all" {
		return true
	}

	if s1.Type == s2.Type {
		switch s1.Type {
		case "category":
			for _, c1 := range s1.CategoryIDs {
				for _, c2 := range s2.CategoryIDs {
					if c1 == c2 {
						return true
					}
				}
			}
		case "store":
			for _, st1 := range s1.StoreIDs {
				for _, st2 := range s2.StoreIDs {
					if st1 == st2 {
						return true
					}
				}
			}
		case "sku":
			for _, sku1 := range s1.SKUIds {
				for _, sku2 := range s2.SKUIds {
					if sku1 == sku2 {
						return true
					}
				}
			}
		}
	}

	return false
}

func (e *PromoEngine) willCausePriceBelowCost(rule *models.PromoRule) bool {
	if rule.PromoType != models.PromoTypeDiscount {
		return false
	}

	var config models.DiscountConfig
	configBytes, _ := json.Marshal(rule.Config)
	json.Unmarshal(configBytes, &config)

	skuIDs := e.getScopeSKUIds(&rule.Scope)

	for _, skuID := range skuIDs {
		ctx := context.Background()
		var price, costPrice float64
		err := db.Pool.QueryRow(ctx, "SELECT price, cost_price FROM skus WHERE id = $1", skuID).Scan(&price, &costPrice)
		if err != nil {
			continue
		}

		discountedPrice := price * config.DiscountRate
		if discountedPrice < costPrice {
			return true
		}
	}

	return false
}

func (e *PromoEngine) getScopeSKUIds(scope *models.PromoScope) []int64 {
	ctx := context.Background()
	var skuIDs []int64

	switch scope.Type {
	case "all":
		rows, _ := db.Pool.Query(ctx, "SELECT id FROM skus")
		defer rows.Close()
		for rows.Next() {
			var id int64
			rows.Scan(&id)
			skuIDs = append(skuIDs, id)
		}
	case "category":
		for _, catID := range scope.CategoryIDs {
			rows, _ := db.Pool.Query(ctx, "SELECT id FROM skus WHERE category_id = $1", catID)
			defer rows.Close()
			for rows.Next() {
				var id int64
				rows.Scan(&id)
				skuIDs = append(skuIDs, id)
			}
		}
	case "store":
		for _, storeID := range scope.StoreIDs {
			rows, _ := db.Pool.Query(ctx, "SELECT id FROM skus WHERE store_id = $1", storeID)
			defer rows.Close()
			for rows.Next() {
				var id int64
				rows.Scan(&id)
				skuIDs = append(skuIDs, id)
			}
		}
	case "sku":
		skuIDs = scope.SKUIds
	}

	return skuIDs
}

func (e *PromoEngine) EstimateEffect(rule *models.PromoRule) (*models.EstimationResult, error) {
	result := &models.EstimationResult{}

	ctx := context.Background()
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	rows, err := db.Pool.Query(ctx, `
		SELECT o.id, o.total_amount, o.discount_amount
		FROM orders o
		WHERE o.created_at >= $1 AND o.status = 'completed'
	`, thirtyDaysAgo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type OrderData struct {
		ID              int64
		TotalAmount     float64
		DiscountAmount  float64
		Items           []models.CartItem
	}

	var orders []OrderData
	for rows.Next() {
		var order OrderData
		err := rows.Scan(&order.ID, &order.TotalAmount, &order.DiscountAmount)
		if err != nil {
			continue
		}

		itemRows, _ := db.Pool.Query(ctx, `
			SELECT sku_id, store_id, original_price, quantity
			FROM order_items
			WHERE order_id = $1
		`, order.ID)
		
		for itemRows.Next() {
			var item models.CartItem
			itemRows.Scan(&item.SKUId, &item.StoreID, &item.Price, &item.Quantity)
			order.Items = append(order.Items, item)
		}
		itemRows.Close()

		orders = append(orders, order)
	}

	for _, order := range orders {
		cart := &models.Cart{
			Items: order.Items,
		}

		newDiscount := e.calculateRuleDiscount(rule, e.initCalculationResult(cart))

		if newDiscount > 0 {
			result.EstimatedOrders++
			result.EstimatedDiscount += newDiscount
			result.EstimatedGMVChange += (order.TotalAmount - newDiscount) - (order.TotalAmount - order.DiscountAmount)
		}
	}

	return result, nil
}
