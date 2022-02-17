package server

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	contextmarshaller2 "github.com/f0mster/micro/pkg/interfaces/contextmarshaller"
	logger2 "github.com/f0mster/micro/pkg/interfaces/logger"
	"github.com/f0mster/micro/pkg/pubsub"
	"github.com/f0mster/micro/pkg/registry"
	"github.com/f0mster/micro/pkg/rpc"
)

type RpcWrap func(ctx context.Context, serviceName, rpcName string, rpc func(ctx context.Context) error) error

type Server struct {
	services []runner
	config   *Config
	started  int32
	mu       sync.Mutex
	runLock  chan bool
}

func NewServer(config Config) (*Server, error) {
	s := &Server{
		config:   &config,
		services: []runner{},
		runLock:  make(chan bool),
	}
	if err := s.checkConfig(); err != nil {
		return nil, err
	}
	return s, nil
}

type runner interface {
	Start()
	Stop()
}

func (s *Server) checkConfig() error {
	if s.config == nil {
		return fmt.Errorf("you must use NewServer constructor")
	}
	if s.config.RPC == nil {
		return fmt.Errorf("rpc must be set")
	}
	if s.config.Registry == nil {
		return fmt.Errorf("registry must be set")
	}
	if s.config.PubSub == nil {
		return fmt.Errorf("pubsub must be set")
	}

	if s.config.ContextMarshaller == nil {
		ctx := contextmarshaller2.DefaultCtxMarshaller{}
		s.config.ContextMarshaller = &ctx
	}
	if s.config.Logger == nil {
		ctx := logger2.DefaultLogger{}
		s.config.Logger = &ctx
	}
	return nil
}

func (s *Server) GetConfig() (Config, error) {
	cfg := Config{}
	if s.config != nil {
		cfg = *s.config
	}
	return cfg, s.checkConfig()
}

func (s *Server) Start() error {
	if err := s.checkConfig(); err != nil {
		return err
	}

	if len(s.services) == 0 {
		return fmt.Errorf("no service registered")
	}
	if !atomic.CompareAndSwapInt32(&s.started, 0, 1) {
		return fmt.Errorf("server already started")
	}
	defer atomic.StoreInt32(&s.started, 0)
	if s.config.BeforeStart != nil {
		err := s.config.BeforeStart()
		if err != nil {
			return err
		}
		if atomic.LoadInt32(&s.started) == -1 {
			return nil
		}
	}
	s.mu.Lock()
	for _, si := range s.services {
		si.Start()
	}
	s.mu.Unlock()

	if !atomic.CompareAndSwapInt32(&s.started, 1, 2) {
		return nil
	}
	if s.config.AfterStart != nil {
		err := s.config.AfterStart()
		if err != nil {
			_ = s.Stop()
			return err
		}
	}
	if !atomic.CompareAndSwapInt32(&s.started, 2, 3) {
		_ = s.Stop()
		return nil
	}
	<-s.runLock
	return nil
}

func (s *Server) Stop() error {
	var err error
	old := atomic.SwapInt32(&s.started, -1)
	if old == 0 {
		return fmt.Errorf("service wasn't started")
	}
	if old == -1 {
		return nil
	}
	if s.config.BeforeStop != nil {
		err = s.config.BeforeStop()
	}
	s.mu.Lock()
	for _, si := range s.services {
		si.Stop()
	}
	s.mu.Unlock()

	if old == 3 {
		s.runLock <- true
	}
	if err == nil && s.config.AfterStop != nil {
		err = s.config.AfterStop()
	}
	return err
}

func (s *Server) Register(sr runner) error {
	if s.services == nil {
		return fmt.Errorf("server must be created using NewServer")
	}
	s.services = append(s.services, sr)
	return nil
}

type PubSubWrap func(ctx context.Context, serviceName, eventName string, PubSubCallback func(ctx context.Context) error) error

type Config struct {
	RPCWrapper        RpcWrap
	PubSubWrapper     PubSubWrap
	RPC               rpc.RPC
	PubSub            pubsub.PubSub
	Registry          registry.Registry
	ContextMarshaller contextmarshaller2.ContextMarshaller
	Logger            logger2.Logger

	// If any of these callback returns error - server will be stopped and no other callback will be called
	BeforeStart func() error
	BeforeStop  func() error
	AfterStart  func() error
	AfterStop   func() error
}

type options struct {
	beforeStart []func() error
	beforeStop  []func() error
	afterStart  []func() error
	afterStop   []func() error
	Config
}

type Option func(o *options)

func RpcWrapper(fn RpcWrap) Option {
	return func(o *options) {
		o.RPCWrapper = fn
	}
}

func BeforeStart(fn func() error) Option {
	return func(o *options) {
		o.beforeStart = append(o.beforeStart, fn)
	}
}

func BeforeStop(fn func() error) Option {
	return func(o *options) {
		o.beforeStop = append(o.beforeStop, fn)
	}
}

func AfterStart(fn func() error) Option {
	return func(o *options) {
		o.afterStart = append(o.afterStart, fn)
	}
}

func AfterStop(fn func() error) Option {
	return func(o *options) {
		o.afterStop = append(o.afterStop, fn)
	}
}

func (o *options) GetConfig() Config {
	return o.Config
}
