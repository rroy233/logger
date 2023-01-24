package logger

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"runtime"
)

func (lt *logType) Println(args ...interface{}) {
	subLoggerDefault := makeSubLoggerDefault()
	subLoggerJson := makeSubLoggerJson()
	switch lt.Level {
	case logrus.InfoLevel:
		subLoggerDefault.Infoln(args...)
		subLoggerJson.Infoln(args...)
	case logrus.DebugLevel:
		subLoggerDefault.Debugln(args...)
		subLoggerJson.Debugln(args...)
	case logrus.ErrorLevel:
		subLoggerDefault.Errorln(args...)
		subLoggerJson.Errorln(args...)
	case logrus.FatalLevel:
		subLoggerDefault.Fatalln(args...)
		subLoggerJson.Fatalln(args...)
	}
}
func (lt *logType) Printf(format string, args ...interface{}) {
	subLoggerDefault := makeSubLoggerDefault()
	subLoggerJson := makeSubLoggerJson()
	switch lt.Level {
	case logrus.InfoLevel:
		subLoggerDefault.Printf(format, args...)
		subLoggerJson.Printf(format, args...)
	case logrus.DebugLevel:
		subLoggerDefault.Debugf(format, args...)
		subLoggerJson.Debugf(format, args...)
	case logrus.ErrorLevel:
		subLoggerDefault.Errorf(format, args...)
		subLoggerJson.Errorf(format, args...)
	case logrus.FatalLevel:
		subLoggerDefault.Fatalf(format, args...)
		subLoggerJson.Fatalf(format, args...)
	}
}
func (lt *logType) Fatalln(args ...interface{}) {
	subLoggerDefault := makeSubLoggerDefault()
	subLoggerJson := makeSubLoggerJson()
	switch lt.Level {
	case logrus.ErrorLevel:
		subLoggerDefault.Fatalln(args...)
		subLoggerJson.Fatalln(args...)
	case logrus.FatalLevel:
		subLoggerDefault.Fatalln(args...)
		subLoggerJson.Fatalln(args...)
	}
}
func (lt *logType) Fatalf(format string, args ...interface{}) {
	subLoggerDefault := makeSubLoggerDefault()
	subLoggerJson := makeSubLoggerJson()
	switch lt.Level {
	case logrus.ErrorLevel:
		subLoggerDefault.Fatalf(format, args...)
		subLoggerJson.Fatalf(format, args...)
	case logrus.FatalLevel:
		subLoggerDefault.Fatalf(format, args...)
		subLoggerJson.Fatalf(format, args...)
	}
}

func makeSubLoggerDefault() *logrus.Entry {
	_, file, line, _ := runtime.Caller(2)
	pwd, _ := os.Getwd()
	rel, err := filepath.Rel(pwd, file)
	if err != nil {
		rel = file
	} else {
		rel = "./" + rel
	}

	return defaultLogger.WithFields(logrus.Fields{
		"Caller": fmt.Sprintf("%s:%d", rel, line),
	})
}

func makeSubLoggerJson() *logrus.Entry {
	_, file, line, _ := runtime.Caller(2)
	pwd, _ := os.Getwd()
	rel, err := filepath.Rel(pwd, file)
	if err != nil {
		rel = file
	} else {
		rel = "./" + rel
	}

	return jsonLogger.WithFields(logrus.Fields{
		"Caller": fmt.Sprintf("%s:%d", rel, line),
	})
}
