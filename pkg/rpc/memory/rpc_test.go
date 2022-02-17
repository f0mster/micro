package memory_test

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"

	tests "github.com/f0mster/micro/internal/test"
	"github.com/f0mster/micro/pkg/rnd"
	"github.com/f0mster/micro/pkg/rpc/memory"
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

func TestInMemRpc_Call(t *testing.T) {
	r := memory.New(60 * time.Second)
	tests.Rpc_Call_Test(r, t)
}

func TestInMemRpc_Close(t *testing.T) {
	ng := runtime.NumGoroutine()
	r := memory.New(60 * time.Second)
	r.Close()
	time.Sleep(10 * time.Second)
	if runtime.NumGoroutine() > ng {
		t.Fatal("lets unstopped go routines", runtime.NumGoroutine()-ng)
	}
}

func BenchmarkRedisRpc_Call(b *testing.B) {
	r := memory.New(60 * time.Second)
	wg := sync.WaitGroup{}
	ns := generateRandomString(20)
	for j := 0; j < 10; j++ {
		r.Listen(ns, func(functionName string, arguments []byte) (response []byte, err error) {
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
	b.ResetTimer()
	b.ReportAllocs()
	for j := 0; j < b.N; j++ {
		wg.Add(1)
		go func() {
			_, _ = r.Call(td.NameSpace, td.FunctionName, []byte(td.Arguments))
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestInMemRpc_CallBench(b *testing.T) {
	r := memory.New(60 * time.Second)
	wg := sync.WaitGroup{}
	ns := generateRandomString(20)
	for j := 0; j < 100; j++ {
		r.Listen(ns, func(functionName string, arguments []byte) (response []byte, err error) {
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
	times := int64(2000000)
	for j := int64(0); j < times; j++ {
		wg.Add(1)
		go func() {
			_, _ = r.Call(td.NameSpace, td.FunctionName, []byte(td.Arguments))
			wg.Done()
		}()
	}
	wg.Wait()
	diff := (time.Now().UnixNano() - a)
	fmt.Println("total time", diff, "ns; one run", diff/times, "ns; times per second", int64(time.Second)/(diff/times))
}
