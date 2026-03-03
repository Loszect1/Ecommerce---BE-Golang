package logger

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"
)

// Logger is a very small structured logger interface to avoid tying the app to a specific logging library.
type Logger interface {
	Info(msg string, fields map[string]any)
	Error(msg string, err error, fields map[string]any)
}

type stdLogger struct {
	base  *log.Logger
	level string
}

// New creates a new Logger writing JSON lines to stdout.
func New() Logger {
	return &stdLogger{
		base:  log.New(os.Stdout, "", 0),
		level: "info",
	}
}

type entry struct {
	Level   string         `json:"level"`
	Time    time.Time      `json:"time"`
	Message string         `json:"message"`
	Error   string         `json:"error,omitempty"`
	Fields  map[string]any `json:"fields,omitempty"`
}

func (l *stdLogger) Info(msg string, fields map[string]any) {
	l.write("info", msg, "", fields)
}

func (l *stdLogger) Error(msg string, err error, fields map[string]any) {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	l.write("error", msg, errMsg, fields)
}

func (l *stdLogger) write(level, msg, errMsg string, fields map[string]any) {
	if fields == nil {
		fields = make(map[string]any)
	}
	out := entry{
		Level:   level,
		Time:    time.Now().UTC(),
		Message: msg,
		Error:   errMsg,
		Fields:  fields,
	}
	data, err := json.Marshal(out)
	if err != nil {
		// Fallback to plain text if JSON marshaling fails.
		l.base.Printf(`{"level":"error","message":"failed to marshal log entry","error":"%v"}`, err)
		return
	}
	l.base.Println(string(data))
}

// WithContext is a helper to extract request-scoped fields from context.
func WithContext(ctx context.Context, baseFields map[string]any) map[string]any {
	if baseFields == nil {
		baseFields = make(map[string]any)
	}
	if reqID, ok := ctx.Value(ctxKeyRequestID{}).(string); ok && reqID != "" {
		baseFields["request_id"] = reqID
	}
	return baseFields
}

// ctxKeyRequestID is used for storing request IDs in context.
type ctxKeyRequestID struct{}

// ContextWithRequestID stores a request ID in the given context.
func ContextWithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyRequestID{}, id)
}

