package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/f0mster/micro/client"
	events_memory "github.com/f0mster/micro/pubsub/memory"
	rpc_memory "github.com/f0mster/micro/rpc/memory"
	"github.com/f0mster/micro/server"

	api "github.com/f0mster/micro/examples/rpc/import"
	registry_memory "github.com/f0mster/micro/registry/memory"
)

func main() {
	rpc := rpc_memory.New(60 * time.Second)
	pubsub := events_memory.New()
	reg := registry_memory.New()

	// structure that can handle server functions
	servi := serv{}
	clientCfg := client.Config{PubSub: pubsub, RPC: rpc, Registry: reg}
	clientSession, err := api.NewSessionInternalAPIClient(clientCfg)
	var srv *server.Server
	srv, err = server.NewServer(server.Config{
		Registry: reg,
		RPC:      rpc,
		PubSub:   pubsub,
		AfterStart: func() error {
			req := api.ConnectReq{}
			resp, _ := clientSession.Connect(nil, &req)
			fmt.Println(resp)
			_ = srv.Stop()
			return nil
		},
	})
	// this is client
	if err != nil {
		log.Fatal(err)
	}
	_, err = api.RegisterSessionInternalAPIServer(&servi, srv)
	if err != nil {
		log.Fatal(err)
	}

	err = srv.Start()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("server stopped")
}

type serv struct {
	srv *api.SessionInternalAPIService
}

func (s *serv) Connect(ctx context.Context, req *api.ConnectReq) (resp *api.ConnectResp, err error) {
	resp = &api.ConnectResp{Value: 10}
	fmt.Println("function connect called")
	return
}
