// Copyright (C) 2018 Cranky Kernel
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package log

import (
	"github.com/sirupsen/logrus"
	"path/filepath"
	"fmt"
	"runtime"
	"encoding/json"
	"os"
	"log"
	"golang.org/x/crypto/ssh/terminal"
)

type Fields = logrus.Fields
type LogLevel = logrus.Level

const (
	LogLevelDebug LogLevel = logrus.DebugLevel
	LogLevelInfo  LogLevel = logrus.InfoLevel
)

var logLevel = LogLevelInfo

type FileOutputHook struct {
	file      *os.File
	formatter logrus.Formatter
}

func NewFileOutputHook(filename string) *FileOutputHook {
	var err error
	var file *os.File

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		file, err = os.Create(filename)
	} else {
		file, err = os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
	}
	if err != nil {
		log.Fatal("Failed to open %s for logging: %v", filename, err)
	}

	return &FileOutputHook{
		file:      file,
		formatter: &logrus.JSONFormatter{},
	}
}

func (h *FileOutputHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *FileOutputHook) Fire(entry *logrus.Entry) error {
	line, err := h.formatter.Format(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to format log message: %v\n", err)
		return err
	}
	h.file.Write(line)
	return nil
}

func init() {
	formatter := logrus.TextFormatter{}
	formatter.DisableTimestamp = false
	formatter.FullTimestamp = true
	formatter.TimestampFormat = "2006-01-02 15:04:05.999"
	logrus.SetLevel(logLevel)

	if !terminal.IsTerminal(int(os.Stderr.Fd())) {
		formatter.DisableColors = true
	} else if os.Getenv("SHELL") == "" {
		// The idea here is to disable colour if running in something like
		// cmd.exe, as it doesn't seem to handle colour.
		formatter.DisableColors = true
	}

	logrus.SetFormatter(&formatter)
}

func AddHook(hook logrus.Hook) {
	logrus.AddHook(hook)
}

func SetLevel(level LogLevel) {
	logLevel = level
	logrus.SetLevel(logLevel)
}

func Printf(format string, v ...interface{}) {
	if logLevel == logrus.DebugLevel {
		_, filename, line, _ := runtime.Caller(1)
		logrus.WithFields(Fields{
			"_source": fmt.Sprintf("%s:%d", filepath.Base(filename), line),
		}).Infof(format, v...)
	} else {
		logrus.Infof(format, v...)
	}
}

func Debugf(format string, v ...interface{}) {
	if logLevel == logrus.DebugLevel {
		_, filename, line, _ := runtime.Caller(1)
		logrus.WithFields(Fields{
			"_source": fmt.Sprintf("%s:%d", filepath.Base(filename), line),
		}).Debugf(format, v...)
	} else {
		logrus.Debugf(format, v...)
	}
}

func Infof(format string, v ...interface{}) {
	if logLevel == logrus.DebugLevel {
		_, filename, line, _ := runtime.Caller(1)
		logrus.WithFields(Fields{
			"_source": fmt.Sprintf("%s:%d", filepath.Base(filename), line),
		}).Infof(format, v...)
	} else {
		logrus.Infof(format, v...)
	}
}

func Info(v ...interface{}) {
	if logLevel == logrus.DebugLevel {
		_, filename, line, _ := runtime.Caller(1)
		logrus.WithFields(Fields{
			"_source": fmt.Sprintf("%s:%d", filepath.Base(filename), line),
		}).Info(v...)
	} else {
		logrus.Info(v...)
	}
}

func Errorf(format string, v ...interface{}) {
	if logLevel == logrus.DebugLevel {
		_, filename, line, _ := runtime.Caller(1)
		logrus.WithFields(Fields{
			"_source": fmt.Sprintf("%s:%d", filepath.Base(filename), line),
		}).Errorf(format, v...)
	} else {
		logrus.Errorf(format, v...)
	}
}

func Fatalf(format string, v ...interface{}) {
	if logLevel == logrus.DebugLevel {
		_, filename, line, _ := runtime.Caller(1)
		logrus.WithFields(Fields{
			"_source": fmt.Sprintf("%s:%d", filepath.Base(filename), line),
		}).Fatalf(format, v...)
	} else {
		logrus.Fatalf(format, v...)
	}
}

func Println(v ...interface{}) {
	if logLevel == logrus.DebugLevel {
		_, filename, line, _ := runtime.Caller(1)
		logrus.WithFields(Fields{
			"_source": fmt.Sprintf("%s:%d", filepath.Base(filename), line),
		}).Info(v...)
	} else {
		logrus.Info(v...)
	}
}

func Fatal(v ...interface{}) {
	if logLevel == logrus.DebugLevel {
		_, filename, line, _ := runtime.Caller(1)
		logrus.WithFields(Fields{
			"_source": fmt.Sprintf("%s:%d", filepath.Base(filename), line),
		}).Fatal(v...)
	} else {
		logrus.Fatal(v...)
	}
}

func WithFields(fields Fields) *logrus.Entry {
	if logLevel == logrus.DebugLevel {
		if fields["_source"] == nil {
			_, filename, line, _ := runtime.Caller(1)
			fields["_source"] = fmt.Sprintf("%s:%d", filepath.Base(filename), line)
		}
	}
	return logrus.WithFields(fields)
}

func WithField(field string, value interface{}) *logrus.Entry {
	if logLevel == logrus.DebugLevel {
		fields := logrus.Fields{
			field: value,
		}
		if fields["_source"] == nil {
			_, filename, line, _ := runtime.Caller(1)
			fields["_source"] = fmt.Sprintf("%s:%d", filepath.Base(filename), line)
		}
		return logrus.WithFields(fields)
	}
	return logrus.WithField(field, value)
}

func WithError(err error) *logrus.Entry {
	if logLevel == logrus.DebugLevel {
		_, filename, line, _ := runtime.Caller(1)
		return logrus.WithError(err).WithFields(Fields{
			"_source": fmt.Sprintf("%s:%d", filepath.Base(filename), line),
		})
	}
	return logrus.WithError(err)
}

func ToJson(v interface{}) string {
	bytes, err := json.Marshal(v)
	if err != nil {
		return "<failed to encode to json>"
	}
	return string(bytes)
}
