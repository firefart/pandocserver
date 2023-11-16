package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	_ "go.uber.org/automaxprocs"
)

type Logger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Info(args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	SetOutput(io.Writer)
	SetLevel(logrus.Level)
}

type application struct {
	logger         Logger
	pandocPath     string
	pandocDataDir  string
	commandTimeout time.Duration
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func lookupEnvOrString(log Logger, key string, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

func lookupEnvOrBool(log Logger, key string, defaultVal bool) bool {
	if val, ok := os.LookupEnv(key); ok {
		v, err := strconv.ParseBool(val)
		if err != nil {
			log.Errorf("lookupEnvOrBool[%s]: %v", key, err)
			return defaultVal
		}
		return v
	}
	return defaultVal
}

func lookupEnvOrDuration(log Logger, key string, defaultVal time.Duration) time.Duration {
	if val, ok := os.LookupEnv(key); ok {
		v, err := time.ParseDuration(val)
		if err != nil {
			log.Errorf("lookupEnvOrDuration[%s]: %v", key, err)
			return defaultVal
		}
		return v
	}
	return defaultVal
}

func main() {
	app := &application{
		logger: logrus.New(),
	}

	var host string
	var wait time.Duration
	var debugOutput bool
	flag.StringVar(&host,
		"host",
		lookupEnvOrString(app.logger, "PANDOC_HOST", ":8080"),
		"IP and Port to bind to. Can also be set through the PANDOC_HOST environment variable.")
	flag.BoolVar(&debugOutput,
		"debug",
		lookupEnvOrBool(app.logger, "PANDOC_DEBUG", false),
		"Enable DEBUG mode. Can also be set through the PANDOC_DEBUG environment variable.")
	flag.DurationVar(&wait,
		"graceful-timeout",
		lookupEnvOrDuration(app.logger, "PANDOC_GRACEFUL_TIMEOUT", 5*time.Second),
		"the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m. Can also be set through the PANDOC_GRACEFUL_TIMEOUT environment variable.")
	flag.DurationVar(&app.commandTimeout,
		"command-timeout",
		lookupEnvOrDuration(app.logger, "PANDOC_COMMAND_TIMEOUT", 1*time.Minute),
		"the timeout for the conversion command. Can also be set through the PANDOC_COMMAND_TIMEOUT environment variable.")
	flag.StringVar(&app.pandocPath,
		"pandoc-path",
		lookupEnvOrString(app.logger, "PANDOC_PATH", "/usr/local/bin/pandoc"),
		"The path of the pandoc binary. Can also be set through the PANDOC_PATH environment variable.")
	flag.StringVar(&app.pandocDataDir,
		"pandoc-data-dir",
		lookupEnvOrString(app.logger, "PANDOC_DATA_DIR", "/.pandoc"),
		"The pandoc data dir containing the templates. Can also be set through the PANDOC_DATA_DIR environment variable.")
	flag.Parse()

	gin.SetMode(gin.ReleaseMode)

	app.logger.SetOutput(os.Stdout)
	app.logger.SetLevel(logrus.InfoLevel)
	if debugOutput {
		gin.SetMode(gin.DebugMode)
		app.logger.SetLevel(logrus.DebugLevel)
		app.logger.Debug("DEBUG mode enabled")
	}

	app.logger.Info("Starting pandocserver with the following parameters:")
	app.logger.Infof("host: %s", host)
	app.logger.Infof("debug: %t", debugOutput)
	app.logger.Infof("graceful timeout: %s", wait)
	app.logger.Infof("command timeout: %s", app.commandTimeout)
	app.logger.Infof("pandoc path: %s", app.pandocPath)
	app.logger.Infof("pandoc data dir: %s", app.pandocDataDir)

	srv := &http.Server{
		Addr:    host,
		Handler: app.routes(),
	}
	app.logger.Infof("Starting server on %s", host)

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			app.logger.Error(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	<-c
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		app.logger.Error(err)
	}
	app.logger.Info("shutting down")
	os.Exit(0)
}

func errorJson(errorText string) gin.H {
	return gin.H{
		"error": errorText,
	}
}

func (app *application) routes() http.Handler {
	r := gin.Default()
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, errorJson("Page not found"))
	})
	r.POST("/convert", func(c *gin.Context) {
		var json struct {
			Input     []byte            `json:"input" binding:"required"`
			Resources map[string][]byte `json:"resources"`
			Template  string            `json:"template"`
		}

		if err := c.ShouldBindJSON(&json); err != nil {
			app.logger.Errorf("[CONVERT]: %v", err)
			c.AbortWithStatusJSON(http.StatusBadRequest, errorJson("invalid request"))
			return
		}

		bin, err := app.convert(c.Request.Context(), json.Input, json.Resources, json.Template)
		if err != nil {
			app.logger.Errorf("[CONVERT]: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, errorJson("error converting markdown"))
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"content": bin,
		})
	})
	return r
}

func (app *application) convert(ctx context.Context, inputFile []byte, resources map[string][]byte, template string) ([]byte, error) {
	tmpdir := path.Join(os.TempDir(), fmt.Sprintf("pandocserver_%s", randStringRunes(10)))
	if err := os.Mkdir(tmpdir, 0750); err != nil {
		return nil, fmt.Errorf("could not create dir %q: %w", tmpdir, err)
	}
	defer os.RemoveAll(tmpdir)

	inputFileName := filepath.Join(tmpdir, fmt.Sprintf("%s.md", randStringRunes(10)))
	if err := os.WriteFile(inputFileName, inputFile, 0600); err != nil {
		return nil, fmt.Errorf("could not create inputfile: %w", err)
	}

	outputDir := path.Join(tmpdir, "output")
	if err := os.Mkdir(outputDir, 0750); err != nil {
		return nil, fmt.Errorf("could not create output directory: %w", err)
	}
	outputFilename := filepath.Join(outputDir, fmt.Sprintf("%s.pdf", randStringRunes(10)))

	// we need to set --data-dir as you need to have a .pandoc folder in your home
	// and we run as a different user than the docker image defaults to (which is root)
	// so we set the global data-dir to make sure the template can be found
	args := []string{
		inputFileName,
		fmt.Sprintf("--output=%s", outputFilename),
		fmt.Sprintf("--data-dir=%s", app.pandocDataDir),
		"--from=markdown+yaml_metadata_block+raw_html+emoji",
		"--sandbox",
	}

	// the pdf processor does not seem to respect the --resource-path
	// parameter so we need to store them in the root so that referencing
	// them works correctly
	if len(resources) > 0 {
		for fname, content := range resources {
			cleaned := filepath.Clean(filepath.Join(tmpdir, fname))
			// actual dir, check and create
			if !strings.HasPrefix(cleaned, tmpdir) {
				return nil, fmt.Errorf("tried to access file %s which is outside the current working directory (%s)", cleaned, tmpdir)
			}
			if err := os.MkdirAll(filepath.Dir(cleaned), 0750); err != nil {
				return nil, fmt.Errorf("could not create dir path for %s: %w", cleaned, err)
			}
			if err := os.WriteFile(cleaned, content, 0600); err != nil {
				return nil, fmt.Errorf("could not create resource file %s: %w", cleaned, err)
			}
			app.logger.Debugf("created resource file %s", cleaned)
		}
	}

	if template != "" {
		args = append(args, fmt.Sprintf("--template=%s", template))
	}

	commandCtx, cancel := context.WithTimeout(ctx, app.commandTimeout)
	defer cancel()

	app.logger.Debugf("going to call pandoc with the following args: %v", args)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.CommandContext(commandCtx, app.pandocPath, args...)
	cmd.Dir = tmpdir
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		app.killProcessIfRunning(cmd)
		return nil, fmt.Errorf("could not execute command %w: %s", err, stderr.String())
	}

	app.logger.Debugf("STDOUT: %s", out.String())
	app.logger.Debugf("STDERR: %s", stderr.String())

	app.killProcessIfRunning(cmd)

	content, err := os.ReadFile(outputFilename)
	if err != nil {
		return nil, fmt.Errorf("could not read output file: %w", err)
	}

	return content, nil
}

func (app *application) killProcessIfRunning(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	if err := cmd.Process.Release(); err != nil {
		return
	}
	if err := cmd.Process.Kill(); err != nil {
		return
	}
}
