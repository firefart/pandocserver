package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type echoJsonError struct {
	err         error
	code        int
	userMessage string
}

func (e echoJsonError) Error() string {
	return e.userMessage
}

func newEchoJsonError(err error, code int, message string) echoJsonError {
	return echoJsonError{
		err:         err,
		code:        code,
		userMessage: message,
	}
}

func (app *application) newServer(ctx context.Context) http.Handler {
	e := echo.New()
	e.HideBanner = true
	e.Debug = app.debug
	e.HTTPErrorHandler = app.customHTTPErrorHandler

	if app.config.Cloudflare {
		e.IPExtractor = extractIPFromCloudflareHeader()
	}

	e.Use(app.middlewareRequestLogger(ctx))
	e.Use(middleware.Secure())
	e.Use(app.middlewareRecover())

	// add all the routes
	app.addRoutes(e)
	return e
}

func (app *application) customHTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	type jsonErrorResponse struct {
		Error string `json:"error"`
	}

	code := http.StatusInternalServerError
	msg := "error occured - please see log"
	var echoError *echo.HTTPError
	var jsonError *echoJsonError
	switch {
	case errors.As(err, &echoError):
		code = echoError.Code
		msg = fmt.Sprintf("%v", echoError.Message)
	case errors.As(err, &jsonError):
		code = jsonError.code
		msg = jsonError.userMessage
	}

	// send an asynchronous notification (but ignore 404 and stuff)
	if err != nil && code > 499 {
		app.logger.Error("error on request", slog.String("err", err.Error()))

		go func(e error) {
			app.logger.Debug("sending error notification", slog.String("err", e.Error()))
			if err2 := app.notify.Send(context.Background(), "ERROR", e.Error()); err2 != nil {
				app.logger.Error("error on notification send", slog.String("err", err2.Error()))
			}
		}(err)
	}

	// send error json
	if err2 := c.JSON(code, jsonErrorResponse{Error: msg}); err2 != nil {
		app.logger.Error("could not send error page", slog.String("err", err2.Error()))
		return
	}
}

func extractIPFromCloudflareHeader() echo.IPExtractor {
	return func(req *http.Request) string {
		if realIP := req.Header.Get(cloudflareIPHeaderName); realIP != "" {
			return realIP
		}
		// fall back to normal ip extraction
		return echo.ExtractIPDirect()(req)
	}
}
