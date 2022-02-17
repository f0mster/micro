package logger

import (
	"fmt"
)

type DefaultLogger struct {
}

func (d *DefaultLogger) Error(err error, text, serviceName, rpcName, value string) {
	fmt.Printf("ERROR %s in service %s in RPC %s: '%s' error %s\r\n", text, serviceName, rpcName, value, err)
}

func (d *DefaultLogger) Info(text string, serviceName string, rpcName string) {
	fmt.Printf("INFO service %s; RPC %s; text '%s'\r\n", text, serviceName, rpcName)
}

func (d *DefaultLogger) Debug(text string, serviceName string, rpcName string) {
	fmt.Printf("DEBUG service %s; RPC %s; text '%s'\r\n", text, serviceName, rpcName)
}
