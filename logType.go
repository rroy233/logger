package logger

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"runtime"
)

func (lt *logType) Println(args ...interface{}) {
	subLogger := makeSubLogger()
	switch lt.Level {
	case logrus.InfoLevel:
		subLogger.Infoln(args...)
	case logrus.DebugLevel:
		subLogger.Debugln(args...)
	case logrus.ErrorLevel:
		subLogger.Errorln(args...)
	case logrus.FatalLevel:
		subLogger.Fatalln(args...)
	}
}
func (lt *logType) Printf(format string, args ...interface{}) {
	subLogger := makeSubLogger()
	switch lt.Level {
	case logrus.InfoLevel:
		subLogger.Printf(format, args...)
	case logrus.DebugLevel:
		subLogger.Debugf(format, args...)
	case logrus.ErrorLevel:
		subLogger.Errorf(format, args...)
	case logrus.FatalLevel:
		subLogger.Fatalf(format, args...)
	}
}
func (lt *logType) Fatalln(args ...interface{}) {
	subLogger := makeSubLogger()
	switch lt.Level {
	case logrus.ErrorLevel:
		subLogger.Fatalln(args...)
	case logrus.FatalLevel:
		subLogger.Fatalln(args...)
	}
}
func (lt *logType) Fatalf(format string, args ...interface{}) {
	subLogger := makeSubLogger()
	switch lt.Level {
	case logrus.ErrorLevel:
		subLogger.Fatalf(format, args...)
	case logrus.FatalLevel:
		subLogger.Fatalf(format, args...)
	}
}

func makeSubLogger() *logrus.Entry {
	ptr, file, line, _ := runtime.Caller(2)
	return sysLogger.WithFields(logrus.Fields{
		"Caller": fmt.Sprintf("%s:%d(%v)", file, line, ptr),
	})
}
