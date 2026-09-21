package logx

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

func createWriter(cfg *Config) io.Writer {
	var writers []io.Writer
	if cfg.Target == "console" || cfg.Target == "both" {
		if cfg.Format == "text" {
			writers = append(writers, zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
		} else {
			writers = append(writers, os.Stdout)
		}
	}
	if cfg.Target == "file" || cfg.Target == "both" {
		writers = append(writers, &lumberjack.Logger{
			Filename: cfg.FilePath, MaxSize: cfg.MaxSize, MaxBackups: cfg.MaxBackups,
			LocalTime: true, Compress: true,
		})
	}
	if len(writers) == 0 {
		return os.Stdout
	}
	return zerolog.MultiLevelWriter(writers...)
}
