package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"promo-engine/internal/cache"
	"promo-engine/internal/config"
	"promo-engine/internal/db"
	"promo-engine/internal/engine"
	"promo-engine/internal/handlers"
	"promo-engine/internal/models"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	cfg := config.Load()

	if err := db.Init(cfg); err != nil {
		panic("failed to connect database: " + err.Error())
	}
	defer db.Close()

	cache.Init(cfg)

	strategy := engine.StrategyGreedy
	if s := os.Getenv("CALCULATION_STRATEGY"); s != "" {
		strategy = engine.CalculationStrategy(s)
	}
	engine.Init(strategy, cfg.CacheTTL)

	e := echo.New()

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.OPTIONS},
		AllowHeaders: []string{"*"},
	}))

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	promoHandler := handlers.NewPromoHandler()
	mutexHandler := handlers.NewMutexHandler()
	couponHandler := handlers.NewCouponHandler()

	api := e.Group("/api")
	{
		api.POST("/price/calculate", promoHandler.CalculatePrice)

		rules := api.Group("/rules")
		{
			rules.GET("", promoHandler.ListRules)
			rules.GET("/:id", promoHandler.GetRule)
			rules.POST("", promoHandler.CreateRule)
			rules.PUT("/:id", promoHandler.UpdateRule)
			rules.DELETE("/:id", promoHandler.DeleteRule)
			rules.PATCH("/:id/status", promoHandler.UpdateRuleStatus)
			rules.GET("/:id/versions", promoHandler.GetRuleVersions)
			rules.POST("/:id/versions/:version/rollback", promoHandler.RollbackVersion)
		}

		api.POST("/rules/detect-conflicts", promoHandler.DetectConflicts)
		api.POST("/rules/estimate-effect", promoHandler.EstimateEffect)

		mutex := api.Group("/mutex-groups")
		{
			mutex.GET("", mutexHandler.ListGroups)
			mutex.GET("/:id", mutexHandler.GetGroup)
			mutex.POST("", mutexHandler.CreateGroup)
			mutex.PUT("/:id", mutexHandler.UpdateGroup)
			mutex.DELETE("/:id", mutexHandler.DeleteGroup)
			mutex.POST("/:id/rules/:ruleId", mutexHandler.AddRuleToGroup)
			mutex.DELETE("/:id/rules/:ruleId", mutexHandler.RemoveRuleFromGroup)
		}

		couponBatches := api.Group("/coupon-batches")
		{
			couponBatches.GET("", couponHandler.ListBatches)
			couponBatches.GET("/:id", couponHandler.GetBatch)
			couponBatches.POST("", couponHandler.CreateBatch)
			couponBatches.GET("/:id/coupons", couponHandler.ListBatchCoupons)
		}

		coupons := api.Group("/coupons")
		{
			coupons.POST("/claim", couponHandler.ClaimCoupon)
			coupons.GET("/user/:user_id", couponHandler.ListUserCoupons)
			coupons.POST("/use", couponHandler.UseCoupon)
			coupons.POST("/return", couponHandler.ReturnCoupon)
		}

		api.GET("/health", func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
		})
	}

	go startStatusScheduler()

	go func() {
		if err := e.Start(":" + cfg.ServerPort); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down the server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}

func startStatusScheduler() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		now := time.Now()

		db.Pool.Exec(ctx, `
			UPDATE promo_rules 
			SET status = $1, updated_at = CURRENT_TIMESTAMP
			WHERE status = $2 
			AND (time_condition->>'end_time')::timestamptz < $3
		`, models.PromoStatusExpired, models.PromoStatusActive, now)

		db.Pool.Exec(ctx, `
			UPDATE promo_rules 
			SET status = $1, updated_at = CURRENT_TIMESTAMP
			WHERE status = $2 
			AND (time_condition->>'start_time')::timestamptz <= $3
			AND (time_condition->>'end_time')::timestamptz > $3
		`, models.PromoStatusActive, models.PromoStatusReview, now)
	}
}
