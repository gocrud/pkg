package logx

import (
	"time"

	"github.com/rs/zerolog"
)

func NewInstance(cfg *Config) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	return zerolog.New(createWriter(cfg)).With().
		Timestamp().
		CallerWithSkipFrameCount(2).
		Logger().
		Level(level)
}
