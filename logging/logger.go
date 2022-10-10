package logging

import (
	"log"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	standardFields logrus.Fields
	logger         *logrus.Logger
)

func init() {
	logger = logrus.New()
	logger.Formatter = customFormatter{&logrus.TextFormatter{
		TimestampFormat: time.RFC3339Nano,
	}}
	logger.Out = os.Stdout
}

// SetStandardFields sets up the service name, version, hostname and pid fields
func SetStandardFields(service, version string) {
	//hostname, _ := os.Hostname()
	standardFields = logrus.Fields{
		"service": service,
		"version": version,
		//"hostname": hostname,
	}

	Info("Starting")
}

// UsePrettyPrint tells the logger to print in human readable format
func UsePrettyPrint() {
	logger.Formatter = customFormatter{&logrus.TextFormatter{
		FullTimestamp:    true,
		TimestampFormat:  time.RFC3339Nano,
		QuoteEmptyFields: true,
	}}
}

// WarnLogger creates a logger that can plug in to an HTTP server
func WarnLogger() (basicLogger *log.Logger, dispose func()) {
	w := logger.WriterLevel(logrus.WarnLevel)
	basicLogger = log.New(w, "", 0)
	dispose = func() {
		w.Close()
	}

	return
}

// ErrorLogger creates a logger that can plug in to an HTTP server
func ErrorLogger() (basicLogger *log.Logger, dispose func()) {
	w := logger.WriterLevel(logrus.ErrorLevel)
	basicLogger = log.New(w, "", 0)
	dispose = func() {
		w.Close()
	}

	return
}

func Debug(msg string) {
	logger.Debug(msg)
}

func Info(msg string) {
	logger.Info(msg)
}

func Infof(msg string, things ...any) {
	logger.Infof(msg, things...)
}

func Warn(msg string) {
	logger.Warn(msg)
}

func Error(msg string) {
	logger.Error(msg)
}

func Fatal(msg string) {
	logger.Fatal(msg)
}

// WithField returns a logger with the supplied field added to the standard fields
func WithField(key string, value any) *logrus.Entry {
	return logger.WithField(key, value)
}

// WithError returns a logger with the supplied error added to the logs
func WithError(err error) *logrus.Entry {
	return logger.WithField("error", err)
}

func SubscribeToErrorChan(errors <-chan error) {
	go func() {
		for {
			e := <-errors
			logger.Error(e.Error())
		}
	}()
}
