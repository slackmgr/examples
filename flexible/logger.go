package main

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/slackmgr/examples/flexible/config"
	"github.com/slackmgr/types"
)

type Logger struct {
	logger zerolog.Logger
}

func newLogger(cfg *config.Config) *Logger {
	zerolog.TimestampFieldName = "timestamp"
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.DurationFieldInteger = true
	zerolog.DurationFieldUnit = time.Millisecond

	var level zerolog.Level

	if cfg.Verbose {
		level = zerolog.DebugLevel
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		level = zerolog.InfoLevel
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	var loggerInstance zerolog.Logger

	if cfg.LogJSON {
		loggerInstance = zerolog.New(os.Stderr).Level(level).With().Timestamp().Logger()
	} else {
		output := zerolog.ConsoleWriter{Out: os.Stderr}
		log.Logger = log.Output(output)
		loggerInstance = zerolog.New(output).Level(level).With().Timestamp().Logger()
	}

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
	switch v := value.(type) {
	case string:
		return &Logger{logger: l.logger.With().Str(key, v).Logger()}
	case int:
		return &Logger{logger: l.logger.With().Int(key, v).Logger()}
	case int32:
		return &Logger{logger: l.logger.With().Int32(key, v).Logger()}
	case int64:
		return &Logger{logger: l.logger.With().Int64(key, v).Logger()}
	case float64:
		return &Logger{logger: l.logger.With().Float64(key, v).Logger()}
	case bool:
		return &Logger{logger: l.logger.With().Bool(key, v).Logger()}
	case time.Time:
		return &Logger{logger: l.logger.With().Time(key, v).Logger()}
	case time.Duration:
		return &Logger{logger: l.logger.With().Dur(key, v).Logger()}
	default:
		return &Logger{logger: l.logger.With().Any(key, value).Logger()}
	}
}

func (l *Logger) WithFields(fields map[string]any) types.Logger { //nolint:ireturn
	return &Logger{logger: l.logger.With().Fields(fields).Logger()}
}
