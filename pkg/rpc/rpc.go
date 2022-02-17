package rpc

import (
	"context"
)

// Do not call any of this methods directly or from client.
// Methods must be wrapped inside service server/client

// RPC defines the common interface for the Remote Procedure Call technology
type Caller interface {
	Call(namespace string, functionName string, arguments []byte) (response []byte, err error)
}

type Listener interface {
	//TODO: listen may halt in progress. think about it
	Listen(namespace string, onListen func(functionName string, arguments []byte) (response []byte, err error)) context.CancelFunc
}

type RPC interface {
	Caller
	Listener
}
