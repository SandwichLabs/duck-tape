package cmd

import (
	"log/slog"
	"os"
)

func SetJsonHandler(logLevel *slog.LevelVar) {
	h := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(h)
}

func SetLogLevel(stringLevel string, logLevel *slog.LevelVar) {
	switch stringLevel {
	case "debug":
		logLevel.Set(slog.LevelDebug)
	case "info":
		logLevel.Set(slog.LevelInfo)
	case "warn":
		logLevel.Set(slog.LevelWarn)
	case "error":
		logLevel.Set(slog.LevelError)
	default:
		logLevel.Set(slog.LevelInfo)
	}
}
