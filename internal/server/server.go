package server

import (
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func Router(db *gorm.DB) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.Renderer = NewRenderer()

	registerRoutes(e, db)
	return e
}
