package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

//	func MyCaller(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
//		enc.AppendString(filepath.Base(caller.FullPath()))
//	}
//

// InitLog initializes the zap logger with a custom core and options
func InitLog(core zapcore.Core, options ...zap.Option) {
	log = zap.New(core, options...)

	// log.WithOptions(zap.WithCaller(true))
}

// Error logs an error message
func Error(msg string, fields ...zapcore.Field) {
	log.Error(msg, fields...) // Use zap.Logger's Error method
}

// Info logs an informational message
func Info(msg string, fields ...zapcore.Field) {
	log.Info(msg, fields...) // Use zap.Logger's Info method
}

// Warn logs a warning message
func Warn(msg string, fields ...zapcore.Field) {
	log.Warn(msg, fields...) // Use zap.Logger's Warn method
}

// Debug logs a debug message
func Debug(msg string, fields ...zapcore.Field) {
	log.Debug(msg, fields...) // Use zap.Logger's Debug method
}

func Fatal(msg string, fields ...zapcore.Field) {
	log.Fatal(msg, fields...)
}
