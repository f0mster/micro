package logger

type Logger interface {
	Error(err error, text, serviceName string, rpcName, value string)
	Info(text string, serviceName string, rpcName string)
	Debug(text string, serviceName string, rpcName string)
}
