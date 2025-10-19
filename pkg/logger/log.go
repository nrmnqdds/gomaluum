package logger

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/sdk/log"
	"go.uber.org/zap"
)

type AppLogger struct {
	*zap.Logger
}

func New() *AppLogger {

	otlpEndpoint := os.Getenv("OTLP_LOG_ENDPOINT")

	ctx := context.Background()
	exporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint(otlpEndpoint),
		otlploghttp.WithInsecure(),
	)
	if err != nil {
		fmt.Println(err)
	}

	processor := log.NewBatchProcessor(exporter)

	provider := log.NewLoggerProvider(
		log.WithProcessor(processor),
	)

	defer func() {
		err := provider.Shutdown(context.Background())
		if err != nil {
			fmt.Println(err)
		}
	}()

	logger := zap.New(otelzap.NewCore("gomaluum-logs", otelzap.WithLoggerProvider(provider)))
	defer logger.Sync()

	return &AppLogger{
		Logger: logger,
	}
}

func (l *AppLogger) GetLogger() *zap.Logger {
	return l.Logger
}
