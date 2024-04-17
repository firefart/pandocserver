package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
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
		fmt.Sprintf("--data-dir=%s", app.config.PandocDataDir),
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
			app.logger.Debug("created resource file", slog.String("filename", cleaned))
		}
	}

	if template != "" {
		args = append(args, fmt.Sprintf("--template=%s", template))
	}

	commandCtx, cancel := context.WithTimeout(ctx, app.config.CommandTimeout)
	defer cancel()

	app.logger.Debug("going to call pandoc", slog.String("args", strings.Join(args, ",")))

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.CommandContext(commandCtx, app.config.PandocPath, args...)
	cmd.Dir = tmpdir
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		app.killProcessIfRunning(cmd)
		return nil, fmt.Errorf("could not execute command %w: %s", err, stderr.String())
	}

	app.logger.Debug("STDOUT", slog.String("out", out.String()))
	app.logger.Debug("STDERR", slog.String("out", stderr.String()))

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
