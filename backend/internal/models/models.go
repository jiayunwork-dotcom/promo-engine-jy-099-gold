package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type PromoType string

const (
	PromoTypeFullReduction  PromoType = "full_reduction"
	PromoTypeDiscount       PromoType = "discount"
	PromoTypeBuyGift        PromoType = "buy_gift"
	PromoTypeNthItem        PromoType = "nth_item"
	PromoTypeCrossStore     PromoType = "cross_store"
	PromoTypeCombo          PromoType = "combo"
)

type PromoStatus string

const (
	PromoStatusDraft    PromoStatus = "draft"
	PromoStatusReview   PromoStatus = "review"
	PromoStatusActive   PromoStatus = "active"
	PromoStatusExpired  PromoStatus = "expired"
)

type PromoScope struct {
	Type        string   `json:"type"`
	CategoryIDs []int64  `json:"category_ids,omitempty"`
	StoreIDs    []int64  `json:"store_ids,omitempty"`
	SKUIds      []int64  `json:"sku_ids,omitempty"`
	UserTags    []string `json:"user_tags,omitempty"`
}

type TimeCondition struct {
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Weekdays    []int     `json:"weekdays,omitempty"`
	TimeRanges  []TimeRange `json:"time_ranges,omitempty"`
}

type TimeRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type UsageLimit struct {
	PerUser     int `json:"per_user,omitempty"`
	PerUserDay  int `json:"per_user_day,omitempty"`
	TotalOrders int `json:"total_orders,omitempty"`
}

type FullReductionConfig struct {
	Tiers []ReductionTier `json:"tiers"`
}

type ReductionTier struct {
	Threshold float64 `json:"threshold"`
	Discount  float64 `json:"discount"`
}

type DiscountConfig struct {
	DiscountRate float64 `json:"discount_rate"`
}

type BuyGiftConfig struct {
	BuySKUId    int64 `json:"buy_sku_id"`
	BuyQuantity int   `json:"buy_quantity"`
	GiftSKUId   int64 `json:"gift_sku_id"`
	GiftQuantity int  `json:"gift_quantity"`
}

type NthItemConfig struct {
	NthItem     int     `json:"nth_item"`
	DiscountRate float64 `json:"discount_rate"`
	Free        bool    `json:"free,omitempty"`
}

type CrossStoreConfig struct {
	Threshold float64 `json:"threshold"`
	Discount  float64 `json:"discount"`
}

type ComboConfig struct {
	SKUIds      []int64 `json:"sku_ids"`
	ComboPrice  float64 `json:"combo_price"`
}

type PromoRule struct {
	ID             int64         `json:"id"`
	Name           string        `json:"name"`
	Description    string        `json:"description,omitempty"`
	PromoType      PromoType     `json:"promo_type"`
	Status         PromoStatus   `json:"status"`
	Version        int           `json:"version"`
	Config         JSONB         `json:"config"`
	Scope          PromoScope    `json:"scope"`
	TimeCondition  TimeCondition `json:"time_condition"`
	UsageLimit     *UsageLimit   `json:"usage_limit,omitempty"`
	Priority       int           `json:"priority"`
	CreatedBy      string        `json:"created_by,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

type SKU struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	CategoryID int64   `json:"category_id"`
	StoreID    int64   `json:"store_id"`
	Price      float64 `json:"price"`
	CostPrice  float64 `json:"cost_price"`
	Stock      int     `json:"stock"`
}

type CartItem struct {
	SKUId     int64   `json:"sku_id"`
	SKUName   string  `json:"sku_name,omitempty"`
	StoreID   int64   `json:"store_id"`
	Price     float64 `json:"price"`
	CostPrice float64 `json:"cost_price,omitempty"`
	Quantity  int     `json:"quantity"`
}

type Cart struct {
	UserID    string     `json:"user_id"`
	Items     []CartItem `json:"items"`
	UserTags  []string   `json:"user_tags,omitempty"`
}

type CalculatedItem struct {
	SKUId          int64   `json:"sku_id"`
	SKUName        string  `json:"sku_name"`
	StoreID        int64   `json:"store_id"`
	OriginalPrice  float64 `json:"original_price"`
	DiscountAmount float64 `json:"discount_amount"`
	PayPrice       float64 `json:"pay_price"`
	Quantity       int     `json:"quantity"`
	PromoRuleIDs   []int64 `json:"promo_rule_ids"`
}

type ConflictDetectionResult struct {
	HasConflict     bool     `json:"has_conflict"`
	ConflictingRuleIDs []int64 `json:"conflicting_rule_ids,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
	Errors          []string `json:"errors,omitempty"`
}

type EstimationResult struct {
	EstimatedOrders    int     `json:"estimated_orders"`
	EstimatedDiscount  float64 `json:"estimated_discount"`
	EstimatedGMVChange float64 `json:"estimated_gmv_change"`
}

type CouponType string

const (
	CouponTypeFullReduction CouponType = "full_reduction"
	CouponTypeDiscount      CouponType = "discount"
	CouponTypeNoThreshold   CouponType = "no_threshold"
)

type CouponStatus string

const (
	CouponStatusAvailable CouponStatus = "available"
	CouponStatusClaimed   CouponStatus = "claimed"
	CouponStatusUsed      CouponStatus = "used"
	CouponStatusExpired   CouponStatus = "expired"
)

type CouponScope struct {
	Type        string  `json:"type"`
	CategoryIDs []int64 `json:"category_ids,omitempty"`
	StoreIDs    []int64 `json:"store_ids,omitempty"`
}

type CouponBatch struct {
	ID               int64       `json:"id"`
	Name             string      `json:"name"`
	Description      string      `json:"description,omitempty"`
	CouponType       CouponType  `json:"coupon_type"`
	DiscountAmount   float64     `json:"discount_amount,omitempty"`
	DiscountRate     float64     `json:"discount_rate,omitempty"`
	MaxDiscountAmount float64    `json:"max_discount_amount,omitempty"`
	ThresholdAmount  float64     `json:"threshold_amount"`
	Scope            CouponScope `json:"scope"`
	ValidFrom        time.Time   `json:"valid_from"`
	ValidTo          time.Time   `json:"valid_to"`
	TotalQuantity    int         `json:"total_quantity"`
	ClaimedQuantity  int         `json:"claimed_quantity"`
	UsedQuantity     int         `json:"used_quantity"`
	PerUserLimit     int         `json:"per_user_limit"`
	CreatedBy        string      `json:"created_by,omitempty"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}

type Coupon struct {
	ID        int64        `json:"id"`
	BatchID   int64        `json:"batch_id"`
	Code      string       `json:"code"`
	UserID    string       `json:"user_id,omitempty"`
	Status    CouponStatus `json:"status"`
	OrderID   string       `json:"order_id,omitempty"`
	ClaimedAt *time.Time   `json:"claimed_at,omitempty"`
	UsedAt    *time.Time   `json:"used_at,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
	Batch     *CouponBatch `json:"batch,omitempty"`
}

type CalculationResult struct {
	Items          []CalculatedItem `json:"items"`
	TotalOriginal  float64          `json:"total_original"`
	TotalDiscount  float64          `json:"total_discount"`
	TotalPay       float64          `json:"total_pay"`
	GiftItems      []CalculatedItem `json:"gift_items,omitempty"`
	CouponDiscount float64          `json:"coupon_discount,omitempty"`
	CouponCode     string           `json:"coupon_code,omitempty"`
}

type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, j)
}
