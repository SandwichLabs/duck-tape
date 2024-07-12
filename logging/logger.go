package logging

import (
	"log/slog"
	"os"
)

func New(logLevel *slog.LevelVar) *slog.Logger {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	return logger
}
