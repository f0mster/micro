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
	rpc_redis "github.com/f0mster/micro/rpc/redis"
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
		t.Log("started ok")
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
	NameSpace    string
	FunctionName string
	Context      string
	Arguments    string
	Response     string
	Error        bool
}

func TestRedisRpc_Call(t *testing.T) {
	tctx := TestContext{}
	tctx.SetUp(t)
	defer tctx.TearDown(t)
	r, err := rpc_redis.New("tcp", tctx.redisAddr, 8, 60*time.Second)

	require.NoError(t, err)

	done := make([]func(), 10)
	mu := sync.Mutex{}
	datas := map[string]map[string]testData{}
	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		ns := generateRandomString(20)
		mu.Lock()
		datas[ns] = map[string]testData{}
		for j := 0; j < 10; j++ {
			done[i] = r.Listen(ns, func(functionName string, context []byte, arguments []byte) (response []byte, err error) {
				mu.Lock()
				defer mu.Unlock()
				require.Equal(t, datas[ns][functionName].Context, string(context))
				require.Equal(t, datas[ns][functionName].Arguments, string(arguments))
				resp := datas[ns][functionName].Response
				if datas[ns][functionName].Error {
					return nil, fmt.Errorf(string(resp))
				}
				return []byte(resp), nil
			})
		}
		for j := 0; j < 200; j++ {
			wg.Add(1)
			fn := ""
			for {
				fn = generateRandomString(24)
				if _, ok := datas[ns][fn]; !ok {
					break
				}
			}
			td := testData{
				NameSpace:    ns,
				FunctionName: fn,
				Context:      generateRandomString(20),
				Arguments:    generateRandomString(22),
				Response:     generateRandomString(23),
				Error:        time.Now().UnixNano()%1 == 0,
			}
			datas[ns][td.FunctionName] = td
			go func() {
				resp, err := r.Call(td.NameSpace, td.FunctionName, []byte(td.Context), []byte(td.Arguments))
				if td.Error {
					require.Equal(t, td.Response, err.Error(), td)
					require.Equal(t, []byte(nil), resp)
				} else {
					require.Equal(t, td.Response, string(resp))
					require.Equal(t, nil, err)
				}
				mu.Lock()
				delete(datas[ns], td.FunctionName)
				if len(datas[ns]) == 0 {
					delete(datas, ns)
				}
				mu.Unlock()
				wg.Done()
			}()
		}
		mu.Unlock()
	}
	fmt.Println("do")
	wg.Wait()
	fmt.Println("done")
	require.Equal(t, 0, len(datas), "something left")
}

func TestRedisRpc_Close(t *testing.T) {
	tctx := TestContext{}
	tctx.SetUp(t)
	defer tctx.TearDown(t)
	ng := runtime.NumGoroutine()
	r, err := rpc_redis.New("tcp", tctx.redisAddr, 8, 60*time.Second)
	require.NoError(t, err)
	r.Close()
	time.Sleep(10 * time.Second)
	if runtime.NumGoroutine() > ng {
		t.Fatal("lets unstopped go routines", runtime.NumGoroutine()-ng)
	}
}

func BenchmarkRedisRpc_Call(b *testing.B) {
	tctx := TestContext{}
	tctx.SetUp(b)
	defer tctx.TearDown(b)
	r, err := rpc_redis.New("tcp", tctx.redisAddr, 8, 60*time.Second)
	require.NoError(b, err)
	wg := sync.WaitGroup{}
	ns := generateRandomString(20)
	for j := 0; j < 10; j++ {
		r.Listen(ns, func(functionName string, context []byte, arguments []byte) (response []byte, err error) {
			return []byte("adadawd"), nil
		})
	}
	td := testData{
		NameSpace:    ns,
		FunctionName: "",
		Context:      "",
		Arguments:    "",
		Response:     "",
		Error:        time.Now().UnixNano()%1 == 0,
	}
	b.ReportAllocs()
	b.StartTimer()
	for j := 0; j < b.N; j++ {
		wg.Add(1)
		go func() {
			_, _ = r.Call(td.NameSpace, td.FunctionName, []byte(td.Context), []byte(td.Arguments))
			wg.Done()
		}()
	}
	wg.Wait()
	b.StopTimer()
}

func TestRedisRpc_CallBench(b *testing.T) {
	tctx := TestContext{}
	tctx.SetUp(b)
	defer tctx.TearDown(b)
	r, err := rpc_redis.New("tcp", tctx.redisAddr, 100, 60*time.Second)
	require.NoError(b, err)
	wg := sync.WaitGroup{}
	ns := generateRandomString(20)
	for j := 0; j < 100; j++ {
		r.Listen(ns, func(functionName string, context []byte, arguments []byte) (response []byte, err error) {
			return []byte("adadawd"), nil
		})
	}
	td := testData{
		NameSpace:    ns,
		FunctionName: "",
		Context:      "",
		Arguments:    "",
		Response:     "",
		Error:        time.Now().UnixNano()%1 == 0,
	}
	a := time.Now().UnixNano()
	times := int64(2000)
	for j := int64(0); j < times; j++ {
		wg.Add(1)
		go func() {
			_, _ = r.Call(td.NameSpace, td.FunctionName, []byte(td.Context), []byte(td.Arguments))
			wg.Done()
		}()
	}
	wg.Wait()
	diff := (time.Now().UnixNano() - a)
	fmt.Println("total time", diff, "ns; one run", diff/times, "ns; times per second", int64(time.Second)/(diff/times))
}

func TestRedisRpc_Call2(b *testing.T) {
	tctx := TestContext{}
	tctx.SetUp(b)
	defer tctx.TearDown(b)
	pool, err := radix.NewPool("tcp", tctx.redisAddr, 100)
	if err != nil {
		b.Fatal(err)
	}
	a := time.Now().UnixNano()
	times := int64(1000000)
	goroutines := int64(100000)
	wg := sync.WaitGroup{}
	for i := int64(0); i < goroutines; i++ {
		wg.Add(1)
		go func() {
			for j := int64(0); j < times/goroutines; j++ {
				err := pool.Do(radix.FlatCmd(nil, "RPUSH", "queue", string("qwdqwdqwd")))
				if err != nil {
					b.Fatal(err)
				}
				resp := ""
				err = pool.Do(radix.FlatCmd(&resp, "LPOP", "queue"))
				if err != nil {
					b.Fatal(err)
				}

			}
			wg.Done()
		}()
	}
	wg.Wait()

	diff := (time.Now().UnixNano() - a)
	fmt.Println("total time", diff, "ns; one run", diff/times, "ns; times per second", int64(time.Second)/(diff/times))
}
