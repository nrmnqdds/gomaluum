package logger

import (
	"log/slog"
	"os"

	slogmulti "github.com/samber/slog-multi"
	"go.opentelemetry.io/contrib/bridges/otelslog"
)

// instrumentationName identifies this application as the source of OTel log
// records. It shows up as the scope name in the logging backend (SigNoz).
const instrumentationName = "github.com/nrmnqdds/gomaluum"

// New builds the application logger. Records fan out to a human-readable text
// handler on stderr and to the OpenTelemetry logger bridge, which exports them
// over OTLP to the configured backend (SigNoz). The returned logger is also
// installed as the slog default, so package-level slog calls share it.
//
// InitLoggerProvider must be called before New so the otelslog bridge picks up
// the real LoggerProvider instead of the no-op global.
func New() *slog.Logger {
	console := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	otelHandler := otelslog.NewHandler(instrumentationName)

	logger := slog.New(slogmulti.Fanout(console, otelHandler))
	slog.SetDefault(logger)

	return logger
}
