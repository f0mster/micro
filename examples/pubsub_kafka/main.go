package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Shopify/sarama"

	"github.com/f0mster/micro/client"
	"github.com/f0mster/micro/examples/pubsub/api"
	pubsub2 "github.com/f0mster/micro/pubsub"
	events_kafka "github.com/f0mster/micro/pubsub/kafka"
	registry_memory "github.com/f0mster/micro/registry/memory"
	rpc_memory "github.com/f0mster/micro/rpc/memory"
	"github.com/f0mster/micro/server"
)

func main() {
	var err error
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Partitioner = sarama.NewRandomPartitioner
	config.Producer.Return.Successes = true
	pubsub, err = events_kafka.New(config, []string{"127.0.0.1:29092"}, "ddd")
	if err != nil {
		panic(fmt.Errorf("failed to connect to kafka: %w", err))
	}
	cancel := clientSide()
	serverSide()
	time.Sleep(5 * time.Second)
	cancel()
}

var rpc = rpc_memory.New(60 * time.Second)
var config = sarama.NewConfig()
var pubsub *events_kafka.Events
var reg = registry_memory.New()

func serverSide() {
	// structure that can handle server functions
	servi := serv{}
	serverCfg := server.Config{Registry: reg, RPC: rpc, PubSub: pubsub}
	srv, _ := server.NewServer(serverCfg)
	serviceSession, _ := api.RegisterSessionInternalAPIServer(&servi, srv)
	ctx := context.Background()

	for i := 0; i < 1000; i++ {
		err := serviceSession.PublishConnectEvent(ctx, &api.ConnectEvent{Id: strconv.Itoa(i)})
		if err != nil {
			panic(err)
		}
	}

}

func clientSide() pubsub2.CancelFunc {
	clientCfg := client.Config{Registry: reg, RPC: rpc, PubSub: pubsub}

	// this is a client
	clientSession, err := api.NewSessionInternalAPIClient(clientCfg)
	if err != nil {
		panic(err)
	}
	cancel, err := clientSession.SubscribeConnectEvent(func(context context.Context, event *api.ConnectEvent) error {
		fmt.Println("connect event happens" + event.Id)
		return nil
	})
	if err != nil {
		panic(err)
	}
	return cancel
}

type serv struct {
	srv *api.SessionInternalAPIService
}
