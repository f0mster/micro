package tests

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/f0mster/micro/pkg/rnd"
	"github.com/f0mster/micro/pkg/rpc"
)

type testData struct {
	NameSpace    string
	FunctionName string
	Arguments    string
	Response     string
	Error        bool
}

/*
  Внимание!

  Для запуска необходим docker.

  Адрес указывается через переменную окружения DOCKER_HOST.

*/

func generateRandomString(l int) string {
	a, _ := rnd.GenerateRandomString(12)
	return a
}

func Rpc_Call_Test(rpc rpc.RPC, t *testing.T) {
	done := make([]func(), 10)
	mu := sync.Mutex{}
	datas := map[string]map[string]testData{}
	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		ns := generateRandomString(20)
		mu.Lock()
		datas[ns] = map[string]testData{}
		for j := 0; j < 10; j++ {
			done[i] = rpc.Listen(ns, func(functionName string, arguments []byte) (response []byte, err error) {
				mu.Lock()
				defer mu.Unlock()
				require.Equal(t, datas[ns][functionName].Arguments, string(arguments))
				resp := datas[ns][functionName].Response
				if datas[ns][functionName].Error {
					return nil, fmt.Errorf(string(resp))
				}
				return []byte(resp), nil
			})
		}
		for j := 0; j < 2000; j++ {
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
				Arguments:    generateRandomString(22),
				Response:     generateRandomString(23),
				Error:        time.Now().UnixNano()%1 == 0,
			}
			datas[ns][td.FunctionName] = td
			go func() {
				resp, err := rpc.Call(td.NameSpace, td.FunctionName, []byte(td.Arguments))
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
