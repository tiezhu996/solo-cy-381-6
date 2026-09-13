package router

import (
	"github.com/aasplit/aasplit/internal/handler"
	"github.com/aasplit/aasplit/internal/middleware"
	"github.com/aasplit/aasplit/internal/util"
	"github.com/gin-gonic/gin"
)

// RegisterBudgetRoutes 注册群组月度预算路由。
func RegisterBudgetRoutes(r *gin.RouterGroup, h *handler.BudgetHandler, jwt *util.JWTManager) {
	budgets := r.Group("/groups/:id/budgets")
	budgets.Use(middleware.Auth(jwt))
	{
		budgets.GET("", h.List)
		budgets.POST("", h.Create)
	}
	stats := r.Group("/groups/:id/stats")
	stats.Use(middleware.Auth(jwt))
	{
		stats.GET("/budgets", h.Stats)
	}
	single := r.Group("/budgets")
	single.Use(middleware.Auth(jwt))
	{
		single.PUT("/:id", h.Update)
		single.DELETE("/:id", h.Disable)
	}
}
