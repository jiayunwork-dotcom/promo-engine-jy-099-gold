package engine

import (
	"context"
	"encoding/json"
	"math"
	"promo-engine/internal/db"
	"promo-engine/internal/models"
	"sort"
	"time"
)

func (e *PromoEngine) calculateWithGreedy(cart *models.Cart, rules []*models.PromoRule) (*models.CalculationResult, error) {
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})

	result := e.initCalculationResult(cart)
	appliedRules := make(map[int64]bool)

	for _, rule := range rules {
		if appliedRules[rule.ID] {
			continue
		}
		if e.isMutexWithApplied(rule, appliedRules) {
			continue
		}
		
		discount := e.calculateRuleDiscount(rule, result)
		if discount > 0 {
			e.applyRule(rule, result)
			appliedRules[rule.ID] = true
		}
	}

	e.finalizeResult(result)
	return result, nil
}

func (e *PromoEngine) calculateWithDP(cart *models.Cart, rules []*models.PromoRule) (*models.CalculationResult, error) {
	groups := e.groupMutexRules(rules)
	
	result := e.initCalculationResult(cart)
	appliedRules := make(map[int64]bool)

	for _, group := range groups {
		if len(group) == 0 {
			continue
		}
		
		bestRule := e.findBestRuleInGroup(group, result)
		if bestRule != nil {
			e.applyRule(bestRule, result)
			appliedRules[bestRule.ID] = true
		}
	}

	e.finalizeResult(result)
	return result, nil
}

func (e *PromoEngine) calculateWithBranchBound(cart *models.Cart, rules []*models.PromoRule) (*models.CalculationResult, error) {
	timeout := time.After(50 * time.Millisecond)
	
	bestResult := e.initCalculationResult(cart)
	bestDiscount := 0.0
	
	type state struct {
		ruleIndex  int
		applied    map[int64]bool
		result     *models.CalculationResult
	}
	
	initialResult := e.initCalculationResult(cart)
	stack := []state{{ruleIndex: 0, applied: make(map[int64]bool), result: initialResult}}
	
	for len(stack) > 0 {
		select {
		case <-timeout:
			e.finalizeResult(bestResult)
			return bestResult, nil
		default:
		}
		
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		
		if current.ruleIndex >= len(rules) {
			totalDiscount := e.calculateTotalDiscount(current.result)
			if totalDiscount > bestDiscount {
				bestDiscount = totalDiscount
				bestResult = current.result
			}
			continue
		}
		
		rule := rules[current.ruleIndex]
		
		stack = append(stack, state{
			ruleIndex: current.ruleIndex + 1,
			applied:   copyMap(current.applied),
			result:    copyResult(current.result),
		})
		
		if !e.isMutexWithApplied(rule, current.applied) {
			newResult := copyResult(current.result)
			newApplied := copyMap(current.applied)
			
			discount := e.calculateRuleDiscount(rule, newResult)
			if discount > 0 {
				e.applyRule(rule, newResult)
				newApplied[rule.ID] = true
			}
			
			stack = append(stack, state{
				ruleIndex: current.ruleIndex + 1,
				applied:   newApplied,
				result:    newResult,
			})
		}
	}
	
	e.finalizeResult(bestResult)
	return bestResult, nil
}

func (e *PromoEngine) initCalculationResult(cart *models.Cart) *models.CalculationResult {
	result := &models.CalculationResult{
		Items: make([]models.CalculatedItem, len(cart.Items)),
	}
	
	for i, item := range cart.Items {
		result.Items[i] = models.CalculatedItem{
			SKUId:          item.SKUId,
			SKUName:        item.SKUName,
			StoreID:        item.StoreID,
			OriginalPrice:  item.Price,
			DiscountAmount: 0,
			PayPrice:       item.Price,
			Quantity:       item.Quantity,
			PromoRuleIDs:   []int64{},
		}
		result.TotalOriginal += item.Price * float64(item.Quantity)
	}
	result.TotalPay = result.TotalOriginal
	
	return result
}

func (e *PromoEngine) calculateRuleDiscount(rule *models.PromoRule, result *models.CalculationResult) float64 {
	switch rule.PromoType {
	case models.PromoTypeFullReduction:
		return e.calculateFullReductionDiscount(rule, result)
	case models.PromoTypeDiscount:
		return e.calculateDiscountDiscount(rule, result)
	case models.PromoTypeNthItem:
		return e.calculateNthItemDiscount(rule, result)
	case models.PromoTypeCrossStore:
		return e.calculateCrossStoreDiscount(rule, result)
	case models.PromoTypeCombo:
		return e.calculateComboDiscount(rule, result)
	default:
		return 0
	}
}

func (e *PromoEngine) calculateFullReductionDiscount(rule *models.PromoRule, result *models.CalculationResult) float64 {
	var config models.FullReductionConfig
	
	tiersData, ok := rule.Config["tiers"]
	if !ok {
		return 0
	}
	
	configBytes, _ := json.Marshal(tiersData)
	json.Unmarshal(configBytes, &config.Tiers)
	
	if len(config.Tiers) == 0 {
		return 0
	}
	
	total := e.calculateSubtotalForScope(rule.Scope, result)
	maxDiscount := 0.0
	
	for _, tier := range config.Tiers {
		if total > tier.Threshold && tier.Discount > maxDiscount {
			maxDiscount = tier.Discount
		}
	}
	
	return maxDiscount
}

func (e *PromoEngine) calculateDiscountDiscount(rule *models.PromoRule, result *models.CalculationResult) float64 {
	discountRate, ok := rule.Config["discount_rate"].(float64)
	if !ok {
		discountRate = 0
	}
	
	totalDiscount := 0.0
	for i := range result.Items {
		if e.isItemInScope(&result.Items[i], rule.Scope) {
			currentPrice := result.Items[i].PayPrice
			newPrice := currentPrice * discountRate
			discount := currentPrice - newPrice
			totalDiscount += discount * float64(result.Items[i].Quantity)
		}
	}
	
	return totalDiscount
}

func (e *PromoEngine) calculateNthItemDiscount(rule *models.PromoRule, result *models.CalculationResult) float64 {
	var config models.NthItemConfig
	configBytes, _ := json.Marshal(rule.Config)
	json.Unmarshal(configBytes, &config)
	
	totalDiscount := 0.0
	itemGroups := make(map[int64][]models.CalculatedItem)
	
	for _, item := range result.Items {
		if e.isItemInScope(&item, rule.Scope) {
			itemGroups[item.SKUId] = append(itemGroups[item.SKUId], item)
		}
	}
	
	for _, items := range itemGroups {
		if len(items) == 0 {
			continue
		}
		
		sort.Slice(items, func(i, j int) bool {
			return items[i].PayPrice < items[j].PayPrice
		})
		
		for _, item := range items {
			for q := 1; q <= item.Quantity; q++ {
				if q%config.NthItem == 1 {
					if config.Free {
						totalDiscount += item.PayPrice
					} else {
						totalDiscount += item.PayPrice * (1 - config.DiscountRate)
					}
				}
			}
		}
	}
	
	return totalDiscount
}

func (e *PromoEngine) calculateCrossStoreDiscount(rule *models.PromoRule, result *models.CalculationResult) float64 {
	var config models.CrossStoreConfig
	configBytes, _ := json.Marshal(rule.Config)
	json.Unmarshal(configBytes, &config)
	
	storeTotals := make(map[int64]float64)
	for _, item := range result.Items {
		storeTotals[item.StoreID] += item.PayPrice * float64(item.Quantity)
	}
	
	crossTotal := 0.0
	for _, storeID := range rule.Scope.StoreIDs {
		crossTotal += storeTotals[storeID]
	}
	
	if crossTotal >= config.Threshold {
		return config.Discount
	}
	
	return 0
}

func (e *PromoEngine) calculateComboDiscount(rule *models.PromoRule, result *models.CalculationResult) float64 {
	var config models.ComboConfig
	configBytes, _ := json.Marshal(rule.Config)
	json.Unmarshal(configBytes, &config)
	
	foundItems := make(map[int64]bool)
	minQuantity := math.MaxInt32
	
	for _, skuID := range config.SKUIds {
		for _, item := range result.Items {
			if item.SKUId == skuID {
				foundItems[skuID] = true
				if item.Quantity < minQuantity {
					minQuantity = item.Quantity
				}
			}
		}
	}
	
	if len(foundItems) != len(config.SKUIds) || minQuantity == 0 {
		return 0
	}
	
	originalTotal := 0.0
	for _, skuID := range config.SKUIds {
		for _, item := range result.Items {
			if item.SKUId == skuID {
				originalTotal += item.PayPrice * float64(minQuantity)
				break
			}
		}
	}
	
	return originalTotal - config.ComboPrice*float64(minQuantity)
}

func (e *PromoEngine) applyRule(rule *models.PromoRule, result *models.CalculationResult) {
	switch rule.PromoType {
	case models.PromoTypeFullReduction:
		e.applyFullReduction(rule, result)
	case models.PromoTypeDiscount:
		e.applyDiscount(rule, result)
	case models.PromoTypeBuyGift:
		e.applyBuyGift(rule, result)
	case models.PromoTypeNthItem:
		e.applyNthItem(rule, result)
	case models.PromoTypeCrossStore:
		e.applyCrossStore(rule, result)
	case models.PromoTypeCombo:
		e.applyCombo(rule, result)
	}
}

func (e *PromoEngine) applyFullReduction(rule *models.PromoRule, result *models.CalculationResult) {
	var config models.FullReductionConfig
	
	tiersData, ok := rule.Config["tiers"]
	if !ok {
		return
	}
	
	configBytes, _ := json.Marshal(tiersData)
	json.Unmarshal(configBytes, &config.Tiers)
	
	total := e.calculateSubtotalForScope(rule.Scope, result)
	maxDiscount := 0.0
	
	for _, tier := range config.Tiers {
		if total >= tier.Threshold && tier.Discount > maxDiscount {
			maxDiscount = tier.Discount
		}
	}
	
	if maxDiscount <= 0 {
		return
	}
	
	e.allocateDiscount(result, rule.Scope, maxDiscount, rule.ID)
}

func (e *PromoEngine) applyDiscount(rule *models.PromoRule, result *models.CalculationResult) {
	discountRate, ok := rule.Config["discount_rate"].(float64)
	if !ok {
		return
	}

	for i := range result.Items {
		if e.isItemInScope(&result.Items[i], rule.Scope) {
			currentPrice := result.Items[i].PayPrice
			newPrice := currentPrice * discountRate
			discount := currentPrice - newPrice

			result.Items[i].DiscountAmount += discount
			result.Items[i].PayPrice = newPrice
			result.Items[i].PromoRuleIDs = append(result.Items[i].PromoRuleIDs, rule.ID)

			result.TotalDiscount += discount * float64(result.Items[i].Quantity)
			result.TotalPay -= discount * float64(result.Items[i].Quantity)
		}
	}
}

func (e *PromoEngine) applyBuyGift(rule *models.PromoRule, result *models.CalculationResult) {
	var config models.BuyGiftConfig
	configBytes, _ := json.Marshal(rule.Config)
	json.Unmarshal(configBytes, &config)
	
	buyCount := 0
	for _, item := range result.Items {
		if item.SKUId == config.BuySKUId {
			buyCount += item.Quantity
		}
	}
	
	giftCount := (buyCount / config.BuyQuantity) * config.GiftQuantity
	if giftCount <= 0 {
		return
	}
	
	if !e.checkGiftStock(config.GiftSKUId, giftCount) {
		return
	}
	
	ctx := context.Background()
	var giftSKU models.SKU
	db.Pool.QueryRow(ctx, "SELECT id, name, price FROM skus WHERE id = $1", config.GiftSKUId).Scan(
		&giftSKU.ID, &giftSKU.Name, &giftSKU.Price,
	)
	
	giftItem := models.CalculatedItem{
		SKUId:          giftSKU.ID,
		SKUName:        giftSKU.Name,
		OriginalPrice:  giftSKU.Price,
		DiscountAmount: giftSKU.Price,
		PayPrice:       0,
		Quantity:       giftCount,
		PromoRuleIDs:   []int64{rule.ID},
	}
	
	result.GiftItems = append(result.GiftItems, giftItem)
}

func (e *PromoEngine) applyNthItem(rule *models.PromoRule, result *models.CalculationResult) {
	var config models.NthItemConfig
	configBytes, _ := json.Marshal(rule.Config)
	json.Unmarshal(configBytes, &config)
	
	itemGroups := make(map[int64][]*models.CalculatedItem)
	
	for i := range result.Items {
		if e.isItemInScope(&result.Items[i], rule.Scope) {
			itemGroups[result.Items[i].SKUId] = append(itemGroups[result.Items[i].SKUId], &result.Items[i])
		}
	}
	
	for _, items := range itemGroups {
		if len(items) == 0 {
			continue
		}
		
		sort.Slice(items, func(i, j int) bool {
			return items[i].PayPrice < items[j].PayPrice
		})
		
		for _, item := range items {
			for q := 1; q <= item.Quantity; q++ {
				if q%config.NthItem == 0 {
					var discount float64
					if config.Free {
						discount = item.PayPrice / float64(item.Quantity)
					} else {
						discount = item.PayPrice * (1 - config.DiscountRate) / float64(item.Quantity)
					}
					
					item.DiscountAmount += discount
					item.PayPrice -= discount
					item.PromoRuleIDs = append(item.PromoRuleIDs, rule.ID)
					
					result.TotalDiscount += discount
					result.TotalPay -= discount
				}
			}
		}
	}
}

func (e *PromoEngine) applyCrossStore(rule *models.PromoRule, result *models.CalculationResult) {
	var config models.CrossStoreConfig
	configBytes, _ := json.Marshal(rule.Config)
	json.Unmarshal(configBytes, &config)
	
	storeTotals := make(map[int64]float64)
	for _, item := range result.Items {
		storeTotals[item.StoreID] += item.PayPrice * float64(item.Quantity)
	}
	
	crossTotal := 0.0
	for _, storeID := range rule.Scope.StoreIDs {
		crossTotal += storeTotals[storeID]
	}
	
	if crossTotal < config.Threshold {
		return
	}
	
	crossScope := models.PromoScope{
		Type:     "store",
		StoreIDs: rule.Scope.StoreIDs,
	}
	e.allocateDiscount(result, crossScope, config.Discount, rule.ID)
}

func (e *PromoEngine) applyCombo(rule *models.PromoRule, result *models.CalculationResult) {
	var config models.ComboConfig
	configBytes, _ := json.Marshal(rule.Config)
	json.Unmarshal(configBytes, &config)
	
	minQuantity := math.MaxInt32
	for _, skuID := range config.SKUIds {
		for _, item := range result.Items {
			if item.SKUId == skuID {
				if item.Quantity < minQuantity {
					minQuantity = item.Quantity
				}
			}
		}
	}
	
	originalTotal := 0.0
	for _, skuID := range config.SKUIds {
		for _, item := range result.Items {
			if item.SKUId == skuID {
				originalTotal += item.PayPrice * float64(minQuantity)
			}
		}
	}
	
	totalDiscount := originalTotal - config.ComboPrice*float64(minQuantity)
	if totalDiscount <= 0 {
		return
	}
	
	comboScope := models.PromoScope{
		Type:    "sku",
		SKUIds: config.SKUIds,
	}
	e.allocateDiscount(result, comboScope, totalDiscount, rule.ID)
}

func (e *PromoEngine) allocateDiscount(result *models.CalculationResult, scope models.PromoScope, totalDiscount float64, ruleID int64) {
	if totalDiscount <= 0 {
		return
	}
	
	var applicableItems []*models.CalculatedItem
	applicableTotal := 0.0
	
	for i := range result.Items {
		if e.isItemInScope(&result.Items[i], scope) {
			itemTotal := result.Items[i].PayPrice * float64(result.Items[i].Quantity)
			applicableItems = append(applicableItems, &result.Items[i])
			applicableTotal += itemTotal
		}
	}
	
	if applicableTotal <= 0 {
		return
	}
	
	remainingDiscount := totalDiscount
	for i, item := range applicableItems {
		itemTotal := item.PayPrice * float64(item.Quantity)
		var allocated float64
		
		if i == len(applicableItems)-1 {
			allocated = remainingDiscount
		} else {
			ratio := itemTotal / applicableTotal
			allocated = math.Round(totalDiscount*ratio*100) / 100
			remainingDiscount -= allocated
		}
		
		perItemDiscount := allocated / float64(item.Quantity)
		item.DiscountAmount += perItemDiscount
		item.PayPrice -= perItemDiscount
		item.PromoRuleIDs = append(item.PromoRuleIDs, ruleID)
	}
	
	result.TotalDiscount += totalDiscount
	result.TotalPay -= totalDiscount
}

func (e *PromoEngine) calculateSubtotalForScope(scope models.PromoScope, result *models.CalculationResult) float64 {
	total := 0.0
	for _, item := range result.Items {
		if e.isItemInScope(&item, scope) {
			total += item.PayPrice * float64(item.Quantity)
		}
	}
	return total
}

func (e *PromoEngine) IsItemInScope(item *models.CalculatedItem, scope models.PromoScope) bool {
	switch scope.Type {
	case "all":
		return true
	case "category":
		for _, catID := range scope.CategoryIDs {
			if e.isSKUInCategory(item.SKUId, catID) {
				return true
			}
		}
	case "store":
		for _, storeID := range scope.StoreIDs {
			if item.StoreID == storeID {
				return true
			}
		}
	case "sku":
		for _, skuID := range scope.SKUIds {
			if item.SKUId == skuID {
				return true
			}
		}
	}
	return false
}

func (e *PromoEngine) isItemInScope(item *models.CalculatedItem, scope models.PromoScope) bool {
	return e.IsItemInScope(item, scope)
}

func (e *PromoEngine) isMutexWithApplied(rule *models.PromoRule, applied map[int64]bool) bool {
	ctx := context.Background()
	rows, _ := db.Pool.Query(ctx, `
		SELECT DISTINCT pr.rule_id
		FROM promo_mutex_relations pr
		JOIN promo_mutex_relations pr2 ON pr.group_id = pr2.group_id
		WHERE pr2.rule_id = $1
	`, rule.ID)
	defer rows.Close()
	
	for rows.Next() {
		var mutexRuleID int64
		rows.Scan(&mutexRuleID)
		if applied[mutexRuleID] {
			return true
		}
	}
	
	return false
}

func (e *PromoEngine) groupMutexRules(rules []*models.PromoRule) [][]*models.PromoRule {
	ruleGroups := make(map[int64]int)
	groupID := 0
	
	for _, rule := range rules {
		if _, exists := ruleGroups[rule.ID]; !exists {
			groupMutexRules := e.getMutexGroupRules(rule.ID, rules)
			for _, r := range groupMutexRules {
				ruleGroups[r.ID] = groupID
			}
			groupID++
		}
	}
	
	groups := make([][]*models.PromoRule, groupID)
	for _, rule := range rules {
		groups[ruleGroups[rule.ID]] = append(groups[ruleGroups[rule.ID]], rule)
	}
	
	return groups
}

func (e *PromoEngine) getMutexGroupRules(ruleID int64, allRules []*models.PromoRule) []*models.PromoRule {
	ctx := context.Background()
	rows, _ := db.Pool.Query(ctx, `
		SELECT DISTINCT pr2.rule_id
		FROM promo_mutex_relations pr
		JOIN promo_mutex_relations pr2 ON pr.group_id = pr2.group_id
		WHERE pr.rule_id = $1
	`, ruleID)
	defer rows.Close()
	
	mutexRuleIDs := make(map[int64]bool)
	mutexRuleIDs[ruleID] = true
	
	for rows.Next() {
		var id int64
		rows.Scan(&id)
		mutexRuleIDs[id] = true
	}
	
	var result []*models.PromoRule
	for _, rule := range allRules {
		if mutexRuleIDs[rule.ID] {
			result = append(result, rule)
		}
	}
	
	if len(result) == 0 {
		for _, rule := range allRules {
			if rule.ID == ruleID {
				result = append(result, rule)
				break
			}
		}
	}
	
	return result
}

func (e *PromoEngine) findBestRuleInGroup(group []*models.PromoRule, result *models.CalculationResult) *models.PromoRule {
	bestRule := (*models.PromoRule)(nil)
	bestDiscount := 0.0
	
	for _, rule := range group {
		discount := e.calculateRuleDiscount(rule, result)
		if discount > bestDiscount {
			bestDiscount = discount
			bestRule = rule
		}
	}
	
	return bestRule
}

func (e *PromoEngine) calculateTotalDiscount(result *models.CalculationResult) float64 {
	return result.TotalDiscount
}

func (e *PromoEngine) finalizeResult(result *models.CalculationResult) {
	ctx := context.Background()
	for i := range result.Items {
		var costPrice float64
		err := db.Pool.QueryRow(ctx, "SELECT cost_price FROM skus WHERE id = $1", result.Items[i].SKUId).Scan(&costPrice)
		if err == nil {
			if result.Items[i].PayPrice < costPrice {
				diff := costPrice - result.Items[i].PayPrice
				result.Items[i].PayPrice = costPrice
				result.Items[i].DiscountAmount -= diff
				result.TotalDiscount -= diff * float64(result.Items[i].Quantity)
				result.TotalPay += diff * float64(result.Items[i].Quantity)
			}
		}
	}
	
	result.TotalOriginal = math.Round(result.TotalOriginal*100) / 100
	result.TotalDiscount = math.Round(result.TotalDiscount*100) / 100
	result.TotalPay = math.Round(result.TotalPay*100) / 100
}

func (e *PromoEngine) checkGiftStock(skuID int64, quantity int) bool {
	ctx := context.Background()
	var stock int
	err := db.Pool.QueryRow(ctx, "SELECT stock FROM skus WHERE id = $1", skuID).Scan(&stock)
	if err != nil {
		return false
	}
	return stock >= quantity
}

func copyMap(m map[int64]bool) map[int64]bool {
	result := make(map[int64]bool)
	for k, v := range m {
		result[k] = v
	}
	return result
}

func copyResult(r *models.CalculationResult) *models.CalculationResult {
	result := &models.CalculationResult{
		Items:         make([]models.CalculatedItem, len(r.Items)),
		TotalOriginal: r.TotalOriginal,
		TotalDiscount: r.TotalDiscount,
		TotalPay:      r.TotalPay,
		GiftItems:     make([]models.CalculatedItem, len(r.GiftItems)),
	}
	
	for i, item := range r.Items {
		result.Items[i] = models.CalculatedItem{
			SKUId:          item.SKUId,
			SKUName:        item.SKUName,
			StoreID:        item.StoreID,
			OriginalPrice:  item.OriginalPrice,
			DiscountAmount: item.DiscountAmount,
			PayPrice:       item.PayPrice,
			Quantity:       item.Quantity,
			PromoRuleIDs:   append([]int64{}, item.PromoRuleIDs...),
		}
	}
	
	for i, item := range r.GiftItems {
		result.GiftItems[i] = models.CalculatedItem{
			SKUId:          item.SKUId,
			SKUName:        item.SKUName,
			StoreID:        item.StoreID,
			OriginalPrice:  item.OriginalPrice,
			DiscountAmount: item.DiscountAmount,
			PayPrice:       item.PayPrice,
			Quantity:       item.Quantity,
			PromoRuleIDs:   append([]int64{}, item.PromoRuleIDs...),
		}
	}
	
	return result
}
