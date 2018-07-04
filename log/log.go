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
	"runtime"
	"fmt"
	"path/filepath"
	"golang.org/x/crypto/ssh/terminal"
	"os"
)

type Fields = logrus.Fields
type Entry = logrus.Entry
type LogLevel = logrus.Level

const (
	LogLevelDebug LogLevel = logrus.DebugLevel
	LogLevelInfo  LogLevel = logrus.InfoLevel
)

var logLevel = LogLevelInfo

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

func Printf(format string, args ...interface{}) {
	withSource().Printf(format, args...)
}

func Println(args ...interface{}) {
	withSource().Println(args...)
}

func Fatal(v ...interface{}) {
	withSource().Fatal(v...)
}

func withSource() *Entry {
	_, filename, line, _ := runtime.Caller(2)
	return logrus.WithField("_source", formatSource(filename, line))
}

func formatSource(filename string, line int) string {
	return fmt.Sprintf("%s:%d", filepath.Base(filename), line)
}
