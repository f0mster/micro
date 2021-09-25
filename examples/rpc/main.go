package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/f0mster/micro/client"
	events_memory "github.com/f0mster/micro/pubsub/memory"
	rpc_memory "github.com/f0mster/micro/rpc/memory"
	"github.com/f0mster/micro/server"

	"github.com/f0mster/micro/examples/rpc/api"
	registry_memory "github.com/f0mster/micro/registry/memory"
)

var rpc = rpc_memory.New(60 * time.Second)
var pubsub = events_memory.New()
var reg = registry_memory.New()

func startServer(afterStart func()) *server.Server {
	var srv *server.Server
	srv, err := server.NewServer(server.Config{
		Registry: reg,
		RPC:      rpc,
		PubSub:   pubsub,
		AfterStart: func() error {
			afterStart()
			return nil
		},
	})
	// structure that can handle server functions
	servi := serv{}
	_, err = api.RegisterSessionInternalAPIServer(&servi, srv)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		err = srv.Start()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("server stopped")
	}()
	return srv
}

func main() {
	// this is client
	clientCfg := client.Config{PubSub: pubsub, RPC: rpc, Registry: reg}
	clientSession, err := api.NewSessionInternalAPIClient(clientCfg)
	if err != nil {
		log.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	srv := startServer(func() {
		wg.Done()
	})
	wg.Wait()
	req := api.ConnectReq{}
	resp, _ := clientSession.Connect(nil, &req)
	fmt.Println(resp)
	srv.Stop()
}

type serv struct {
	srv *api.SessionInternalAPIService
}

func (s *serv) Connect(ctx context.Context, req *api.ConnectReq) (resp *api.ConnectResp, err error) {
	resp = &api.ConnectResp{Value: 10}
	fmt.Println("function connect called")
	return
}
