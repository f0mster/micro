package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	rpc_memory "github.com/f0mster/micro/pkg/rpc/memory"
	"github.com/f0mster/micro/pkg/server"

	"github.com/f0mster/micro/pkg/client"
	"github.com/f0mster/micro/pkg/metadata"
	events_memory "github.com/f0mster/micro/pkg/pubsub/memory"

	"github.com/f0mster/micro/examples/rpc/api"
	registry_memory "github.com/f0mster/micro/pkg/registry/memory"
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
	ctx := metadata.NewContext(context.Background(), metadata.Metadata{"2222key": "222112val"})
	resp, _ := clientSession.Connect(ctx, &req)
	fmt.Printf("\nresp: '%+v'", resp)
	srv.Stop()
}

type serv struct {
	srv *api.SessionInternalAPIService
}

func (s *serv) SnakeFuncName(ctx context.Context, req *api.SnakeMessage) (resp *api.SnakeMessage, err error) {
	panic("implement me")
}

func (s *serv) Connect(ctx context.Context, req *api.ConnectReq) (resp *api.ConnectResp, err error) {
	resp = &api.ConnectResp{Value: 10}
	meta, ok := metadata.FromContext(ctx)
	if !ok {
		panic("no meta data in context")
	}
	fmt.Printf("function connect called, got context '%+v'", meta)
	return
}
