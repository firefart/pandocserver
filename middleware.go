package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func (app *application) middlewareRecover() echo.MiddlewareFunc {
	return middleware.RecoverWithConfig(middleware.RecoverConfig{
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			// send the error to the default error handler
			return fmt.Errorf("PANIC! %v - %s", err, string(stack))
		},
	})
}

func (app *application) middlewareRequestLogger(ctx context.Context) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:        true,
		LogURI:           true,
		LogUserAgent:     true,
		LogLatency:       true,
		LogRemoteIP:      true,
		LogMethod:        true,
		LogContentLength: true,
		LogResponseSize:  true,
		LogError:         true,
		HandleError:      true, // forwards error to the global error handler, so it can decide appropriate status code
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			logLevel := slog.LevelInfo
			errString := ""
			// only set error on real errors
			if v.Error != nil && v.Status > 499 {
				errString = v.Error.Error()
				logLevel = slog.LevelError
			}
			app.logger.LogAttrs(ctx, logLevel, "REQUEST",
				slog.String("ip", v.RemoteIP),
				slog.String("method", v.Method),
				slog.String("uri", v.URI),
				slog.Int("status", v.Status),
				slog.String("user-agent", v.UserAgent),
				slog.Duration("request-duration", v.Latency),
				slog.String("request-length", v.ContentLength), // request content length
				slog.Int64("response-size", v.ResponseSize),
				slog.String("err", errString))

			return nil
		},
	})
}
