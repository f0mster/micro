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

type SessionInternalAPIService interface {

	// qwqwdqwd some test comment
	Connect(ctx context.Context, req *ConnectReq) (resp *ConnectResp, err error)

	Disconnect(ctx context.Context, req *StringMsg) (resp *BoolMsg, err error)
}

type SessionInternalAPIServiceService struct {
	handler    SessionInternalAPIService
	config     server.Config
	instanceId string
	started    int32
	stopListen context.CancelFunc
	publisher  *SessionInternalAPIServiceEventsPublisher
}

func (h *SessionInternalAPIServiceService) GetInstanceId() string {
	return h.instanceId
}

/*
func RegisterSessionInternalAPIServiceServerWithOptions(service SessionInternalAPIService, opts ... server.Option) *SessionInternalAPIServiceServer {
	//TODO
}
*/

func RegisterSessionInternalAPIServiceServer(handler SessionInternalAPIService, srv *server.Server) (*SessionInternalAPIServiceService, error) {
	insId, err := rnd.GenerateRandomString(20)
	if err != nil {
		return nil, err
	}
	cfg, err := srv.GetConfig()
	if err != nil {
		return nil, err
	}
	ins := &SessionInternalAPIServiceService{
		handler:    handler,
		instanceId: insId,
		config:     cfg,
		publisher:  NewSessionInternalAPIServiceEventsPublisher(cfg.PubSub),
	}
	ins.publisher.SetContextMarshaller(cfg.ContextMarshaller)
	ins.publisher.SetWrapper(cfg.PubSubWrapper)

	str := starterSessionInternalAPIService{ins: ins}
	err = srv.Register(&str)
	if err != nil {
		return nil, err
	}

	return ins, nil
}

type starterSessionInternalAPIService struct {
	ins *SessionInternalAPIServiceService
}

func (s *starterSessionInternalAPIService) Start() {
	s.ins.stopListen = s.ins.config.RPC.Listen("SessionInternalAPIService", s.ins.listenRPC)
	s.ins.config.Registry.Register("SessionInternalAPIService", registry.InstanceId(s.ins.instanceId))
	atomic.StoreInt32(&s.ins.started, 1)
}

func (s *starterSessionInternalAPIService) Stop() {
	if atomic.LoadInt32(&s.ins.started) == 0 {
		return
	}
	s.ins.config.Registry.Unregister("SessionInternalAPIService", registry.InstanceId(s.ins.instanceId))
	s.ins.stopListen()
	atomic.StoreInt32(&s.ins.started, 0)
}

// watch on service instance register
func (c *SessionInternalAPIServiceService) WatchInstanceRegistered(callback func(instanceId registry.InstanceId)) registry.CancelFunc {
	return c.config.Registry.WatchInstanceRegistered("SessionInternalAPIService", callback)
}

// watch on service instance unregister
func (c *SessionInternalAPIServiceService) WatchInstanceUnregistered(callback func(instanceId registry.InstanceId)) registry.CancelFunc {
	return c.config.Registry.WatchInstanceUnregistered("SessionInternalAPIService", callback)
}
func (s *SessionInternalAPIServiceService) PublishConnectEvent(ctx context.Context, event *ConnectEvent) error {
	return s.publisher.PublishConnectEvent(ctx, event)
}

func (h *SessionInternalAPIServiceService) listenRPC(funcName string, ctx []byte, arguments []byte) (r []byte, err error) {
	pCtx, _, err := h.config.ContextMarshaller.Unmarshal(ctx)
	if err != nil {
		return nil, err
	}

	callRpc := func(ctx context.Context) error {

		switch funcName {
		case "Connect":
			arg := ConnectReq{}
			err = arg.Unmarshal(arguments)
			if err != nil {
				return err
			}

			var resp *ConnectResp
			resp, err = h.handler.Connect(ctx, &arg)
			if err != nil {
				return err
			}

			// XXX: avoid nil panic if handler returned <nil, nil>.
			if resp == nil {
				resp = new(ConnectResp)
			}

			r, err = resp.Marshal()
			if err != nil {
				return err
			}
			return nil
		case "Disconnect":
			arg := StringMsg{}
			err = arg.Unmarshal(arguments)
			if err != nil {
				return err
			}

			var resp *BoolMsg
			resp, err = h.handler.Disconnect(ctx, &arg)
			if err != nil {
				return err
			}

			// XXX: avoid nil panic if handler returned <nil, nil>.
			if resp == nil {
				resp = new(BoolMsg)
			}

			r, err = resp.Marshal()
			if err != nil {
				return err
			}
			return nil
		default:
			//TODO: not sure about this behavior
			return fmt.Errorf("call of unknown rpc %s %s", "in service SessionInternalAPIService ", funcName)
		}

	}

	wrap := h.config.RPCWrapper
	if wrap != nil {
		err = wrap(pCtx, "SessionInternalAPIService", funcName, callRpc)
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
type SessionInternalAPIServiceEventsPublisher struct {
	ps      pubsub.PubSub
	cm      contextmarshaller.ContextMarshaller
	wrapper PubSubWrap
}

func NewSessionInternalAPIServiceEventsPublisher(ps pubsub.PubSub) *SessionInternalAPIServiceEventsPublisher {
	return &SessionInternalAPIServiceEventsPublisher{
		ps: ps,
		cm: &contextmarshaller.DefaultCtxMarshaller{},
	}
}

func (s *SessionInternalAPIServiceEventsPublisher) SetContextMarshaller(cm contextmarshaller.ContextMarshaller) {
	s.cm = cm
}

func (s *SessionInternalAPIServiceEventsPublisher) SetWrapper(PubSubWrapper func(ctx context.Context, ServiceName, EventName string, PubSubCall func(ctx context.Context) error) error) {
	s.wrapper = PubSubWrapper
}
func (s *SessionInternalAPIServiceEventsPublisher) PublishConnectEvent(ctx context.Context, event *ConnectEvent) error {
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

		err = s.ps.Publish("SessionInternalAPIService", "ConnectEvent", ctxData, r)
		if err != nil {
			return fmt.Errorf("error while publishing error: %w", err)
		}

		return nil
	}

	if s.wrapper == nil {
		return cb(ctx)
	}
	return s.wrapper(ctx, "SessionInternalAPIService", "ConnectEvent", cb)
}

//
// Client
//

type SessionInternalAPIServiceClient struct {
	client *client.Client
	block  bool
}

func NewSessionInternalAPIServiceClient(config client.Config, opt ...ClientOpt) (*SessionInternalAPIServiceClient, error) {
	cClient, err := client.NewClient(config)
	if err != nil {
		return nil, err
	}
	cli := &SessionInternalAPIServiceClient{
		client: cClient,
	}
	for _, o := range opt {
		o(cli)
	}

	if cli.block {
		cli.client.WaitForServiceStarted("SessionInternalAPIService")
	}

	return cli, nil
}

type ClientOpt func(c *SessionInternalAPIServiceClient)

func WithBlock() ClientOpt {
	return func(c *SessionInternalAPIServiceClient) {
		c.block = true
	}
}

// watch on service register
func (c *SessionInternalAPIServiceClient) WatchRegistered(callback func()) registry.CancelFunc {
	return c.client.GetConfig().Registry.WatchRegistered("SessionInternalAPIService", callback)
}

// watch on service unregister
func (c *SessionInternalAPIServiceClient) WatchUnregistered(callback func()) registry.CancelFunc {
	return c.client.GetConfig().Registry.WatchUnregistered("SessionInternalAPIService", callback)
}

// qwqwdqwd some test comment
func (c *SessionInternalAPIServiceClient) Connect(ctx context.Context, request *ConnectReq, callOpts ...client.CallOption) (response *ConnectResp, err error) {
	response = &ConnectResp{}
	err = c.client.CallWithMarshaller(ctx, "SessionInternalAPIService", "Connect", request, response, callOpts...)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (c *SessionInternalAPIServiceClient) Disconnect(ctx context.Context, request *StringMsg, callOpts ...client.CallOption) (response *BoolMsg, err error) {
	response = &BoolMsg{}
	err = c.client.CallWithMarshaller(ctx, "SessionInternalAPIService", "Disconnect", request, response, callOpts...)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (c *SessionInternalAPIServiceClient) UnsubscribeConnectEvent() {
	c.client.Unsubscribe("SessionInternalAPIService", "ConnectEvent")
}

// Nil error in callback required to stop event retrying, if it supported by pubsub provider
func (c *SessionInternalAPIServiceClient) SubscribeConnectEvent(callback func(ctx context.Context, event *ConnectEvent) error) (stop pubsub.CancelFunc, err error) {
	stop, err = c.client.GetConfig().PubSub.Subscribe("SessionInternalAPIService", "ConnectEvent", func(ctxData, event []byte) error {
		log := c.client.GetConfig().Logger
		ev := ConnectEvent{}
		err := ev.Unmarshal(event)
		log.Debug("got event", "SessionInternalAPIService", "ConnectEvent")
		if err != nil {
			log.Error(err, "error while unmarshalling event", "SessionInternalAPIService", "ConnectEvent", string(event))
			return fmt.Errorf("error while unmarshalling event: %w", err)
		}
		ctx, _, err := c.client.GetConfig().ContextMarshaller.Unmarshal(ctxData)
		if err != nil {
			log.Error(err, "error while unmarshalling context data", "SessionInternalAPIService", "ConnectEvent", string(ctxData))
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
			err = wrap(ctx, "SessionInternalAPIService", "ConnectEvent", wrapCB)
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
