package logger

import (
	"log"
	"os"
	"path/filepath"
	"time"

	gfcron "github.com/greatfocus/gf-cron"
)

// Logger struct
type Logger struct {
	WarningLogger *log.Logger
	InfoLogger    *log.Logger
	ErrorLogger   *log.Logger
	DebugLogger   *log.Logger
}

// NewLogger Start the server
func NewLogger(serviceName string, cron *gfcron.Cron) *Logger {
	debugLogger := loggerFile(serviceName, "DEBUG", cron)
	infoLogger := loggerFile(serviceName, "INFO", cron)
	warningLogger := loggerFile(serviceName, "WARNING", cron)
	errorLogger := loggerFile(serviceName, "ERROR", cron)

	l := &Logger{
		WarningLogger: warningLogger,
		InfoLogger:    infoLogger,
		ErrorLogger:   errorLogger,
		DebugLogger:   debugLogger,
	}
	return l
}

// loggerFile creates new logger and adds logs rotate
func loggerFile(serviceName, level string, cron *gfcron.Cron) *log.Logger {
	file := getFile(serviceName)
	logger := log.New(file, level+": ", log.Ldate|log.Ltime|log.Lshortfile)
	logger.SetOutput(file)
	cron.MustAddJob("*/2 * * * *", logRotate, serviceName, logger)
	return logger
}

// getFile returns os file
func getFile(serviceName string) *os.File {
	currentTime := time.Now()
	date := currentTime.Format("2006-01-02")
	fileName := os.Getenv("APP_PATH") + "/logs/" + serviceName + "-" + date + ".log"
	path := filepath.Clean(fileName)
	file, err := createFile(path)
	if err != nil {
		log.Fatal(err)
	}
	return file
}

// logRotate rotate logger at based on the scheduler
func logRotate(serviceName string, logger *log.Logger) {
	file := getFile(serviceName)
	logger.SetOutput(file)
}

// create file
func createFile(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0770); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal(err)
	}
	return file, nil
}
