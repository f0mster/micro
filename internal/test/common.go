package tests

import (
	"testing"
)

type Logger struct {
	T *testing.T
}

func (d Logger) Error(err error, text, serviceName, rpcName, value string) {
	d.T.Logf("error! %s in service %s in RPC %s: '%s' error %s\r\n", text, serviceName, rpcName, value, err)
}

func (d Logger) Info(text string, serviceName string, rpcName string) {
	d.T.Logf("info! service %s; RPC %s; text '%s'\r\n", text, serviceName, rpcName)
}

func (d Logger) Debug(text string, serviceName string, rpcName string) {
	d.T.Logf("debug! service %s; RPC %s; text '%s'\r\n", text, serviceName, rpcName)
}
