package utils

import (
	"runtime"

	"github.com/nrmnqdds/gomaluum/pkg/logger"
)

func CatchPanic(funcName string, _ ...map[string]any) {
	if err := recover(); err != nil {
		stack := make([]byte, 8096)
		stack = stack[:runtime.Stack(stack, false)]
		logger := logger.New()
		logger.Sugar().Debugf("recovered from panic -%s", funcName)
	}
}
