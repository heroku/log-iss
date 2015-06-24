package main

import "github.com/heroku/log-iss/Godeps/_workspace/src/github.com/Sirupsen/logrus"

// DefaultFieldsHook adds the Fields to all logs if the log entry does not
// already contain a field with the same key
type DefaultFieldsHook struct {
	Fields logrus.Fields
}

func (dfh *DefaultFieldsHook) Fire(entry *logrus.Entry) (err error) {
	for k, v := range dfh.Fields {
		if _, ok := entry.Data[k]; !ok {
			entry.Data[k] = v
		}
	}
	return nil
}

func (dfh *DefaultFieldsHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	}
}
