package logger

import (
	"fmt"
	"log"
	"os"
)

type Logger interface {
    Info(message string)
    Warn(message string)
    Error(message string)
    Infof(format string, args ...any)
    Warnf(format string, args ...any)
    Errorf(format string, args ...any)
}

type ConsoleLogger struct {
    logger *log.Logger
}

func NewLogger() *ConsoleLogger {
    return &ConsoleLogger{
        logger: log.New(os.Stdout, "", log.LstdFlags),
    }
}

func (l *ConsoleLogger) Info(message string) {
    l.logger.Println("[INFO]", message)
}

func (l *ConsoleLogger) Warn(message string) {
    l.logger.Println("[WARN]", message)
}

func (l *ConsoleLogger) Error(message string) {
    l.logger.Println("[ERROR]", message)
}

func (l *ConsoleLogger) Infof(format string, args ...any) {
    l.Info(fmt.Sprintf(format, args...))
}

func (l *ConsoleLogger) Warnf(format string, args ...any) {
    l.Warn(fmt.Sprintf(format, args...))
}

func (l *ConsoleLogger) Errorf(format string, args ...any) {
    l.Error(fmt.Sprintf(format, args...))
}