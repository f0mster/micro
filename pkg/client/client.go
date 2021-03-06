package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	contextmarshaller2 "github.com/f0mster/micro/pkg/interfaces/contextmarshaller"
	"github.com/f0mster/micro/pkg/interfaces/logger"
	"github.com/f0mster/micro/pkg/rpc"

	"github.com/f0mster/micro/pkg/pubsub"
	"github.com/f0mster/micro/pkg/registry"
)

type TransportWrap func(ctx context.Context, serviceName, rpcName string, rpc func(ctx context.Context) error) error
type PubSubWrap func(ctx context.Context, serviceName, eventName string, PubSubCallback func(ctx context.Context) error) error
type DataWithContext struct {
	Data []byte `json:"data"`
	Ctx  []byte `json:"ctx"`
}

type Config struct {
	RPCWrapper        TransportWrap
	PubSubWrapper     PubSubWrap
	RPC               rpc.RPC
	PubSub            pubsub.PubSub
	Registry          registry.Registry
	ContextMarshaller contextmarshaller2.ContextMarshaller
	Logger            logger.Logger
}

type CallOption struct {
}

type Client struct {
	config Config
	mu     sync.Mutex
	ready  map[string]bool
}

func NewClient(config Config) (*Client, error) {
	if config.ContextMarshaller == nil {
		config.ContextMarshaller = &contextmarshaller2.DefaultCtxMarshaller{}
	}
	if config.Logger == nil {
		config.Logger = &logger.DefaultLogger{}
	}
	if config.RPC == nil {
		return nil, fmt.Errorf("rpc must be set")
	}
	if config.Registry == nil {
		return nil, fmt.Errorf("registry must be set")
	}
	if config.PubSub == nil {
		return nil, fmt.Errorf("pubsub must be set")
	}

	return &Client{config: config, ready: map[string]bool{}}, nil
}

type marshaller interface {
	MarshalVT() ([]byte, error)
}

type unmarshaler interface {
	UnmarshalVT([]byte) error
}

func (c *Client) CallWithMarshaller(ctx context.Context, serviceName string, funcName string, protoData marshaller, result unmarshaler, opt ...CallOption) (err error) {
	req, err := protoData.MarshalVT()
	if err != nil {
		return
	}
	data, err := c.Call(ctx, serviceName, funcName, req, opt...)
	if err != nil {
		return
	}
	return result.UnmarshalVT(data)
}

func (c *Client) Call(ctx context.Context, serviceName string, funcName string, protoData []byte, opt ...CallOption) (resp []byte, err error) {
	ctxData, err := c.config.ContextMarshaller.Marshal(ctx)
	if err != nil {
		return
	}
	data, err := json.Marshal(DataWithContext{
		Data: protoData,
		Ctx:  ctxData,
	})
	if err != nil {
		return
	}
	c.mu.Lock()
	serviceReady, ok := c.ready[serviceName]
	if !ok {
		c.config.Registry.WatchRegistered(serviceName, func() {
			c.mu.Lock()
			c.ready[serviceName] = true
			c.mu.Unlock()
		})
		c.config.Registry.WatchUnregistered(serviceName, func() {
			c.mu.Lock()
			c.ready[serviceName] = false
			c.mu.Unlock()
		})
		serviceReady = len(c.config.Registry.Instances(serviceName)) > 0
	}
	c.mu.Unlock()
	if !serviceReady {
		return nil, fmt.Errorf("service not started")
	}
	return c.config.RPC.Call(serviceName, funcName, data)
}

func (c *Client) WaitForServiceStarted(serviceName string) {
	c.mu.Lock()
	started, ok := c.ready[serviceName]
	if !ok || !started {
		wg := sync.WaitGroup{}
		wg.Add(1)
		var cancel registry.CancelFunc
		done := int32(0)
		onRegistered := func() {
			old := atomic.SwapInt32(&done, 1)
			if old == 0 {
				cancel()
				c.ready[serviceName] = true
				wg.Done()
			}
		}
		cancel = c.config.Registry.WatchRegistered(serviceName, onRegistered)
		list := c.config.Registry.Instances(serviceName)
		if len(list) > 0 {
			onRegistered()
		}
		wg.Wait()
	}
	c.mu.Unlock()
}

func (c *Client) GetConfig() Config {
	return c.config
}
