package server

import (
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func registerRoutes(e *echo.Echo, db *gorm.DB) {
	e.POST("/api/payment-pages", func(c echo.Context) error { return handleCreatePaymentPage(c, db) })
	e.GET("/p/:merchant_id/:page_uid", func(c echo.Context) error { return handleViewPaymentPage(c, db) })
}
