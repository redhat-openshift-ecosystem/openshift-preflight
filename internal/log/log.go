// Package log is a preflight-local logrus bridge, removing the
// dependency on logrus' global logger instance.
package log

import (
	"bytes"
	"fmt"

	"github.com/go-logr/logr"
)

const (
	DBG int = 1
	TRC int = 2
)

func NewBufferSink(buffer *bytes.Buffer) logr.LogSink {
	return bufferSink{
		buffer: buffer,
	}
}

type bufferSink struct {
	name   string
	buffer *bytes.Buffer
}

func (s bufferSink) Bytes() []byte {
	return s.buffer.Bytes()
}

var _ logr.LogSink = bufferSink{}

func (s bufferSink) Enabled(level int) bool {
	return true
}

func (s bufferSink) Error(err error, msg string, keysAndValues ...interface{}) {
	s.buffer.WriteString(fmt.Sprintf("%s %v %s %v\n", s.name, err.Error(), msg, keysAndValues))
}

func (s bufferSink) Info(level int, msg string, keysAndValues ...interface{}) {
	s.buffer.WriteString(fmt.Sprintf("%s %s %v\n", s.name, msg, keysAndValues))
}

func (s bufferSink) Init(info logr.RuntimeInfo) {}

func (s bufferSink) WithName(name string) logr.LogSink {
	newSink := bufferSink{
		name:   name,
		buffer: s.buffer,
	}
	return newSink
}

func (s bufferSink) WithValues(keysAndValues ...interface{}) logr.LogSink {
	return nil
}
