package logger

import (
	"log/slog"
	"runtime"
	"time"
)

func GetSource(skip int) slog.Attr {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return slog.Attr{}
	}

	fn := runtime.FuncForPC(pc)
	funcName := "unknown"
	if fn != nil {
		funcName = fn.Name()
	}

	return slog.Group("source",
		"function", funcName,
		"file", file,
		"line", line,
	)
}

func HttpAttrs(method, route string, status int, dur time.Duration) slog.Attr {
	sec := float64(dur) / float64(time.Second)
	return slog.Group("",
		"http.request.method", method,
		"http.route", route,
		"http.response.status_code", status,
		"http.server.request.duration", sec)

}
