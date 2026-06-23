package utils

import (
	"log/slog"
	"runtime"
)

func CatchPanic(funcName string, _ ...map[string]any) {
	if err := recover(); err != nil {
		stack := make([]byte, 8096)
		stack = stack[:runtime.Stack(stack, false)]
		slog.Error("recovered from panic", "func", funcName, "panic", err, "stack", string(stack))
	}
}
