package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type AppLogger struct {
	*zap.Logger
}

func New() *AppLogger {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	logger, _ := config.Build()

	return &AppLogger{
		Logger: logger,
	}
}

func (l *AppLogger) GetLogger() *zap.Logger {
	return l.Logger
}
