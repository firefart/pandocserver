package main

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

func (app *application) handleTestPanic(c echo.Context) error {
	// no checks in debug mode
	if app.debug {
		panic("test")
	}

	headerValue := c.Request().Header.Get(secretKeyHeaderName)
	switch headerValue {
	case "":
		app.logger.Error("test_panic called without secret header")
	case app.config.Notifications.SecretKeyHeader:
		panic("test")
	default:
		app.logger.Error("test_panic called without valid header")
	}
	return c.Render(http.StatusOK, "index.html", nil)
}

func (app *application) handleTestNotification(c echo.Context) error {
	// no checks in debug mode
	if app.debug {
		return fmt.Errorf("test")
	}

	headerValue := c.Request().Header.Get(secretKeyHeaderName)
	switch headerValue {
	case "":
		app.logger.Error("test_notification called without secret header")
	case app.config.Notifications.SecretKeyHeader:
		return fmt.Errorf("test")
	default:
		app.logger.Error("test_notification called without valid header")
	}
	return c.Render(http.StatusOK, "index.html", nil)
}
