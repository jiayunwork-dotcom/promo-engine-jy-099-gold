CREATE TABLE IF NOT EXISTS coupon_batches (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    coupon_type VARCHAR(30) NOT NULL,
    discount_amount DECIMAL(12,2),
    discount_rate DECIMAL(5,4),
    max_discount_amount DECIMAL(12,2),
    threshold_amount DECIMAL(12,2) DEFAULT 0,
    scope JSONB NOT NULL,
    valid_from TIMESTAMPTZ NOT NULL,
    valid_to TIMESTAMPTZ NOT NULL,
    total_quantity INT NOT NULL,
    claimed_quantity INT DEFAULT 0,
    used_quantity INT DEFAULT 0,
    per_user_limit INT DEFAULT 1,
    created_by VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS coupons (
    id BIGSERIAL PRIMARY KEY,
    batch_id BIGINT REFERENCES coupon_batches(id) ON DELETE CASCADE,
    code VARCHAR(12) UNIQUE NOT NULL,
    user_id VARCHAR(100),
    status VARCHAR(30) DEFAULT 'available',
    order_id VARCHAR(100),
    claimed_at TIMESTAMPTZ,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_coupons_code ON coupons(code);
CREATE INDEX IF NOT EXISTS idx_coupons_user ON coupons(user_id, status);
CREATE INDEX IF NOT EXISTS idx_coupons_batch ON coupons(batch_id);
CREATE INDEX IF NOT EXISTS idx_coupon_batches_valid ON coupon_batches(valid_from, valid_to);
