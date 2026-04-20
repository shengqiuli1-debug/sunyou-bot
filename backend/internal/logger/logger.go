package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func New(env, logDir string) (*slog.Logger, func()) {
	lvl := slog.LevelInfo
	if env == "development" {
		lvl = slog.LevelDebug
	}

	writers := []io.Writer{os.Stdout}
	var setupErr string
	var file *os.File

	if dir := strings.TrimSpace(logDir); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			setupErr = "create log dir failed: " + err.Error()
		} else {
			path := filepath.Join(dir, "app.log")
			f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
			if err != nil {
				setupErr = "open log file failed: " + err.Error()
			} else {
				file = f
				writers = append(writers, f)
			}
		}
	}

	out := io.MultiWriter(writers...)
	h := slog.NewTextHandler(out, &slog.HandlerOptions{Level: lvl})
	log := slog.New(h)
	if setupErr != "" {
		log.Warn("file logger not ready, fallback to stdout only", "error", setupErr, "log_dir", logDir)
	}

	cleanup := func() {
		if file != nil {
			_ = file.Close()
		}
	}

	return log, cleanup
}
