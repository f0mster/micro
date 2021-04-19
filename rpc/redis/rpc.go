package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/f0mster/micro/pkg/rnd"
)
import "github.com/mediocregopher/radix/v3"

type Rpc struct {
	pool       *radix.Pool
	pubsub     radix.PubSubConn
	timeout    time.Duration
	rpcAddress string
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

func New(network, addr string, poolsize int, timeout time.Duration) (inst *Rpc, err error) {
	inst = &Rpc{timeout: timeout}
	inst.pool, err = radix.NewPool(network, addr, poolsize)
	if err != nil {
		// handle error
		return nil, err
	}
	inst.pubsub, err = radix.PersistentPubSubWithOpts(network, addr)
	if err != nil {
		// handle error
		return nil, err
	}
	inst.rpcAddress, err = rnd.GenerateRandomString(20)
	if err != nil {
		// handle error
		return nil, err
	}
	return
}

func (r *Rpc) Close() {
	r.pool.Close()
	r.pubsub.Close()
}

func (r *Rpc) Call(namespace string, functionName string, context []byte, arguments []byte) (response []byte, err error) {
	//TODO: pubsub through pool
	msgCh := make(chan radix.PubSubMessage, 10000)
	// todo: to @Pasha - HANDLE ERROR!
	myTopic, err := rnd.GenerateRandomString(20)
	err = r.pubsub.Subscribe(msgCh, myTopic)
	if err != nil {
		return
	}
	req := RequestMsg{
		FunctionName: functionName,
		Context:      context,
		Arguments:    arguments,
		ResponseTo:   myTopic,
	}
	sreq, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	err = r.pool.Do(radix.Cmd(nil, "RPUSH", namespace+"queue", string(sreq)))
	if err != nil {
		fmt.Println(err.Error())
	}
	err = r.pool.Do(radix.Cmd(nil, "PUBLISH", namespace+"queuePS", "1"))
	if err != nil {
		fmt.Println(err.Error())
	}
	errCh := make(chan error, 1)
	ticker := time.NewTimer(r.timeout)

	select {
	case <-ticker.C:
		err = fmt.Errorf("rpc request timeout")
	case msg := <-msgCh:
		resp := ResponseMsg{}
		err = json.Unmarshal(msg.Message, &resp)
		if err == nil {
			if resp.Err != "" {
				err = fmt.Errorf(resp.Err)
			} else {
				response = resp.Response
			}
		}
	case err = <-errCh:
	}
	_ = r.pubsub.Unsubscribe(msgCh, myTopic)
	return
}

func (r *Rpc) Listen(namespace string, onListen func(functionName string, context []byte, arguments []byte) (response []byte, err error)) context.CancelFunc {
	do := true
	msgCh := make(chan radix.PubSubMessage, 10000)
	err := r.pubsub.Subscribe(msgCh, namespace+"queuePS")
	if err != nil {
		fmt.Println(err.Error())
	}

	go func() {
		for do {
			var fooVal string
			err := r.pool.Do(radix.Cmd(&fooVal, "LPOP", namespace+"queue"))
			if err != nil && err != io.EOF {
				fmt.Println("radix error", err.Error())
				panic(0)
			}
			if len(fooVal) > 0 {
				req := RequestMsg{}
				// todo: to @Pasha - HANDLE ERROR!
				err = json.Unmarshal([]byte(fooVal), &req)
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
				// todo: to @Pasha - HANDLE ERROR!
				r.pool.Do(radix.Cmd(nil, "PUBLISH", req.ResponseTo, string(res)))
			} else {
				for len(msgCh) > 0 {
					<-msgCh
				}
				<-msgCh
			}
		}
	}()
	return func() {
		do = false
		// todo: to @Pasha - HANDLE ERROR!
		r.pubsub.Unsubscribe(msgCh, namespace+"queuePS")
		msgCh <- radix.PubSubMessage{}
		close(msgCh)
	}
}

func (r *Rpc) GetRPCAddress() string {
	return r.rpcAddress
}
