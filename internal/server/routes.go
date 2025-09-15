package server

import (
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func registerRoutes(e *echo.Echo, db *gorm.DB) {
	e.Static("/.well-known", "public/.well-known")
	e.File("/applePayIntegrationTest.html", "public/applePayIntegrationTest.html")
	e.File("/", "public/index.html")
	e.POST("/api/payment-pages", func(c echo.Context) error { return handleCreatePaymentPage(c, db) })
	e.POST("/api/payments/:merchant_id/:page_uid/charge", func(c echo.Context) error { return handleChargePayment(c, db) })

	e.GET("/p/:merchant_id/:page_uid", func(c echo.Context) error { return handleViewPaymentPage(c, db) })
	e.GET("/qr/:merchant_id/:page_uid", func(c echo.Context) error { return handleQRPaymentPage(c) })

}
