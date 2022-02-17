package main

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/Shopify/sarama"

	"github.com/f0mster/micro/examples/pubsub/api"
	pkgpubsub "github.com/f0mster/micro/pkg/pubsub"
	events_kafka "github.com/f0mster/micro/pkg/pubsub/kafka"
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
	wg.Wait()
	time.Sleep(1 * time.Second)
	cancel()
}

var config = sarama.NewConfig()
var pubsub *events_kafka.Events
var wg sync.WaitGroup

func serverSide() {
	// structure that can handle server functions
	ctx := context.Background()
	publisher := api.NewSessionInternalAPIEventsPublisher(pubsub)
	for i := 0; i < 1000; i++ {
		err := publisher.PublishConnectEvent(ctx, &api.ConnectEvent{Id: strconv.Itoa(i)})
		if err != nil {
			panic(err)
		}
	}
}

func clientSide() pkgpubsub.CancelFunc {
	sub := api.NewSessionInternalAPIEventsSubscriber(pubsub)
	wg.Add(1)
	cancel, err := sub.SubscribeConnectEvent(func(context context.Context, event *api.ConnectEvent) error {
		fmt.Println("connect event happens" + event.Id)
		wg.Done()
		return nil
	})
	if err != nil {
		panic(err)
	}
	return cancel
}
