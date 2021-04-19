package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RPCTransport struct {
	id          uuid.UUID
	timeout     time.Duration
	cancelfuncs []context.CancelFunc
	queues      map[string]chan requestMsg
	mutex       sync.Mutex
}

func (r *RPCTransport) GetRPCAddress() string {
	return fmt.Sprintf("memory://%s", r.id.String())
}

type responseMsg struct {
	Response []byte
	Err      error
}

type requestMsg struct {
	FunctionName string
	Context      []byte
	Arguments    []byte
	ResponseTo   chan responseMsg
}

func New(timeout time.Duration) (inst *RPCTransport) {
	return &RPCTransport{
		id:      uuid.New(),
		timeout: timeout,
		queues:  map[string]chan requestMsg{},
	}
}

func (r *RPCTransport) Close() {
	r.mutex.Lock()
	for i, cancelfunc := range r.cancelfuncs {
		cancelfunc()
		r.cancelfuncs = r.cancelfuncs[i:]
	}
	r.mutex.Unlock()
}

func (r *RPCTransport) Call(namespace string, functionName string, context []byte, arguments []byte) (response []byte, err error) {
	r.mutex.Lock()
	ch, ok := r.queues[namespace]
	if !ok {
		ch = make(chan requestMsg, 10000)
		r.queues[namespace] = ch
	}
	r.mutex.Unlock()

	respCh := make(chan responseMsg, 1)
	req := requestMsg{
		FunctionName: functionName,
		Context:      context,
		Arguments:    arguments,
		ResponseTo:   respCh,
	}

	// if queue is full - drop request and return ResourceExhausted status.
	select {
	case ch <- req:
	default:
		return nil,
			status.New(codes.ResourceExhausted, "queue is full").Err()
	}

	select {
	case <-time.After(r.timeout):
		return nil, status.New(codes.DeadlineExceeded, "").Err()
	case resp, ok := <-respCh:
		if !ok {
			return nil, status.New(codes.Canceled, "").Err()
		}
		if resp.Err != nil {
			return nil, resp.Err
		}
		return resp.Response, nil
	}
}

type Handle func(functionName string, context []byte, arguments []byte) (response []byte, err error)

func (r *RPCTransport) Listen(namespace string, onListen func(functionName string, context []byte, arguments []byte) (response []byte, err error)) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())

	r.mutex.Lock()
	ch, ok := r.queues[namespace]
	if !ok {
		ch = make(chan requestMsg, 10000)
		r.queues[namespace] = ch
	}
	r.cancelfuncs = append(r.cancelfuncs, cancel)
	r.mutex.Unlock()

	go func() {
		for {
			select {
			case req, ok := <-ch:
				if !ok {
					return
				}
				go func() {
					// todo: exit goroutine on timeout?

					// todo: how to pass context (for canceling)?
					resp, err := onListen(req.FunctionName, req.Context, req.Arguments)
					respMsg := responseMsg{
						Response: resp,
						Err:      err,
					}

					req.ResponseTo <- respMsg
					close(req.ResponseTo)
				}()

			case <-ctx.Done():
				return
			}
		}
	}()

	return cancel
}
