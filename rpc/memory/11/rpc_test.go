package memory_test

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"

	"github.com/f0mster/micro/pkg/rnd"
	"github.com/f0mster/micro/rpc/memory"
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
	b.ResetTimer()
	b.ReportAllocs()
	for j := 0; j < b.N; j++ {
		wg.Add(1)
		go func() {
			r.Call(td.NameSpace, td.FunctionName, []byte(td.Context), []byte(td.Arguments))
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
	times := int64(2000000)
	for j := int64(0); j < times; j++ {
		wg.Add(1)
		go func() {
			r.Call(td.NameSpace, td.FunctionName, []byte(td.Context), []byte(td.Arguments))
			wg.Done()
		}()
	}
	wg.Wait()
	diff := (time.Now().UnixNano() - a)
	fmt.Println("total time", diff, "ns; one run", diff/times, "ns; times per second", int64(time.Second)/(diff/times))
}
