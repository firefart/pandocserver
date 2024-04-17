package main

import (
	"log/slog"
	"os"
	"strings"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
)

func newLogger(debugMode, jsonOutput bool) *slog.Logger {
	w := os.Stdout
	var level = new(slog.LevelVar)
	level.Set(slog.LevelInfo)

	var replaceFunc func(groups []string, a slog.Attr) slog.Attr = nil
	if debugMode {
		level.Set(slog.LevelDebug)
		// add source file information
		wd, err := os.Getwd()
		if err != nil {
			panic("unable to determine working directory")
		}
		replaceFunc = func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				source := a.Value.Any().(*slog.Source)
				// remove current working directory and only leave the relative path to the program
				if file, ok := strings.CutPrefix(source.File, wd); ok {
					source.File = file
				}
			}
			return a
		}
	}

	var handler slog.Handler
	if jsonOutput {
		handler = slog.NewJSONHandler(w, &slog.HandlerOptions{
			Level:       level,
			AddSource:   debugMode,
			ReplaceAttr: replaceFunc,
		})
	} else {
		textOptions := &tint.Options{
			Level:       level,
			NoColor:     !isatty.IsTerminal(w.Fd()),
			AddSource:   debugMode,
			ReplaceAttr: replaceFunc,
		}
		handler = tint.NewHandler(w, textOptions)
	}
	return slog.New(handler)
}
