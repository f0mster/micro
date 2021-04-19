package sessionInternalAPI

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mediocregopher/radix/v3"
	"github.com/ory/dockertest/v3"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/f0mster/micro/client"
	"github.com/f0mster/micro/internal/testlogger"
	events_redis "github.com/f0mster/micro/pubsub/redis"
	"github.com/f0mster/micro/registry"
	memRepo "github.com/f0mster/micro/registry/memory"
	rpc_redis "github.com/f0mster/micro/rpc/redis"
	"github.com/f0mster/micro/server"
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

func (tc *TestContext) SetUp(t testing.TB) {
	tc.ctx, tc.ctxCancel = context.WithCancel(context.Background())

	t.Log("SetUp")
	//if os.Getenv("DOCKER_HOST") == "" {
	//	fmt.Println("Specify env DOCKER_HOST")
	//	os.Exit(1)
	// }

	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	if p, e := dockertest.NewPool(""); e != nil {
		t.Fatalf("Could not connect to docker: %s", e)
	} else {
		tc.dockerPool = p
	}

	// pulls an image, creates a container based on it and runs it
	if r, e := tc.dockerPool.Run(
		"redis",
		"6.0.8-alpine3.12",
		nil,
	); e != nil {
		t.Fatalf("Could not start resource: %s", e)
	} else {
		tc.dbRes = r
	}

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	tc.redisAddr = getAddr(tc.dockerPool.Client.Endpoint(), tc.dbRes.GetPort("6379/tcp"))
	if err := tc.dockerPool.Retry(func() error {
		conn, err := radix.Dial("tcp", tc.redisAddr)
		if err != nil {
			return err
		}
		data := ""
		err = conn.Do(radix.Cmd(nil, "SET", "a", "{\"asadasd\"}"))
		if err != nil {
			return err
		}
		err = conn.Do(radix.Cmd(&data, "GET", "a"))
		require.NoError(t, err)
		require.Equal(t, "{\"asadasd\"}", data)
		return nil
	}); err != nil {
		t.Fatalf("Could not connect to docker: %s", err)
	}

	// app logging

	zerolog.SetGlobalLevel(zerolog.FatalLevel)
	if os.Getenv("DEBUG") != "" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	// pipe logging to t.Log
	b := bytes.NewBuffer(nil)

	tc.logWG.Add(1)
	go func() {
		defer tc.logWG.Done()

		readAllAndExit := false

		for {
			l, e := b.ReadString('\n')
			if e != nil && e != io.EOF {
				t.Fatal("log piping error: ", e)
			}
			if e == io.EOF && readAllAndExit {
				return
			}
			if l != "" {
				t.Log(strings.TrimRight("[LOG] "+l, "\n"))
			}

			if readAllAndExit {
				continue
			}

			select {
			case <-tc.ctx.Done():
				readAllAndExit = true
				continue
			default:

			}
			time.Sleep(1 * time.Millisecond)
		}
	}()
}

func (tc *TestContext) TearDown(t testing.TB) {
	t.Log("TearDown")

	tc.ctxCancel()

	if err := tc.dockerPool.Purge(tc.dbRes); err != nil {
		t.Fatalf("Could not purge resource: %s", err)
	}
	tc.dbRes = nil

	tc.logWG.Wait()
}

type testData struct {
	NameSpace    string
	FunctionName string
	Context      string
	Arguments    string
	Response     string
	Error        bool
}

func TestApi_Call(t *testing.T) {
	tctx := TestContext{}
	tctx.SetUp(t)
	defer tctx.TearDown(t)

	goroutinesCnt := runtime.NumGoroutine()
	myRpc, err := rpc_redis.New("tcp", tctx.redisAddr, 8, 60*time.Second)
	myPS, err := events_redis.New("tcp", tctx.redisAddr)
	myReg := memRepo.New()
	clientCfg := client.Config{
		RPC:      myRpc,
		PubSub:   myPS,
		Registry: myReg,
		Logger:   testlogger.New(t),
	}
	srv, err := server.NewServer(server.Config{
		RPC:      myRpc,
		PubSub:   myPS,
		Registry: myReg,
		Logger:   testlogger.New(t),
	})
	require.NoError(t, err, "NewServer failed")

	servi := serv{}
	s, err := RegisterSessionInternalAPIServiceServer(&servi, srv)
	sc, err := NewSessionInternalAPIServiceClient(clientCfg)

	var waitForSvc sync.WaitGroup
	waitForSvc.Add(1)
	s.WatchInstanceRegistered(func(instanceId registry.InstanceId) {
		waitForSvc.Done()
	})

	go func() {
		var srvStartErr error
		srvStartErr = srv.Start()
		require.NoError(t, srvStartErr, "server start error")
	}()

	waitForSvc.Wait()

	// -----------
	var eventReceived int64

	stopEventSubscribing, err := sc.SubscribeConnectEvent(func(context context.Context, event *ConnectEvent) error {
		t.Log("EVENT SUBSCRIBER: got event:", event)
		atomic.AddInt64(&eventReceived, 1)
		return nil
	})
	require.NoError(t, err)

	t.Log("publishing event")
	err = s.PublishConnectEvent(context.Background(), &ConnectEvent{
		SessionId: "9",
		Kind:      ConnectEvent_CONNECTED,
	})
	t.Log("event published")

	resp, err := sc.Connect(nil, &ConnectReq{
		Id:            "9",
		UserId:        8,
		Ip:            "7",
		Os:            "6",
		Version:       "5",
		Build:         "4",
		LastChangedAt: 3,
		PushTokenText: "2",
	})
	stopEventSubscribing()
	require.NoError(t, err, "rpc call error")
	require.Equal(t, int64(10), resp.Value)

	require.Eventually(t,
		func() bool {
			return atomic.LoadInt64(&eventReceived) == 1
		},
		2*time.Second,
		100*time.Millisecond,
		"event not received in subscriber",
	)

	srv.Stop()
	myRpc.Close()
	myPS.Close()

	require.Eventually(
		t,
		func() bool {
			return runtime.NumGoroutine() == goroutinesCnt
		},
		8*time.Second,
		time.Second,
		"some goroutines are still running: %d", runtime.NumGoroutine()-goroutinesCnt,
	)
}

type serv struct {
}

func (s serv) Connect(ctx context.Context, req *ConnectReq) (resp *ConnectResp, err error) {
	resp = &ConnectResp{Value: 10}
	return resp, nil
}

func (s serv) Disconnect(ctx context.Context, req *StringMsg) (msg2 *BoolMsg, err error) {
	panic("implement me")
}
