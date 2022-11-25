package logger

import "github.com/sirupsen/logrus"

func (lt *logType) Println(args ...interface{}) {
	switch lt.Level {
	case logrus.InfoLevel:
		sysLogger.Infoln(args...)
	case logrus.DebugLevel:
		sysLogger.Debugln(args...)
	case logrus.ErrorLevel:
		sysLogger.Errorln(args...)
	case logrus.FatalLevel:
		sysLogger.Fatalln(args...)
	}
}
func (lt *logType) Printf(format string, args ...interface{}) {
	switch lt.Level {
	case logrus.InfoLevel:
		sysLogger.Printf(format, args...)
	case logrus.DebugLevel:
		sysLogger.Debugf(format, args...)
	case logrus.ErrorLevel:
		sysLogger.Errorf(format, args...)
	case logrus.FatalLevel:
		sysLogger.Fatalf(format, args...)
	}
}
func (lt *logType) Fatalln(args ...interface{}) {
	switch lt.Level {
	case logrus.ErrorLevel:
		sysLogger.Fatalln(args...)
	case logrus.FatalLevel:
		sysLogger.Fatalln(args...)
	}
}
func (lt *logType) Fatalf(format string, args ...interface{}) {
	switch lt.Level {
	case logrus.ErrorLevel:
		sysLogger.Fatalf(format, args...)
	case logrus.FatalLevel:
		sysLogger.Fatalf(format, args...)
	}
}
