package main

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"
)

func (app *application) addRoutes(e *echo.Echo) {
	e.GET("/health", app.handleHealth)
	e.GET("/test_panic", app.handleTestPanic)
	e.GET("/test_notifications", app.handleTestNotification)
	e.POST("/convert", func(c *echo.Context) error {
		type jsonData struct {
			Input     []byte            `json:"input"`
			Resources map[string][]byte `json:"resources"`
			Template  string            `json:"template"`
		}
		type jsonResponse struct {
			Content []byte `json:"content"`
		}

		var d jsonData
		if err := c.Bind(&d); err != nil {
			return c.JSON(http.StatusBadRequest, newEchoJsonError(err, http.StatusBadRequest, "invalid input"))
		}

		if d.Input == nil || d.Template == "" {
			return c.JSON(http.StatusBadRequest, echo.NewHTTPError(http.StatusBadRequest, "invalid input"))
		}

		bin, err := app.convert(c.Request().Context(), d.Input, d.Resources, d.Template)
		if err != nil {
			app.logger.Error("error on convert", slog.String("error", err.Error()))
			return c.JSON(http.StatusBadRequest, newEchoJsonError(err, http.StatusBadRequest, "error converting markdown"))
		}

		return c.JSON(http.StatusOK, jsonResponse{Content: bin})
	})
}
