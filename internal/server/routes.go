package server

import (
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func registerRoutes(e *echo.Echo, db *gorm.DB) {
	e.GET("/.well-known/apple-developer-merchantid-domain-association", func(c echo.Context) error {
		path := "public/.well-known/apple-developer-merchantid-domain-association"
		b, err := os.ReadFile(path)
		if err != nil {
			return c.NoContent(http.StatusNotFound)
		}
		return c.Blob(http.StatusOK, "text/plain; charset=utf-8", b)
	})
	e.GET("/applePayIntegrationTest.html", func(c echo.Context) error {
		return c.File("public/applePayIntegrationTest.html")
	})
	e.GET("/", func(c echo.Context) error {
		return c.File("public/index.html")
	})
	e.POST("/api/payment-pages", func(c echo.Context) error { return handleCreatePaymentPage(c, db) })
	e.POST("/api/payments/:merchant_id/:page_uid/charge", func(c echo.Context) error { return handleChargePayment(c, db) })

	e.GET("/p/:merchant_id/:page_uid", func(c echo.Context) error { return handleViewPaymentPage(c, db) })
	e.GET("/qr/:merchant_id/:page_uid", func(c echo.Context) error { return handleQRPaymentPage(c) })

}
