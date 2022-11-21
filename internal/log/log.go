// Package log is a preflight-local logrus bridge, removing the
// dependency on logrus' global logger instance.
package log

import (
	"github.com/sirupsen/logrus"
)

var l *logrus.Logger

// Logger returns the configured logger, or a new unconfigured logger
// if one has not already been configured.
func Logger() *logrus.Logger {
	if l == nil {
		l = logrus.New()
	}

	return l
}

// L is a convenience alias to Logger.
var L = Logger
