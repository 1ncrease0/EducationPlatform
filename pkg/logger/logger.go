package logger

import (
	"log/slog"
	"os"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

type Log interface {
	Debug(message string, args ...interface{})
	Info(message string, args ...interface{})
	Warn(message string, args ...interface{})
	Error(message string, args ...interface{})
	ErrorErr(message string, err error, args ...interface{})
	Fatal(message string, args ...interface{})
	FatalErr(message string, err error, args ...interface{})
}

type Logger struct {
	logger *slog.Logger
}

func New(env string) *Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	default:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}
	l := &Logger{logger: log}
	return l
}

func (l *Logger) Debug(message string, args ...interface{}) {
	l.logger.Debug(message, args...)
}

func (l *Logger) Info(message string, args ...interface{}) {
	l.logger.Info(message, args...)
}

func (l *Logger) Warn(message string, args ...interface{}) {
	l.logger.Warn(message, args...)
}

func (l *Logger) Error(message string, args ...interface{}) {
	l.logger.Error(message, args...)
}

func (l *Logger) Fatal(message string, args ...interface{}) {
	l.logger.Error("FATAL: "+message, args...)
	os.Exit(1)
}

func (l *Logger) ErrorErr(message string, err error, args ...any) {
	allArgs := append(args, Err(err))
	l.logger.Error(message, allArgs...)
}

func (l *Logger) FatalErr(message string, err error, args ...any) {
	allArgs := append(args, Err(err))
	l.logger.Error("FATAL: "+message, allArgs...)
	os.Exit(1)
}

func Err(err error) slog.Attr {
	return slog.Attr{
		Key:   "error",
		Value: slog.StringValue(err.Error()),
	}
}
