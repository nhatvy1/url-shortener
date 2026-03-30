package logger

import "go.uber.org/zap"

type Logger struct {
	zap *zap.Logger
}

func New(log *zap.Logger) *Logger {
	return &Logger{zap: log}
}

func (l *Logger) WithContext(ctx string) *Logger {
	return &Logger{
		zap: l.zap.With(zap.String("context", ctx)),
	}
}

func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.zap.Info(msg, fields...)
}

func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.zap.Error(msg, fields...)
}

func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.zap.Debug(msg, fields...)
}
