// Code generated by api_wrapper_code_generator. DO NOT EDIT.
// NOT THREAD SAFE
package sessionInternalAPI

import (
	"context"
	"fmt"
	"github.com/f0mster/micro/client"
	"github.com/f0mster/micro/interfaces/contextmarshaller"
	"github.com/f0mster/micro/pkg/rnd"
	"github.com/f0mster/micro/pubsub"
	"github.com/f0mster/micro/registry"
	"github.com/f0mster/micro/server"
	"sync/atomic"
)

type SessionInternalAPI interface {
}

type SessionInternalAPIService struct {
	handler    SessionInternalAPI
	config     server.Config
	instanceId string
	started    int32
	stopListen context.CancelFunc
	publisher  *SessionInternalAPIEventsPublisher
}

func (h *SessionInternalAPIService) GetInstanceId() string {
	return h.instanceId
}

/*
func RegisterSessionInternalAPIServerWithOptions(service SessionInternalAPI, opts ... server.Option) *SessionInternalAPIServer {
	//TODO
}
*/

func RegisterSessionInternalAPIServer(handler SessionInternalAPI, srv *server.Server) (*SessionInternalAPIService, error) {
	insId, err := rnd.GenerateRandomString(20)
	if err != nil {
		return nil, err
	}
	cfg, err := srv.GetConfig()
	if err != nil {
		return nil, err
	}
	ins := &SessionInternalAPIService{
		handler:    handler,
		instanceId: insId,
		config:     cfg,
		publisher:  NewSessionInternalAPIEventsPublisher(cfg.PubSub),
	}
	ins.publisher.SetContextMarshaller(cfg.ContextMarshaller)
	ins.publisher.SetWrapper(cfg.PubSubWrapper)

	str := starterSessionInternalAPI{ins: ins}
	err = srv.Register(&str)
	if err != nil {
		return nil, err
	}

	return ins, nil
}

type starterSessionInternalAPI struct {
	ins *SessionInternalAPIService
}

func (s *starterSessionInternalAPI) Start() {
	s.ins.stopListen = s.ins.config.RPC.Listen("SessionInternalAPI", s.ins.listenRPC)
	s.ins.config.Registry.Register("SessionInternalAPI", registry.InstanceId(s.ins.instanceId))
	atomic.StoreInt32(&s.ins.started, 1)
}

func (s *starterSessionInternalAPI) Stop() {
	if atomic.LoadInt32(&s.ins.started) == 0 {
		return
	}
	s.ins.config.Registry.Unregister("SessionInternalAPI", registry.InstanceId(s.ins.instanceId))
	s.ins.stopListen()
	atomic.StoreInt32(&s.ins.started, 0)
}

// watch on service instance register
func (c *SessionInternalAPIService) WatchInstanceRegistered(callback func(instanceId registry.InstanceId)) registry.CancelFunc {
	return c.config.Registry.WatchInstanceRegistered("SessionInternalAPI", callback)
}

// watch on service instance unregister
func (c *SessionInternalAPIService) WatchInstanceUnregistered(callback func(instanceId registry.InstanceId)) registry.CancelFunc {
	return c.config.Registry.WatchInstanceUnregistered("SessionInternalAPI", callback)
}
func (s *SessionInternalAPIService) PublishConnectEvent(ctx context.Context, event *ConnectEvent) error {
	return s.publisher.PublishConnectEvent(ctx, event)
}

func (h *SessionInternalAPIService) listenRPC(funcName string, ctx []byte, arguments []byte) (r []byte, err error) {
	pCtx, _, err := h.config.ContextMarshaller.Unmarshal(ctx)
	if err != nil {
		return nil, err
	}

	callRpc := func(ctx context.Context) error {

		return fmt.Errorf("service have no rpcs")

	}

	wrap := h.config.RPCWrapper
	if wrap != nil {
		err = wrap(pCtx, "SessionInternalAPI", funcName, callRpc)
		if err != nil {
			return nil, err
		}
		return r, nil
	}

	err = callRpc(pCtx)
	if err != nil {
		return nil, err
	}

	return r, nil
}

//
// Event publisher
//

type PubSubWrap func(ctx context.Context, serviceName, eventName string, PubSubWrapper func(ctx context.Context) error) error
type SessionInternalAPIEventsPublisher struct {
	ps      pubsub.PubSub
	cm      contextmarshaller.ContextMarshaller
	wrapper PubSubWrap
}

func NewSessionInternalAPIEventsPublisher(ps pubsub.PubSub) *SessionInternalAPIEventsPublisher {
	return &SessionInternalAPIEventsPublisher{
		ps: ps,
		cm: &contextmarshaller.DefaultCtxMarshaller{},
	}
}

func (s *SessionInternalAPIEventsPublisher) SetContextMarshaller(cm contextmarshaller.ContextMarshaller) {
	s.cm = cm
}

func (s *SessionInternalAPIEventsPublisher) SetWrapper(PubSubWrapper func(ctx context.Context, ServiceName, EventName string, PubSubCall func(ctx context.Context) error) error) {
	s.wrapper = PubSubWrapper
}
func (s *SessionInternalAPIEventsPublisher) PublishConnectEvent(ctx context.Context, event *ConnectEvent) error {
	var r []byte
	var err error
	if event != nil {
		r, err = event.Marshal()
		if err != nil {
			return fmt.Errorf("can't marshal event to proto: %w", err)
		}
	}
	cb := func(ctx context.Context) error {
		ctxData, err := s.cm.Marshal(ctx)
		if err != nil {
			return fmt.Errorf("can't marshal context data: %w", err)
		}

		err = s.ps.Publish("SessionInternalAPI", "ConnectEvent", ctxData, r)
		if err != nil {
			return fmt.Errorf("error while publishing error: %w", err)
		}

		return nil
	}

	if s.wrapper == nil {
		return cb(ctx)
	}
	return s.wrapper(ctx, "SessionInternalAPI", "ConnectEvent", cb)
}

//
// Client
//

type SessionInternalAPIClient struct {
	client *client.Client
	block  bool
}

func NewSessionInternalAPIClient(config client.Config, opt ...ClientOpt) (*SessionInternalAPIClient, error) {
	cClient, err := client.NewClient(config)
	if err != nil {
		return nil, err
	}
	cli := &SessionInternalAPIClient{
		client: cClient,
	}
	for _, o := range opt {
		o(cli)
	}

	if cli.block {
		cli.client.WaitForServiceStarted("SessionInternalAPI")
	}

	return cli, nil
}

type ClientOpt func(c *SessionInternalAPIClient)

func WithBlock() ClientOpt {
	return func(c *SessionInternalAPIClient) {
		c.block = true
	}
}

// watch on service register
func (c *SessionInternalAPIClient) WatchRegistered(callback func()) registry.CancelFunc {
	return c.client.GetConfig().Registry.WatchRegistered("SessionInternalAPI", callback)
}

// watch on service unregister
func (c *SessionInternalAPIClient) WatchUnregistered(callback func()) registry.CancelFunc {
	return c.client.GetConfig().Registry.WatchUnregistered("SessionInternalAPI", callback)
}

func (c *SessionInternalAPIClient) UnsubscribeConnectEvent() {
	c.client.Unsubscribe("SessionInternalAPI", "ConnectEvent")
}

// Nil error in callback required to stop event retrying, if it supported by pubsub provider
func (c *SessionInternalAPIClient) SubscribeConnectEvent(callback func(ctx context.Context, event *ConnectEvent) error) (stop pubsub.CancelFunc, err error) {
	stop, err = c.client.GetConfig().PubSub.Subscribe("SessionInternalAPI", "ConnectEvent", func(ctxData, event []byte) error {
		log := c.client.GetConfig().Logger
		ev := ConnectEvent{}
		err := ev.Unmarshal(event)
		log.Debug("got event", "SessionInternalAPI", "ConnectEvent")
		if err != nil {
			log.Error(err, "error while unmarshalling event", "SessionInternalAPI", "ConnectEvent", string(event))
			return fmt.Errorf("error while unmarshalling event: %w", err)
		}
		ctx, _, err := c.client.GetConfig().ContextMarshaller.Unmarshal(ctxData)
		if err != nil {
			log.Error(err, "error while unmarshalling context data", "SessionInternalAPI", "ConnectEvent", string(ctxData))
			return fmt.Errorf("error while unmarshalling context data: %w", err)
		}
		wrap := c.client.GetConfig().PubSubWrapper
		if wrap != nil {
			wrapCB := func(ctx context.Context) error {
				err = callback(ctx, &ev)
				if err != nil {
					return fmt.Errorf("error while calling subscribe callback: %w", err)
				}
				return nil
			}
			err = wrap(ctx, "SessionInternalAPI", "ConnectEvent", wrapCB)
			return err
		}
		err = callback(ctx, &ev)
		if err != nil {
			return fmt.Errorf("error while calling subscribe callback: %w", err)
		}
		return nil
	})

	return stop, err
}
