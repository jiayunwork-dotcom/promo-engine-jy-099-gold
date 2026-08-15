# 电商促销规则引擎和价格计算服务

## 项目概述

这是一个完整的电商促销规则引擎和价格计算服务，支持多种促销类型，能够自动计算最优优惠组合。

## 功能特性

### 促销类型
- **满减**: 订单金额满X元减Y元，支持阶梯设置
- **折扣**: 指定商品打N折，支持按品类或SKU设置
- **买赠**: 买A商品赠B商品，库存不足自动失效
- **第N件优惠**: 同一商品第2件半价/第3件免费
- **跨店满减**: 多店铺凑单满足条件后统一减免
- **组合优惠**: 指定组合商品享受组合价

### 核心功能
- 规则配置: 适用范围、时间条件、互斥关系、使用限制
- 最优组合计算: 支持贪心、动态规划、分支限界三种算法
- 价格试算API: P99 < 100ms
- 规则冲突检测
- 促销效果预估
- 规则版本管理
- 完整的Web管理后台

## 技术栈

### 后端
- Go 1.21 + Echo 框架
- PostgreSQL 15 (数据持久化)
- Redis 7 (规则缓存、分布式锁、消息通知)

### 前端
- Svelte 4
- Vite
- Chart.js (数据可视化)

## 快速开始

### 使用 Docker Compose 启动

```bash
docker-compose up -d
```

服务启动后:
- 前端管理面板: http://localhost
- 后端API: http://localhost:8080

### 手动启动

#### 启动依赖服务
```bash
docker-compose up -d postgres redis
```

#### 启动后端
```bash
cd backend
go mod download
go run cmd/main.go
```

#### 启动前端
```bash
cd frontend
npm install
npm run dev
```

## API 接口

### 价格计算
```
POST /api/price/calculate
Content-Type: application/json

{
  "user_id": "user123",
  "items": [
    {
      "sku_id": 1,
      "sku_name": "商品A",
      "store_id": 1,
      "price": 100.00,
      "quantity": 2
    }
  ],
  "user_tags": ["vip"]
}
```

### 促销规则管理
- `GET /api/rules` - 获取规则列表
- `POST /api/rules` - 创建规则
- `PUT /api/rules/:id` - 更新规则
- `DELETE /api/rules/:id` - 删除规则
- `PATCH /api/rules/:id/status` - 更新规则状态

### 冲突检测和效果预估
- `POST /api/rules/detect-conflicts` - 检测规则冲突
- `POST /api/rules/estimate-effect` - 预估规则效果

## 计算策略

系统支持三种优惠计算策略，可通过环境变量 `CALCULATION_STRATEGY` 配置:

1. **greedy** (默认): 贪心算法，按优先级依次应用规则
2. **dp**: 动态规划，对互斥组内规则求最优子集
3. **branch_bound**: 分支限界，50ms内搜索更优解

## 项目结构

```
promo-engine/
├── backend/                 # Go 后端服务
│   ├── cmd/                # 应用入口
│   ├── internal/
│   │   ├── config/         # 配置管理
│   │   ├── db/             # 数据库连接
│   │   ├── cache/          # Redis缓存
│   │   ├── models/         # 数据模型
│   │   ├── engine/         # 促销引擎核心
│   │   └── handlers/       # HTTP处理程序
│   ├── db/migrations/      # 数据库迁移
│   └── Dockerfile
├── frontend/               # Svelte 前端管理面板
│   ├── src/
│   │   ├── routes/         # 页面组件
│   │   └── lib/            # 工具和公共组件
│   ├── nginx.conf
│   └── Dockerfile
└── docker-compose.yml
```

## 数据库表结构

- `promo_rules`: 促销规则主表
- `promo_rule_versions`: 规则历史版本
- `mutex_groups`: 互斥组
- `promo_mutex_relations`: 规则互斥关系
- `promo_usage`: 规则使用记录
- `promo_effect_stats`: 效果统计
- `skus`: 商品SKU
- `categories`: 品类
- `stores`: 店铺
- `orders`: 订单
- `order_items`: 订单项

## 注意事项

1. 价格计算时会自动兜底到成本价，防止亏损
2. 跨店满减使用最大余额法处理尾差
3. 活跃规则缓存在内存中，变更时通过Redis Pub/Sub刷新
4. 定时生效的规则会自动切换状态
