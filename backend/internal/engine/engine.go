package engine

import (
	"context"
	"encoding/json"
	"promo-engine/internal/cache"
	"promo-engine/internal/db"
	"promo-engine/internal/models"
	"sync"
	"time"
)

const (
	CacheKeyActiveRules = "promo:active_rules"
	RuleChangeChannel   = "promo:rule_change"
)

type CalculationStrategy string

const (
	StrategyGreedy       CalculationStrategy = "greedy"
	StrategyDP           CalculationStrategy = "dp"
	StrategyBranchBound  CalculationStrategy = "branch_bound"
)

type PromoEngine struct {
	activeRules []*models.PromoRule
	mutex       sync.RWMutex
	strategy    CalculationStrategy
	cacheTTL    int
}

var Engine *PromoEngine

func Init(strategy CalculationStrategy, cacheTTL int) {
	Engine = &PromoEngine{
		strategy: strategy,
		cacheTTL: cacheTTL,
	}
	Engine.loadActiveRules()
	go Engine.subscribeRuleChanges()
}

func (e *PromoEngine) loadActiveRules() {
	ctx := context.Background()
	
	var rules []*models.PromoRule
	err := cache.Get(ctx, CacheKeyActiveRules, &rules)
	if err == nil && len(rules) > 0 {
		e.mutex.Lock()
		e.activeRules = rules
		e.mutex.Unlock()
		return
	}

	query := `
		SELECT id, name, description, promo_type, status, version, 
		       config, scope, time_condition, usage_limit, priority, 
		       created_by, created_at, updated_at
		FROM promo_rules 
		WHERE status = $1
	`
	rows, err := db.Pool.Query(ctx, query, models.PromoStatusActive)
	if err != nil {
		return
	}
	defer rows.Close()

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
		
		rules = append(rules, &rule)
	}

	e.mutex.Lock()
	e.activeRules = rules
	e.mutex.Unlock()

	cache.Set(ctx, CacheKeyActiveRules, rules, e.cacheTTL)
}

func (e *PromoEngine) subscribeRuleChanges() {
	ctx := context.Background()
	pubsub := cache.Subscribe(ctx, RuleChangeChannel)
	defer pubsub.Close()

	ch := pubsub.Channel()
	for range ch {
		e.loadActiveRules()
	}
}

func (e *PromoEngine) GetActiveRules() []*models.PromoRule {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.activeRules
}

func (e *PromoEngine) Calculate(cart *models.Cart) (*models.CalculationResult, error) {
	now := time.Now()
	
	applicableRules := e.filterApplicableRules(cart, now)
	
	switch e.strategy {
	case StrategyDP:
		return e.calculateWithDP(cart, applicableRules)
	case StrategyBranchBound:
		return e.calculateWithBranchBound(cart, applicableRules)
	default:
		return e.calculateWithGreedy(cart, applicableRules)
	}
}

func (e *PromoEngine) filterApplicableRules(cart *models.Cart, now time.Time) []*models.PromoRule {
	e.mutex.RLock()
	rules := e.activeRules
	e.mutex.RUnlock()

	var applicable []*models.PromoRule
	for _, rule := range rules {
		if e.isRuleApplicable(rule, cart, now) {
			applicable = append(applicable, rule)
		}
	}
	return applicable
}

func (e *PromoEngine) isRuleApplicable(rule *models.PromoRule, cart *models.Cart, now time.Time) bool {
	if now.Before(rule.TimeCondition.StartTime) || now.After(rule.TimeCondition.EndTime) {
		return false
	}

	if len(rule.TimeCondition.Weekdays) > 0 {
		weekday := int(now.Weekday())
		found := false
		for _, wd := range rule.TimeCondition.Weekdays {
			if wd == weekday {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(rule.TimeCondition.TimeRanges) > 0 {
		currentTime := now.Format("15:04")
		inRange := false
		for _, tr := range rule.TimeCondition.TimeRanges {
			if currentTime >= tr.Start && currentTime <= tr.End {
				inRange = true
				break
			}
		}
		if !inRange {
			return false
		}
	}

	if !e.isScopeMatch(rule.Scope, cart) {
		return false
	}

	return true
}

func (e *PromoEngine) isScopeMatch(scope models.PromoScope, cart *models.Cart) bool {
	switch scope.Type {
	case "all":
		return true
	case "category":
		for _, item := range cart.Items {
			for _, catID := range scope.CategoryIDs {
				if e.isSKUInCategory(item.SKUId, catID) {
					return true
				}
			}
		}
	case "store":
		for _, item := range cart.Items {
			for _, storeID := range scope.StoreIDs {
				if item.StoreID == storeID {
					return true
				}
			}
		}
	case "sku":
		for _, item := range cart.Items {
			for _, skuID := range scope.SKUIds {
				if item.SKUId == skuID {
					return true
				}
			}
		}
	case "user_tag":
		for _, tag := range cart.UserTags {
			for _, scopeTag := range scope.UserTags {
				if tag == scopeTag {
					return true
				}
			}
		}
	}
	return false
}

func (e *PromoEngine) isSKUInCategory(skuID, categoryID int64) bool {
	ctx := context.Background()
	var catID int64
	err := db.Pool.QueryRow(ctx, "SELECT category_id FROM skus WHERE id = $1", skuID).Scan(&catID)
	if err != nil {
		return false
	}
	return catID == categoryID
}
