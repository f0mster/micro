package testlogger

import (
	"testing"
)

type Logger struct {
	T *testing.T
}

func (s *Logger) Error(err error, text, serviceName, rpcName, value string) {
	s.T.Logf("ERROR %s in service %s in RPC %s: '%s' error %s\r\n", text, serviceName, rpcName, value, err)
}

func (s *Logger) Info(text string, serviceName string, rpcName string) {
	s.T.Logf("INFO service %s; RPC %s; text '%s'\r\n", text, serviceName, rpcName)
}

func (s *Logger) Debug(text string, serviceName string, rpcName string) {
	s.T.Logf("DEBUG service %s; RPC %s; text '%s'\r\n", text, serviceName, rpcName)
}

func New(t *testing.T) *Logger {
	return &Logger{T: t}
}
