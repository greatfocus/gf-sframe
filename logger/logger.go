package logger

import (
	"log"
	"os"
	"path/filepath"
)

// Logger struct
type Logger struct {
	WarningLogger *log.Logger
	InfoLogger    *log.Logger
	ErrorLogger   *log.Logger
	DebugLogger   *log.Logger
}

// Start the server
func NewLogger(serviceName string) *Logger {
	path := os.Getenv("APP_PATH") + "/" + serviceName + ".txt"
	filepath := filepath.Clean(path)
	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	debugLogger := log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	infoLogger := log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	warningLogger := log.New(file, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	errorLogger := log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	l := &Logger{
		WarningLogger: infoLogger,
		InfoLogger:    warningLogger,
		ErrorLogger:   errorLogger,
		DebugLogger:   debugLogger,
	}
	return l
}
