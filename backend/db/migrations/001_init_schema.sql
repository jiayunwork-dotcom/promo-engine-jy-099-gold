CREATE TABLE IF NOT EXISTS categories (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    parent_id BIGINT REFERENCES categories(id),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS stores (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS skus (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    category_id BIGINT REFERENCES categories(id),
    store_id BIGINT REFERENCES stores(id),
    price DECIMAL(12,2) NOT NULL,
    cost_price DECIMAL(12,2) NOT NULL,
    stock INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS promo_rules (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    promo_type VARCHAR(50) NOT NULL,
    status VARCHAR(30) DEFAULT 'draft',
    version INT DEFAULT 1,
    config JSONB NOT NULL,
    scope JSONB NOT NULL,
    time_condition JSONB NOT NULL,
    usage_limit JSONB,
    priority INT DEFAULT 0,
    created_by VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS promo_rule_versions (
    id BIGSERIAL PRIMARY KEY,
    rule_id BIGINT REFERENCES promo_rules(id) ON DELETE CASCADE,
    version INT NOT NULL,
    name VARCHAR(200) NOT NULL,
    config JSONB NOT NULL,
    scope JSONB NOT NULL,
    time_condition JSONB NOT NULL,
    usage_limit JSONB,
    priority INT DEFAULT 0,
    created_by VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(rule_id, version)
);

CREATE TABLE IF NOT EXISTS mutex_groups (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS promo_mutex_relations (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT REFERENCES mutex_groups(id) ON DELETE CASCADE,
    rule_id BIGINT REFERENCES promo_rules(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(group_id, rule_id)
);

CREATE TABLE IF NOT EXISTS promo_usage (
    id BIGSERIAL PRIMARY KEY,
    rule_id BIGINT REFERENCES promo_rules(id) ON DELETE CASCADE,
    user_id VARCHAR(100) NOT NULL,
    order_id VARCHAR(100),
    discount_amount DECIMAL(12,2) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS promo_effect_stats (
    id BIGSERIAL PRIMARY KEY,
    rule_id BIGINT REFERENCES promo_rules(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    usage_count INT DEFAULT 0,
    total_discount DECIMAL(12,2) DEFAULT 0,
    total_gmv DECIMAL(12,2) DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(rule_id, date)
);

CREATE TABLE IF NOT EXISTS orders (
    id BIGSERIAL PRIMARY KEY,
    order_no VARCHAR(64) UNIQUE NOT NULL,
    user_id VARCHAR(100) NOT NULL,
    total_amount DECIMAL(12,2) NOT NULL,
    discount_amount DECIMAL(12,2) DEFAULT 0,
    pay_amount DECIMAL(12,2) NOT NULL,
    status VARCHAR(30) DEFAULT 'pending',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS order_items (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT REFERENCES orders(id) ON DELETE CASCADE,
    sku_id BIGINT REFERENCES skus(id),
    sku_name VARCHAR(200),
    store_id BIGINT,
    original_price DECIMAL(12,2) NOT NULL,
    discount_amount DECIMAL(12,2) DEFAULT 0,
    pay_price DECIMAL(12,2) NOT NULL,
    quantity INT NOT NULL,
    promo_rule_ids BIGINT[],
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_promo_rules_status ON promo_rules(status);
CREATE INDEX IF NOT EXISTS idx_promo_rules_type ON promo_rules(promo_type);
CREATE INDEX IF NOT EXISTS idx_promo_usage_user ON promo_usage(user_id, rule_id);
CREATE INDEX IF NOT EXISTS idx_order_items_sku ON order_items(sku_id);
