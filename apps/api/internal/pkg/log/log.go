package log

import (
	"context"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/fatih/color"
)

const contextKeyRequestID = "request_id"

// WithRequestID adds request ID to context for logging
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, contextKeyRequestID, requestID)
}

// getRequestID retrieves request ID from context
func getRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(contextKeyRequestID).(string); ok {
		return id
	}
	return ""
}

// formatLog formats log message with optional request ID
func formatLog(level string, requestID string, format string, a ...interface{}) string {
	msg := fmt.Sprintf(format, a...)
	if requestID != "" {
		return fmt.Sprintf("[%s] [req_id=%s] %s", level, requestID, msg)
	}
	return fmt.Sprintf("[%s] %s", level, msg)
}

// Info log information
func Info(format string, a ...interface{}) {
	info := color.New(color.FgWhite, color.BgGreen).SprintFunc()
	fmt.Printf("%s ", info("[INFO] "))
	fmt.Printf(format, a...)
	fmt.Println()
}

// InfoWithContext logs information with context (includes request ID if available)
func InfoWithContext(ctx context.Context, format string, a ...interface{}) {
	requestID := getRequestID(ctx)
	msg := formatLog("INFO", requestID, format, a...)
	info := color.New(color.FgWhite, color.BgGreen).SprintFunc()
	fmt.Printf("%s ", info("[INFO] "))
	fmt.Println(msg)
}

// Warn log warning
func Warn(format string, a ...interface{}) {
	info := color.New(color.FgWhite, color.BgGreen).SprintFunc()
	fmt.Printf("%s ", info("[WARN] "))
	fmt.Printf(format, a...)
	fmt.Println()
}

// WarnWithContext logs warning with context (includes request ID if available)
func WarnWithContext(ctx context.Context, format string, a ...interface{}) {
	requestID := getRequestID(ctx)
	msg := formatLog("WARN", requestID, format, a...)
	warn := color.New(color.FgWhite, color.BgYellow).SprintFunc()
	fmt.Printf("%s ", warn("[WARN] "))
	fmt.Println(msg)
}

// Error log error
func Error(format string, a ...interface{}) {
	red := color.New(color.FgRed).SprintFunc()
	fmt.Printf("%s ", red("[Error]"))
	fmt.Printf(format, a...)
	fmt.Println()
}

// ErrorWithContext logs error with context (includes request ID if available)
func ErrorWithContext(ctx context.Context, format string, a ...interface{}) {
	requestID := getRequestID(ctx)
	msg := formatLog("ERROR", requestID, format, a...)
	red := color.New(color.FgRed).SprintFunc()
	fmt.Printf("%s ", red("[Error]"))
	fmt.Println(msg)
}

func InfoStruct(a ...interface{}) {
	spew.Sdump(a...)
}
