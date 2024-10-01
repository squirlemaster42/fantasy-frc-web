package logging

import (
	"errors"
	"fmt"
	"time"
)

type LogWriter interface {
    WriteMessage(message string) error
}

type ConsoleLogger struct {}

func (c *ConsoleLogger) WriteMessage(message string) error {
    fmt.Println(message)
    return nil
}

type TimestampedLogger struct {}

func (t *TimestampedLogger) WriteMessage(message string) error {
    fmt.Printf("%s - %s\n", time.Now().Local().Format(time.StampMilli), message)
    return nil
}

type StructuredLogger struct {}

func (s *StructuredLogger) WriteMessage(message string) error {
    return nil
}

type Logger struct {
    messages chan string
    writer LogWriter
    running bool
}

func NewLogger(logWriter LogWriter) *Logger {
    return &Logger{
        messages: make(chan string, 100),
        writer: logWriter,
        running: false,
    }
}

func (l *Logger) Log(message string) {
    l.messages <- message
}

func (l *Logger) Start() error {
    if l.running {
        return errors.New("Logger already started")
    }
    l.running = true
    go func() {
        l.writer.WriteMessage("Starting Logger")
        for l.running {
            msg := <-l.messages
            l.writer.WriteMessage(msg)
        }
    }()
    return nil
}

func (l *Logger) Stop() error {
    if !l.running {
        return errors.New("Logger already stopped")
    }
    l.running = false
    return nil
}
