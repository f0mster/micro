package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/f0mster/micro/client"
	"github.com/f0mster/micro/examples/pubsub/api"
	pubsub2 "github.com/f0mster/micro/pubsub"
	events_memory "github.com/f0mster/micro/pubsub/memory"
	registry_memory "github.com/f0mster/micro/registry/memory"
	rpc_memory "github.com/f0mster/micro/rpc/memory"
	"github.com/f0mster/micro/server"
)

func main() {
	cancel := clientSide()
	serverSide()
	wg.Wait()
	cancel()
}

var rpc = rpc_memory.New(60 * time.Second)
var pubsub = events_memory.New()
var reg = registry_memory.New()

func serverSide() {
	// structure that can handle server functions
	servi := serv{}
	serverCfg := server.Config{Registry: reg, RPC: rpc, PubSub: pubsub}
	srv, _ := server.NewServer(serverCfg)
	serviceSession, _ := api.RegisterSessionInternalAPIServer(&servi, srv)
	ctx := context.Background()
	err := serviceSession.PublishConnectEvent(ctx, &api.ConnectEvent{Id: "wqdqwd"})
	if err != nil {
		log.Fatal(err)
	}
}

func clientSide() pubsub2.CancelFunc {
	clientCfg := client.Config{Registry: reg, RPC: rpc, PubSub: pubsub}

	// this is a client
	clientSession, err := api.NewSessionInternalAPIClient(clientCfg)
	if err != nil {
		log.Fatal(err)
	}
	wg.Add(1)
	cancel, err := clientSession.SubscribeConnectEvent(func(context context.Context, event *api.ConnectEvent) error {
		fmt.Println("connect event happens")
		wg.Done()
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return cancel
}

var wg sync.WaitGroup

type serv struct {
	srv *api.SessionInternalAPIService
}
