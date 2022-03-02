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
	fileName := os.Getenv("APP_PATH") + "/" + serviceName + ".txt"
	path := filepath.Clean(fileName)
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal(err)
	}

	debugLogger := log.New(file, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	infoLogger := log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	warningLogger := log.New(file, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	errorLogger := log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	l := &Logger{
		WarningLogger: warningLogger,
		InfoLogger:    infoLogger,
		ErrorLogger:   errorLogger,
		DebugLogger:   debugLogger,
	}
	return l
}
