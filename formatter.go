package logger

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"strings"
)

func (f *defaultFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	data := fmt.Sprintf(
		"%s %s [%s] %s\n",
		strings.ToUpper(entry.Level.String()),
		entry.Time.Format("2006-01-02 15:04:05"),
		entry.Data["Caller"],
		entry.Message,
	)
	return []byte(data), nil
}
