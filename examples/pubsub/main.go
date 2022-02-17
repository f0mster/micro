package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/f0mster/micro/examples/pubsub/api"
	"github.com/f0mster/micro/pkg/metadata"
	pkgpubsub "github.com/f0mster/micro/pkg/pubsub"
	events_memory "github.com/f0mster/micro/pkg/pubsub/memory"
)

func main() {
	cancel := clientSide()
	serverSide()
	wg.Wait()
	cancel()
}

var pubsub = events_memory.New()

func serverSide() {
	// structure that can handle server functions
	ctx := context.Background()
	ctx = metadata.NewContext(ctx, metadata.Metadata{"www": "111"})
	publisher := api.NewSessionInternalAPIEventsPublisher(pubsub)
	err := publisher.PublishConnectEvent(ctx, &api.ConnectEvent{Id: "wqdqwd"})
	if err != nil {
		log.Fatal(err)
	}
}

func clientSide() pkgpubsub.CancelFunc {
	sub := api.NewSessionInternalAPIEventsSubscriber(pubsub)
	wg.Add(1)
	cancel, err := sub.SubscribeConnectEvent(func(context context.Context, event *api.ConnectEvent) error {
		fmt.Println("connect event happens")
		meta, ok := metadata.FromContext(context)
		if !ok {
			panic("error getting data from context")
		}
		fmt.Printf("'%+v'", meta)
		fmt.Printf("'%+v'", event)
		wg.Done()
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return cancel
}

var wg sync.WaitGroup
