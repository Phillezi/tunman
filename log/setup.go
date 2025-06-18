package log

import (
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Setup() (*zap.Logger, error) {
	profile := viper.GetString("profile")
	level := viper.GetString("loglevel")
	showStackTrace := viper.GetBool("stacktrace")
	enableTimestamp := viper.GetBool("log.timestamp")

	var zapLevel zapcore.Level
	if err := zapLevel.Set(level); err != nil {
		zap.L().Warn("Invalid log level, falling back to INFO", zap.String("loglevel", level))
		zapLevel = zapcore.InfoLevel
	}

	var cfg zap.Config
	if profile == "prod" || profile == "production" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
		encoderConfig := zapcore.EncoderConfig{
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalColorLevelEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}
		if enableTimestamp {
			encoderConfig.TimeKey = "ts"
			encoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
		}
		cfg.EncoderConfig = encoderConfig
	}

	cfg.Level = zap.NewAtomicLevelAt(zapLevel)
	cfg.DisableStacktrace = !showStackTrace

	logger, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	zap.ReplaceGlobals(logger)
	return logger, nil
}
