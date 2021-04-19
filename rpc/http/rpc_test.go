package rpc_redis_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/rs/zerolog"
)

/*
  Внимание!

  Для запуска необходим docker.

  Адрес указывается через переменную окружения DOCKER_HOST.

*/

var appIndex int32

type TestContext struct {
	httpAddr string

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
		"nginx",
		"latest",
		nil,
	); e != nil {
		t.Fatalf("Could not start resource: %s", e)
	} else {
		tc.dbRes = r
	}

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	tc.httpAddr = "http://" + getAddr(tc.dockerPool.Client.Endpoint(), tc.dbRes.GetPort("80/tcp"))
	if err := tc.dockerPool.Retry(func() error {
		_, err := http.Get(tc.httpAddr)
		fmt.Println(err, tc.httpAddr)
		return err
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

func TestRedisRpc_Call3(b *testing.T) {
	tctx := TestContext{}
	tctx.SetUp(b)
	defer tctx.TearDown(b)
	a := time.Now().UnixNano()
	times := int64(10000)
	goroutines := int64(10000)
	wg := sync.WaitGroup{}
	for i := int64(0); i < goroutines; i++ {
		wg.Add(1)
		go func() {
			for j := int64(0); j < times/goroutines; j++ {
				_, _ = http.Get(tctx.httpAddr)
				//_, _ = http.Get("http://inteart.ru")
			}
			wg.Done()
		}()
	}
	wg.Wait()
	diff := (time.Now().UnixNano() - a)
	fmt.Println("total time", diff, "ns; one run", diff/times, "ns; times per second", int64(time.Second)/(diff/times))
}
