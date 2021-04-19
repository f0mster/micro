package counsul

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/f0mster/micro/pkg/rnd"
)

import "github.com/mediocregopher/radix/v3"

type RedisRpc struct {
	pool   *radix.Pool
	pubsub radix.PubSubConn
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

func New(network, addr string, poolsize int) (inst *RedisRpc, err error) {
	inst = &RedisRpc{}
	inst.pool, err = radix.NewPool(network, addr, poolsize)
	if err != nil {
		return nil, err
	}
	inst.pubsub, err = radix.PersistentPubSubWithOpts(network, addr)
	if err != nil {
		// handle error
		return nil, err
	}
	return
}

func (r *RedisRpc) Call(namespace string, functionName string, context []byte, arguments []byte) (response []byte, err error) {
	//TODO: pubsub through pool
	msgCh := make(chan radix.PubSubMessage)
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
	// todo: to @Pasha - HANDLE ERROR!
	r.pool.Do(radix.FlatCmd(nil, "RPUSH", namespace+"queue", string(sreq)))
	errCh := make(chan error, 1)
	ticker := time.NewTimer(5 * time.Second)

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

func (r *RedisRpc) Listen(namespace string, onListen func(functionName string, context []byte, arguments []byte) (response []byte, err error)) context.CancelFunc {
	do := true
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
				if err != nil {
					continue
				}
				_ = r.pool.Do(radix.Cmd(nil, "PUBLISH", req.ResponseTo, string(res)))
			} else {
				time.Sleep(1 * time.Millisecond)
			}
		}
	}()
	return func() {
		do = false
	}
}
