package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/f0mster/micro/client"
	events_memory "github.com/f0mster/micro/pubsub/memory"
	rpc_memory "github.com/f0mster/micro/rpc/memory"
	"github.com/f0mster/micro/server"

	"time"

	api "github.com/f0mster/micro/examples/pubsub/import"
	registry_memory "github.com/f0mster/micro/registry/memory"
)

func main() {
	rpc := rpc_memory.New(60 * time.Second)
	pubsub := events_memory.New()
	reg := registry_memory.New()

	// structure that can handle server functions
	servi := serv{}
	clientCfg := client.Config{Registry: reg, RPC: rpc, PubSub: pubsub}
	serverCfg := server.Config{Registry: reg, RPC: rpc, PubSub: pubsub}

	// this is client
	clientSession, err := api.NewSessionInternalAPIClient(clientCfg)
	if err != nil {
		log.Fatal(err)
	}
	// this is server
	srv, _ := server.NewServer(serverCfg)
	serviceSession, _ := api.RegisterSessionInternalAPIServer(&servi, srv)

	wg := sync.WaitGroup{}
	wg.Add(1)
	cancel, err := clientSession.SubscribeConnectEvent(func(context context.Context, event *api.ConnectEvent) error {
		fmt.Println("connect event happens")
		wg.Done()
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	err = serviceSession.PublishConnectEvent(ctx, &api.ConnectEvent{Id: "wqdqwd"})
	if err != nil {
		log.Fatal(err)
	}

	wg.Wait()
	cancel()
}

type serv struct {
	srv *api.SessionInternalAPIService
}
