package tests

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/f0mster/micro/client"
	"github.com/f0mster/micro/internal/testlogger"
	"github.com/f0mster/micro/pkg/metadata"
	"github.com/f0mster/micro/pubsub"
	pubsub_memory "github.com/f0mster/micro/pubsub/memory"
	"github.com/f0mster/micro/registry"
	rpc_memory "github.com/f0mster/micro/rpc/memory"
	"github.com/f0mster/micro/server"

	api "github.com/f0mster/micro/internal/test/api"
	registry_memory "github.com/f0mster/micro/registry/memory"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"
)

/*
  Внимание!

  Для запуска необходим docker.

  Адрес указывается через переменную окружения DOCKER_HOST.

*/

var appIndex int32

type TestContext struct {
	redisAddr string

	dockerPool *dockertest.Pool
	dbRes      *dockertest.Resource

	ctx       context.Context
	ctxCancel func()
	logWG     sync.WaitGroup
}

func getAddr(dockerEndpoint, port string) string {
	// experimental support of local docker daemon
	dockerEndpoint = strings.Replace(dockerEndpoint, "tcp://", "", 1)

	host := strings.Split(dockerEndpoint, ":")[0]

	if strings.Contains(dockerEndpoint, "unix:") || strings.Contains(dockerEndpoint, "http://localhost:") {
		host = "0.0.0.0"
	}

	return fmt.Sprintf(
		"%s:%s",
		host,
		port)
}

func TestApi_Call(t *testing.T) {
	wg := sync.WaitGroup{}
	gortns := runtime.NumGoroutine()
	rm := rpc_memory.New(60 * time.Second)
	psm := pubsub_memory.New()
	reg := registry_memory.New()

	servi := serv{}
	wrapperCalled := false
	srv, err := server.NewServer(server.Config{
		RPCWrapper: func(ctx context.Context, serviceName, rpcName string, rpc func(ctx context.Context) error) error {
			wrapperCalled = true
			require.Equal(t, false, servi.connectCalled)
			t.Log("rpc", serviceName, rpcName)
			err := rpc(ctx)
			require.Equal(t, true, servi.connectCalled)
			t.Log("rpc done", serviceName, rpcName)
			return err
		},
		RPC:      rm,
		PubSub:   psm,
		Registry: reg,
		Logger:   testlogger.New(t),
	})
	require.NoError(t, err)
	clientCfg := client.Config{
		RPCWrapper: func(ctx context.Context, serviceName, rpcName string, rpc func(ctx context.Context) error) error {
			wrapperCalled = true
			require.Equal(t, false, servi.connectCalled)
			t.Log("rpc", serviceName, rpcName)
			err := rpc(ctx)
			require.Equal(t, true, servi.connectCalled)
			t.Log("rpc done", serviceName, rpcName)
			return err
		},
		RPC:      rm,
		PubSub:   psm,
		Registry: reg,
		Logger:   testlogger.New(t),
	}

	//TODO: get rpc address

	sc, err := api.NewSessionInternalAPIServiceClient(clientCfg)
	require.NoError(t, err)
	sc.WatchRegistered(func() {
		t.Log("service registered!")
	})

	sc.WatchUnregistered(func() {
		t.Log("service unregistered!")
	})

	s1, err := api.RegisterSessionInternalAPIServiceServer(&servi, srv)
	require.NoError(t, err)
	s1.WatchInstanceRegistered(func(insId registry.InstanceId) {
		t.Log("service instance registered!", insId)
	})

	s1.WatchInstanceUnregistered(func(insId registry.InstanceId) {
		t.Log("service instance unregistered!", insId)
	})

	s, err := api.RegisterSessionInternalAPIServiceServer(&servi, srv)
	require.NoError(t, err)
	stoped := false
	go func() {
		time.Sleep(1 * time.Second)
		var stopConnect pubsub.CancelFunc

		wg.Add(1)
		stopConnect, err = sc.SubscribeOnConnectEvent(func(context context.Context, event *api.OnConnectEvent) error {
			t.Log("got event onconnect!", event)
			stopConnect()
			wg.Done()
			return nil
		})
		ctx := context.Background()
		require.NoError(t, err)
		err = s.PublishOnConnectEvent(ctx, &api.OnConnectEvent{
			Id:            "1",
			UserId:        2,
			Ip:            "3",
			Os:            "4",
			Version:       "5",
			Build:         "6",
			LastChangedAt: 7,
			PushTokenText: "8",
		})
		t.Log("fire done")
		wg.Wait()
		req := api.ConnectReq{
			Id:            "9",
			UserId:        8,
			Ip:            "7",
			Os:            "6",
			Version:       "5",
			Build:         "4",
			LastChangedAt: 3,
			PushTokenText: "2",
		}
		resp, err := sc.Connect(context.Background(), &req)
		require.NoError(t, err)
		require.Equal(t, true, wrapperCalled)
		stopConnect()
		require.Equal(t, int64(10), resp.Value)
		require.NoError(t, err)
		_ = srv.Stop()
		stoped = true
	}()
	err = srv.Start()
	require.NoError(t, err)
	require.Equal(t, true, stoped)
	rm.Close()
	psm.Close()
	time.Sleep(1 * time.Second)
	if runtime.NumGoroutine() > gortns {
		t.Fatal("some goroutines are still running...", runtime.NumGoroutine()-gortns)
	}
}

func TestApi_CallOnNotStartedService(t *testing.T) {
	rr := rpc_memory.New(60 * time.Second)
	er := pubsub_memory.New()
	mr := registry_memory.New()
	cfg := client.Config{
		RPC:      rr,
		PubSub:   er,
		Registry: mr,
		Logger:   testlogger.New(t),
	}
	//TODO: get rpc address

	sc, err := api.NewSessionInternalAPIServiceClient(cfg)
	require.NoError(t, err)
	req := api.ConnectReq{}
	resp, err := sc.Connect(context.Background(), &req)
	require.Error(t, err)
	require.Nil(t, resp)
}

type serv struct {
	connectCalled bool
}

func (s *serv) Connect(ctx context.Context, req *api.ConnectReq) (resp *api.ConnectResp, err error) {
	resp = &api.ConnectResp{Value: 10}
	s.connectCalled = true
	return
}

func (s *serv) Disconnect(ctx context.Context, msg *api.StringMsg) (resp *api.BoolMsg, err error) {
	panic("implement me")
}

func TestClientErrors(t *testing.T) {
	rr := rpc_memory.New(60 * time.Second)
	er := pubsub_memory.New()
	mr := registry_memory.New()
	cfg := client.Config{
		RPC:      nil,
		PubSub:   er,
		Registry: mr,
		Logger:   testlogger.New(t),
	}
	//TODO: get rpc address
	sc, err := api.NewSessionInternalAPIServiceClient(cfg)
	require.Error(t, err)
	require.Nil(t, sc)

	cfg = client.Config{
		RPC:      rr,
		PubSub:   nil,
		Registry: mr,
		Logger:   testlogger.New(t),
	}
	//TODO: get rpc address
	sc, err = api.NewSessionInternalAPIServiceClient(cfg)
	require.Error(t, err)
	require.Nil(t, sc)

	cfg = client.Config{
		RPC:      rr,
		PubSub:   er,
		Registry: nil,
		Logger:   testlogger.New(t),
	}
	//TODO: get rpc address
	sc, err = api.NewSessionInternalAPIServiceClient(cfg)
	require.Error(t, err)
	require.Nil(t, sc)

	cfg = client.Config{
		RPC:      rr,
		PubSub:   er,
		Registry: mr,
		Logger:   testlogger.New(t),
	}
	//TODO: get rpc address
	sc, err = api.NewSessionInternalAPIServiceClient(cfg)
	require.NoError(t, err)
	require.NotNil(t, sc)
}

func TestServerErrors(t *testing.T) {
	servi := serv{}
	sc, err := api.RegisterSessionInternalAPIServiceServer(&servi, &server.Server{})
	require.Error(t, err)
	require.Nil(t, sc)

	rr := rpc_memory.New(60 * time.Second)
	er := pubsub_memory.New()
	mr := registry_memory.New()
	srv, err := server.NewServer(server.Config{
		RPC:      nil,
		PubSub:   er,
		Registry: mr,
		Logger:   testlogger.New(t),
	})
	require.Error(t, err)
	require.Nil(t, srv)

	_, err = server.NewServer(server.Config{
		RPC:      rr,
		PubSub:   nil,
		Registry: mr,
		Logger:   testlogger.New(t),
	})
	require.Error(t, err)

	_, err = server.NewServer(server.Config{
		RPC:      rr,
		PubSub:   er,
		Registry: nil,
		Logger:   testlogger.New(t),
	})
	require.Error(t, err)

	_, err = server.NewServer(server.Config{
		RPC:      rr,
		PubSub:   er,
		Registry: mr,
	})
	require.NoError(t, err)

	_, err = server.NewServer(server.Config{
		RPC:      rr,
		PubSub:   er,
		Registry: mr,
		Logger:   testlogger.New(t),
	})
	require.NoError(t, err)
}

func TestServerAfter_BeforeFunctions(t *testing.T) {
	rr := rpc_memory.New(60 * time.Second)
	er := pubsub_memory.New()
	mr := registry_memory.New()
	funcs := map[int]int64{}
	srv, err := server.NewServer(server.Config{
		RPC:      rr,
		PubSub:   er,
		Registry: mr,
		Logger:   testlogger.New(t),
		BeforeStart: func() error {
			funcs[1] = time.Now().UnixNano()
			time.Sleep(1 * time.Millisecond)
			return nil
		},
		AfterStart: func() error {
			funcs[2] = time.Now().UnixNano()
			time.Sleep(1 * time.Millisecond)
			return nil
		},
		BeforeStop: func() error {
			funcs[3] = time.Now().UnixNano()
			time.Sleep(1 * time.Millisecond)
			return nil
		},
		AfterStop: func() error {
			funcs[4] = time.Now().UnixNano()
			time.Sleep(1 * time.Millisecond)
			return nil
		},
	})
	require.NoError(t, err)
	servi := serv{}
	//TODO: get rpc address
	_, err = api.RegisterSessionInternalAPIServiceServer(&servi, srv)
	require.NoError(t, err)
	go func() {
		time.Sleep(100 * time.Millisecond)
		err = srv.Stop()
		require.NoError(t, err)
	}()
	err = srv.Start()
	require.NoError(t, err)
	require.Greater(t, funcs[1], int64(0))
	require.Greater(t, funcs[2], funcs[1]+1000)
	require.Greater(t, funcs[3], funcs[2]+1000)
	require.Greater(t, funcs[4], funcs[3]+1000)
}

func TestServerAfter_BeforeFunctions_Stop_From_AfterStart(t *testing.T) {
	rr := rpc_memory.New(60 * time.Second)
	er := pubsub_memory.New()
	mr := registry_memory.New()
	funcs := map[int]int64{}
	var srv *server.Server
	var err error
	srv, err = server.NewServer(server.Config{
		RPC:      rr,
		PubSub:   er,
		Registry: mr,
		Logger:   testlogger.New(t),
		BeforeStart: func() error {
			funcs[1] = time.Now().UnixNano()
			time.Sleep(1 * time.Millisecond)
			return nil
		},
		AfterStart: func() error {
			funcs[2] = time.Now().UnixNano()
			time.Sleep(1 * time.Millisecond)
			err = srv.Stop()
			require.NoError(t, err)
			return nil
		},
		BeforeStop: func() error {
			funcs[3] = time.Now().UnixNano()
			time.Sleep(1 * time.Millisecond)
			return nil
		},
		AfterStop: func() error {
			funcs[4] = time.Now().UnixNano()
			time.Sleep(1 * time.Millisecond)
			return nil
		},
	})
	servi := serv{}
	//TODO: get rpc address
	sc, err := api.RegisterSessionInternalAPIServiceServer(&servi, srv)
	require.NoError(t, err)
	require.NotNil(t, sc)
	err = srv.Start()
	require.NoError(t, err)
	require.Greater(t, funcs[1], int64(0))
	require.Greater(t, funcs[2], funcs[1]+1000)
	require.Greater(t, funcs[3], funcs[2]+1000)
	require.Greater(t, funcs[4], funcs[3]+1000)
}

func TestServerAfter_BeforeFunctions_Stop_From_BeforeStart(t *testing.T) {
	rr := rpc_memory.New(60 * time.Second)
	er := pubsub_memory.New()
	mr := registry_memory.New()
	funcs := map[int]int64{}
	var srv *server.Server
	var err error
	srv, err = server.NewServer(server.Config{
		RPC:      rr,
		PubSub:   er,
		Registry: mr,
		Logger:   testlogger.New(t),
		BeforeStart: func() error {
			funcs[1] = time.Now().UnixNano()
			time.Sleep(1 * time.Millisecond)
			err = srv.Stop()
			return nil
		},
		AfterStart: func() error {
			funcs[2] = time.Now().UnixNano()
			time.Sleep(1 * time.Millisecond)
			err = srv.Stop()
			require.NoError(t, err)
			return nil
		},
		BeforeStop: func() error {
			funcs[3] = time.Now().UnixNano()
			time.Sleep(1 * time.Millisecond)
			return nil
		},
		AfterStop: func() error {
			funcs[4] = time.Now().UnixNano()
			time.Sleep(1 * time.Millisecond)
			return nil
		},
	})
	servi := serv{}
	//TODO: get rpc address
	sc, err := api.RegisterSessionInternalAPIServiceServer(&servi, srv)
	err = srv.Start()
	require.NoError(t, err)
	require.NotNil(t, sc)
	require.Greater(t, funcs[1], int64(0))
	require.Equal(t, int64(0), funcs[2])
	require.Greater(t, funcs[3], funcs[1]+1000)
	require.Greater(t, funcs[4], funcs[3]+1000)
}

func TestServerPubSubWrapper(t *testing.T) {
	rr := rpc_memory.New(60 * time.Second)
	er := pubsub_memory.New()
	mr := registry_memory.New()
	var srv *server.Server
	var err error
	var pubsubPublisherWrapperCalled int
	var pubsubSubscriberWrapperCalled int
	var sc *api.SessionInternalAPIServiceService
	pubsubServiceName := "SessionInternalAPIService"
	pubsubEventName := "OnConnectEvent"

	sessClient, err := api.NewSessionInternalAPIServiceClient(client.Config{
		RPC:      rr,
		PubSub:   er,
		Registry: mr,
		PubSubWrapper: func(ctx context.Context, serviceName, eventName string, PubSubCallback func(ctx context.Context) error) error {
			pubsubSubscriberWrapperCalled++
			require.Equal(t, pubsubServiceName, serviceName)
			require.Equal(t, pubsubEventName, eventName)
			ctx = metadata.Set(ctx, "sctx", "sctxval")
			PubSubCallback(ctx)
			return nil
		},
	})
	stop, err := sessClient.SubscribeOnConnectEvent(func(ctx context.Context, event *api.OnConnectEvent) error {
		val, _ := metadata.Get(ctx, "pctx")
		require.Equal(t, "pctxval", val)
		val, _ = metadata.Get(ctx, "sctx")
		require.Equal(t, "sctxval", val)
		return nil
	})

	srv, err = server.NewServer(server.Config{
		RPC:      rr,
		PubSub:   er,
		Registry: mr,
		PubSubWrapper: func(ctx context.Context, serviceName, eventName string, PubSubWrap func(ctx context.Context) error) error {
			pubsubPublisherWrapperCalled++
			require.Equal(t, pubsubServiceName, serviceName)
			require.Equal(t, pubsubEventName, eventName)
			ctx = metadata.Set(ctx, "pctx", "pctxval")
			PubSubWrap(ctx)
			return nil
		},
		Logger: testlogger.New(t),
		AfterStart: func() error {
			sc.PublishOnConnectEvent(context.Background(), &api.OnConnectEvent{Id: "ssss"})
			time.Sleep(1 * time.Millisecond)
			err = srv.Stop()
			require.NoError(t, err)
			return nil
		},
	})
	servi := serv{}
	//TODO: get rpc address
	sc, err = api.RegisterSessionInternalAPIServiceServer(&servi, srv)
	err = srv.Start()
	stop()
	require.Equal(t, pubsubPublisherWrapperCalled, 1)
	require.NoError(t, err)
	require.NotNil(t, sc)
}
