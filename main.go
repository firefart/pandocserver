package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/firefart/pandocserver/internal/config"

	"github.com/nikoksr/notify"

	_ "net/http/pprof"
)

var secretKeyHeaderName = http.CanonicalHeaderKey("X-Secret-Key-Header")
var cloudflareIPHeaderName = http.CanonicalHeaderKey("CF-Connecting-IP")

type application struct {
	logger *slog.Logger
	debug  bool
	config config.Configuration
	notify *notify.Notify
}

func main() {
	var debugMode bool
	var configFilename string
	var jsonOutput bool
	flag.BoolVar(&debugMode, "debug", false, "Enable DEBUG mode")
	flag.StringVar(&configFilename, "config", "", "config file to use")
	flag.BoolVar(&jsonOutput, "json", false, "output in json instead")
	flag.Parse()

	logger := newLogger(debugMode, jsonOutput)
	ctx := context.Background()
	if err := run(ctx, logger, configFilename, debugMode); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger, configFilename string, debug bool) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	app := &application{
		logger: logger,
		debug:  debug,
	}

	configuration, err := config.GetConfig(configFilename)
	if err != nil {
		return err
	}
	app.config = configuration

	app.notify, err = setupNotifications(configuration, logger)
	if err != nil {
		return err
	}

	tlsConfig, err := app.setupTLSConfig()
	if err != nil {
		return err
	}

	app.logger.Info("Starting server",
		slog.String("host", configuration.Server.Listen),
		slog.Duration("gracefultimeout", configuration.Server.GracefulTimeout),
		slog.Duration("timeout", configuration.Timeout),
		slog.Bool("debug", app.debug),
	)

	srv := &http.Server{
		Addr:         configuration.Server.Listen,
		Handler:      app.newServer(ctx),
		TLSConfig:    tlsConfig,
		ReadTimeout:  configuration.Timeout,
		WriteTimeout: configuration.Timeout,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.logger.Error("error on listenandserve", slog.String("err", err.Error()))
			// emit signal to kill server
			cancel()
		}
	}()

	app.logger.Info("Starting pprof server",
		slog.String("host", app.config.Server.PprofListen),
	)

	pprofSrv := &http.Server{
		Addr: app.config.Server.PprofListen,
	}
	go func() {
		pprofMux := http.NewServeMux()
		pprofMux.Handle("/debug/pprof/", http.DefaultServeMux)
		pprofSrv.Handler = pprofMux
		if err := pprofSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.logger.Error("error on pprof listenandserve", slog.String("err", err.Error()))
			// emit signal to kill server
			cancel()
		}
	}()

	var wg sync.WaitGroup
	wg.Go(func() {
		// wait for a signal
		<-ctx.Done()
		app.logger.Info("received shutdown signal")
		// create a new context for shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), configuration.Server.GracefulTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			app.logger.Error("error on srv shutdown", slog.String("err", err.Error()))
		}
		if err := pprofSrv.Shutdown(shutdownCtx); err != nil {
			app.logger.Error("error on pprofsrv shutdown", slog.String("err", err.Error()))
		}
	})
	wg.Wait()
	return nil
}
