package server

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"gorm.io/gorm"
	"net/http"
)

func Router(db *gorm.DB) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.Logger.SetLevel(log.INFO)

	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.RequestIDWithConfig(middleware.RequestIDConfig{
		Generator: func() string { return newUUID() },
	}))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
    AllowOrigins: []string{"*"}, // allow any origin (development only!)
    AllowMethods: []string{
        http.MethodGet,
        http.MethodHead,
        http.MethodPut,
        http.MethodPost,
        http.MethodPatch,
        http.MethodDelete,
        http.MethodOptions,
    },
    AllowHeaders: []string{
        echo.HeaderOrigin,
        echo.HeaderContentType,
        echo.HeaderAccept,
        echo.HeaderAuthorization,
        "X-Requested-With",
        "X-CSRF-Token",
    },
    ExposeHeaders: []string{
        echo.HeaderContentLength,
        echo.HeaderContentType,
        echo.HeaderAuthorization,
    },
    AllowCredentials: false, // must stay false with "*" origins
    MaxAge: 86400,
}))

	e.Use(middleware.Recover())
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `${time_rfc3339} id=${id} remote_ip=${remote_ip} method=${method} uri=${uri} status=${status} latency=${latency_human} bytes_in=${bytes_in} bytes_out=${bytes_out} ua=${user_agent} error=${error}\n`,
	}))

	e.Renderer = NewRenderer()

	registerRoutes(e, db)
	return e
}
