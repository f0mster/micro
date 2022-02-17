// Code generated by api_wrapper_code_generator. DO NOT EDIT.
// NOT THREAD SAFE
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/f0mster/micro/pkg/client"
	"github.com/f0mster/micro/pkg/interfaces/contextmarshaller"
	"github.com/f0mster/micro/pkg/interfaces/logger"
	"github.com/f0mster/micro/pkg/pubsub"
	"github.com/f0mster/micro/pkg/registry"
	"github.com/f0mster/micro/pkg/rnd"
	"github.com/f0mster/micro/pkg/server"
)

type DataWithContextApiProtoRpcGo struct {
	Data []byte `json:"data"`
	Ctx  []byte `json:"ctx"`
}

type SessionInternalAPI interface {
	// Connect.
	//
	// Do something.
	Connect(ctx context.Context, req *ConnectReq) (resp *ConnectResp, err error)
	SnakeFuncName(ctx context.Context, req *SnakeMessage) (resp *SnakeMessage, err error)
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

func (h *SessionInternalAPIService) listenRPC(funcName string, arguments []byte) (r []byte, err error) {
	d := DataWithContextApiProtoRpcGo{}
	err = json.Unmarshal(arguments, &d)
	if err != nil {
		return nil, err
	}
	pCtx, _, err := h.config.ContextMarshaller.Unmarshal(d.Ctx)
	if err != nil {
		return nil, err
	}

	callRpc := func(ctx context.Context) error {
		switch funcName {
		case "Connect":
			arg := ConnectReq{}
			err = arg.UnmarshalVT(d.Data)
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

			r, err = resp.MarshalVT()
			if err != nil {
				return err
			}
			return nil
		case "snake_func_name":
			arg := SnakeMessage{}
			err = arg.UnmarshalVT(d.Data)
			if err != nil {
				return err
			}

			var resp *SnakeMessage
			resp, err = h.handler.SnakeFuncName(ctx, &arg)
			if err != nil {
				return err
			}

			// XXX: avoid nil panic if handler returned <nil, nil>.
			if resp == nil {
				resp = new(SnakeMessage)
			}

			r, err = resp.MarshalVT()
			if err != nil {
				return err
			}
			return nil
		default:
			//TODO: not sure about this behavior
			return fmt.Errorf("call of unknown rpc %s %s", "in service SessionInternalAPI ", funcName)
		}
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

//
// Client
//

type SessionInternalAPIClient struct {
	client     *client.Client
	block      bool
	subscriber *SessionInternalAPIEventsSubscriber
}

func NewSessionInternalAPIClient(config client.Config, opt ...ClientOpt) (*SessionInternalAPIClient, error) {
	cClient, err := client.NewClient(config)
	if err != nil {
		return nil, err
	}
	cli := &SessionInternalAPIClient{
		client:     cClient,
		subscriber: NewSessionInternalAPIEventsSubscriber(config.PubSub),
	}
	cli.subscriber.SetLogger(config.Logger)
	cli.subscriber.SetWrapper(config.PubSubWrapper)
	cli.subscriber.SetContextMarshaller(config.ContextMarshaller)

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

// Connect.
//
// Do something.
func (c *SessionInternalAPIClient) Connect(ctx context.Context, request *ConnectReq, callOpts ...client.CallOption) (response *ConnectResp, err error) {
	response = &ConnectResp{}
	err = c.client.CallWithMarshaller(ctx, "SessionInternalAPI", "Connect", request, response, callOpts...)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (c *SessionInternalAPIClient) SnakeFuncName(ctx context.Context, request *SnakeMessage, callOpts ...client.CallOption) (response *SnakeMessage, err error) {
	response = &SnakeMessage{}
	err = c.client.CallWithMarshaller(ctx, "SessionInternalAPI", "snake_func_name", request, response, callOpts...)
	if err != nil {
		return nil, err
	}

	return response, nil
}

//
// Events Subscriber
//

type SessionInternalAPIEventsSubscriber struct {
	ps      pubsub.PubSub
	cm      contextmarshaller.ContextMarshaller
	wrapper PubSubWrap
	logger  logger.Logger
}

func NewSessionInternalAPIEventsSubscriber(ps pubsub.PubSub) *SessionInternalAPIEventsSubscriber {
	return &SessionInternalAPIEventsSubscriber{
		ps:     ps,
		cm:     &contextmarshaller.DefaultCtxMarshaller{},
		logger: &logger.DefaultLogger{},
	}
}

func (s *SessionInternalAPIEventsSubscriber) SetContextMarshaller(cm contextmarshaller.ContextMarshaller) {
	s.cm = cm
}

func (s *SessionInternalAPIEventsSubscriber) SetWrapper(PubSubWrapper func(ctx context.Context, ServiceName, EventName string, PubSubCall func(ctx context.Context) error) error) {
	s.wrapper = PubSubWrapper
}

func (s *SessionInternalAPIEventsSubscriber) SetLogger(log logger.Logger) {
	s.logger = log
}
