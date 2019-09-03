package logger

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

type CustomFormatter logrus.TextFormatter

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	result := []byte{'['}
	switch entry.Level {
	case logrus.PanicLevel:
		result = append(result, "PANI"...)
	case logrus.FatalLevel:
		result = append(result, "FATA"...)
	case logrus.ErrorLevel:
		result = append(result, "ERRO"...)
	case logrus.WarnLevel:
		result = append(result, "WARN"...)
	case logrus.InfoLevel:
		result = append(result, "INFO"...)
	case logrus.DebugLevel:
		result = append(result, "DEBU"...)
	case logrus.TraceLevel:
		result = append(result, "TRAC"...)
	}
	result = append(result, ']')
	result = append(result, entry.Time.Format("2006-01-02T15:04:05.999Z07:00")...)
	result = append(result, ' ')
	result = append(result, entry.Message...)
	if len(entry.Data) > 0 {
		result = append(result, fmt.Sprintf(" Fields:%+v", entry.Data)...)
	}
	result = append(result, '\n')
	return result, nil
}
