package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

type Rpc struct {
	timeout    time.Duration
	connection *nats.Conn
}

type ResponseMsg struct {
	Response []byte
	Err      string
}

type RequestMsg struct {
	FunctionName string
	Context      []byte
	Arguments    []byte
	ResponseTo   string
}

func New(addr string, timeout time.Duration) (inst *Rpc, err error) {
	// Connect to a server
	conn, err := nats.Connect(addr)
	if err != nil {
		return
	}
	inst = &Rpc{timeout: timeout, connection: conn}
	return
}

func (r *Rpc) Close() {
	r.connection.Close()
}

func (r *Rpc) Call(namespace string, functionName string, context []byte, arguments []byte) (response []byte, err error) {
	req := RequestMsg{
		FunctionName: functionName,
		Context:      context,
		Arguments:    arguments,
	}
	sreq, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	//TODO: pubsub through pool
	msg, err := r.connection.Request(namespace, sreq, r.timeout)
	if err != nil {
		//t.Log(err)
		return
	}
	resp := ResponseMsg{}
	err = json.Unmarshal(msg.Data, &resp)
	if err == nil {
		if resp.Err != "" {
			err = fmt.Errorf(resp.Err)
		} else {
			response = resp.Response
		}
	}
	return
}

func (r *Rpc) Listen(namespace string, onListen func(functionName string, context []byte, arguments []byte) (response []byte, err error)) (context.CancelFunc, error) {
	s, err := r.connection.QueueSubscribe(
		namespace,
		"queuename...",
		func(msg *nats.Msg) {
			req := RequestMsg{}
			err := json.Unmarshal(msg.Data, &req)
			resp, err := onListen(req.FunctionName, req.Context, req.Arguments)
			errStr := ""
			if err != nil {
				errStr = err.Error()
			}
			respMsg := ResponseMsg{
				Response: resp,
				Err:      errStr,
			}
			res, err := json.Marshal(respMsg)
			err = msg.Respond(res)
		},
	)
	if err != nil {
		return nil, err
	}
	r.connection.Flush()
	return func() {
		_ = s.Unsubscribe()
	}, nil
}
