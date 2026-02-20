package main

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/slackmgr/types"
)

type Logger struct {
	logger zerolog.Logger
}

func newLogger() *Logger {
	zerolog.TimestampFieldName = "timestamp"
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.DurationFieldInteger = true
	zerolog.DurationFieldUnit = time.Millisecond

	level := zerolog.DebugLevel
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	var loggerInstance zerolog.Logger

	output := zerolog.ConsoleWriter{Out: os.Stderr}
	log.Logger = log.Output(output)
	loggerInstance = zerolog.New(output).Level(level).With().Timestamp().Logger()

	return &Logger{logger: loggerInstance}
}

func (l *Logger) Debug(msg string) {
	l.logger.Debug().Msg(msg)
}

func (l *Logger) Debugf(format string, args ...any) {
	l.logger.Debug().Msgf(format, args...)
}

func (l *Logger) Info(msg string) {
	l.logger.Info().Msg(msg)
}

func (l *Logger) Infof(format string, args ...any) {
	l.logger.Info().Msgf(format, args...)
}

func (l *Logger) Error(msg string) {
	l.logger.Error().Msg(msg)
}

func (l *Logger) Errorf(format string, args ...any) {
	l.logger.Error().Msgf(format, args...)
}

func (l *Logger) WithField(key string, value any) types.Logger { //nolint:ireturn
	return &Logger{logger: l.logger.With().Any(key, value).Logger()}
}

func (l *Logger) WithFields(fields map[string]any) types.Logger { //nolint:ireturn
	return &Logger{logger: l.logger.With().Fields(fields).Logger()}
}
