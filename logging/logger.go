package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"proxy/config"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

type Logger struct {
	writer io.Writer
	level  LogLevel
	format string
}

type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

func NewLogger(cfg *config.LoggingConfig) *Logger {
	level := parseLogLevel(cfg.Level)

	return &Logger{
		writer: os.Stdout,
		level:  level,
		format: cfg.Format,
	}
}

func parseLogLevel(levelStr string) LogLevel {
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN", "WARNING":
		return WARN
	case "ERROR":
		return ERROR
	default:
		return INFO
	}
}

func (l *Logger) Debug(message string, fields map[string]interface{}) {
	l.log(DEBUG, message, fields)
}

func (l *Logger) Info(message string, fields map[string]interface{}) {
	l.log(INFO, message, fields)
}

func (l *Logger) Warn(message string, fields map[string]interface{}) {
	l.log(WARN, message, fields)
}

func (l *Logger) Error(message string, fields map[string]interface{}) {
	l.log(ERROR, message, fields)
}

func (l *Logger) log(level LogLevel, message string, fields map[string]interface{}) {
	if level < l.level {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level.String(),
		Message:   message,
		Fields:    fields,
	}

	var output string
	if l.format == "json" {
		jsonBytes, err := json.Marshal(entry)
		if err != nil {
			output = fmt.Sprintf("Failed to marshal log entry: %v", err)
		} else {
			output = string(jsonBytes)
		}
	} else {
		fieldsStr := ""
		if len(fields) > 0 {
			var pairs []string
			for k, v := range fields {
				pairs = append(pairs, fmt.Sprintf("%s=%v", k, v))
			}
			fieldsStr = " " + strings.Join(pairs, " ")
		}
		output = fmt.Sprintf("[%s] %s %s%s", entry.Timestamp, entry.Level, message, fieldsStr)
	}

	fmt.Fprintln(l.writer, output)
}

func (l *Logger) SetOutput(w io.Writer) {
	l.writer = w
}

func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}