package redis_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mediocregopher/radix/v3"
	"github.com/ory/dockertest/v3"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/f0mster/micro/pkg/rnd"
	"github.com/f0mster/micro/pubsub"
	events_redis "github.com/f0mster/micro/pubsub/redis"
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
	//}

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

func generateRandomString(l int) string {
	a, _ := rnd.GenerateRandomString(12)
	return a
}

type testData struct {
	NameSpace string
	EventName string
	Context   string
	EventData string
	cnt       int
}

func TestRedisRpc_Call(t *testing.T) {
	tctx := TestContext{}
	tctx.SetUp(t)
	defer tctx.TearDown(t)

	r, err := events_redis.New("tcp", tctx.redisAddr)
	NumGoroutine := runtime.NumGoroutine()

	require.NoError(t, err)

	done := make([]func(), 0)
	mu := sync.Mutex{}
	datas := map[string]map[string]*testData{}
	wg := sync.WaitGroup{}

	for i := 0; i < 10; i++ {
		ns := generateRandomString(20)
		mu.Lock()
		for z := 0; z < 10; z++ {
			eventName := generateRandomString(20)
			datas[ns+eventName] = map[string]*testData{}
			callbackPerEvent := 10
			for j := 0; j < callbackPerEvent; j++ {
				var cancel pubsub.CancelFunc
				cancel, err = r.Subscribe(ns, eventName, func(context []byte, arguments []byte) error {
					mu.Lock()
					if _, ok := datas[ns+eventName]; !ok {
						t.Fatal("no element", ns+eventName)
					}
					if _, ok := datas[ns+eventName][string(arguments)]; !ok {
						t.Fatal("no element", ns+eventName, string(arguments))
					}
					el := datas[ns+eventName][string(arguments)]
					require.Equal(t, el.Context, string(context), el)
					require.Equal(t, el.EventData, string(arguments))
					el.cnt++
					if el.cnt >= callbackPerEvent {
						delete(datas[ns+eventName], el.EventData)
						if len(datas[ns+eventName]) == 0 {
							delete(datas, ns+eventName)
						}
					}
					mu.Unlock()
					wg.Done()
					return nil
				})
				require.NoError(t, err, "Subscribe")
				done = append(done, cancel)
			}
			for j := 0; j < 100; j++ {
				wg.Add(1 * callbackPerEvent)
				eventData := ""
				for {
					eventData = generateRandomString(24)
					if _, ok := datas[ns+eventName][eventData]; !ok {
						break
					}
				}
				td := testData{
					NameSpace: ns,
					EventName: eventName,
					Context:   generateRandomString(20),
					EventData: generateRandomString(22),
				}
				datas[ns+eventName][td.EventData] = &td
				go func() {
					err := r.Publish(td.NameSpace, td.EventName, []byte(td.Context), []byte(td.EventData))
					require.NoError(t, err)
				}()
			}
		}
		mu.Unlock()
	}
	wg.Wait()
	require.Equal(t, 0, len(datas), "something left")
	for i := range done {
		done[i]()
	}
	if runtime.NumGoroutine() > NumGoroutine {
		t.Fatal("have", runtime.NumGoroutine()-NumGoroutine, "unclosed goroutines!")
	}
}
