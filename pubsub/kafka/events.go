package kafka

import (
	"context"
	"fmt"
	"sync"

	"github.com/Shopify/sarama"

	"github.com/f0mster/micro/pubsub"
)

type Events struct {
	consumerGroup      sarama.ConsumerGroup
	syncProducer       sarama.SyncProducer
	client             sarama.Client
	subscribeCancelMap map[string][]*pubsub.CancelFunc
	m                  sync.RWMutex
	config             *sarama.Config
	topics             map[string]bool
}

func New(config *sarama.Config, brokers []string, consumerGroup string) (*Events, error) {
	client, err := sarama.NewClient(brokers, config)
	cg, err := sarama.NewConsumerGroupFromClient(consumerGroup, client)
	if err != nil {
		return nil, err
	}
	producer, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		return nil, err
	}
	return &Events{
		consumerGroup:      cg,
		client:             client,
		syncProducer:       producer,
		subscribeCancelMap: map[string][]*pubsub.CancelFunc{},
		config:             config,
		topics:             map[string]bool{},
	}, nil
}

func (r *Events) Close() {
	// todo
}

func (r *Events) Publish(namespace string, eventName string, eventData []byte) (err error) {
	topic := namespace + "." + eventName

	r.m.Lock()
	if !r.topics[topic] {
		// Setup the Topic details in CreateTopicRequest struct
		topicDetail := &sarama.TopicDetail{}
		topicDetail.NumPartitions = int32(1)
		topicDetail.ReplicationFactor = int16(1)
		topicDetail.ConfigEntries = make(map[string]*string)

		topicDetails := make(map[string]*sarama.TopicDetail)
		topicDetails[topic] = topicDetail

		// Send request to Broker
		if ok, err := r.client.Brokers()[0].Connected(); err != nil {
			panic(err)
		} else if !ok {
			err = r.client.Brokers()[0].Open(r.config)
			if err != nil {
				panic(err)
			}
		}

		admin, err := sarama.NewClusterAdminFromClient(r.client)
		topics, err := admin.ListTopics()
		if _, ok := topics[topic]; !ok {
			err = admin.CreateTopic(topic, topicDetail, true)
			if err != nil {
				panic(err)
			}
		}
		r.topics[topic] = true
	}
	r.m.Unlock()
	_, _, err = r.syncProducer.SendMessage(&sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(eventData),
	})
	if err != nil {
		return err
	}
	return
}

func (r *Events) Unsubscribe(namespace string, eventName string) {
	topic := namespace + "." + eventName
	r.m.Lock()
	for i := range r.subscribeCancelMap[topic] {
		(*r.subscribeCancelMap[topic][i])()
	}
	r.subscribeCancelMap[topic] = r.subscribeCancelMap[topic][0:0]
	r.m.Unlock()
}

func (r *Events) Subscribe(namespace string, eventName string, callback func(event []byte) error) (cancel pubsub.CancelFunc, err error) {
	topic := namespace + "." + eventName
	ctx, ctxcancel := context.WithCancel(context.Background())
	cancel = func() {
		ctxcancel()
	}
	c := Consumer{
		cb:    callback,
		ready: make(chan bool),
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			if err := r.consumerGroup.Consume(ctx, []string{topic}, &c); err != nil {
				panic(fmt.Sprintf("Error from consumer: %v", err))
			}
			// check if context was cancelled, signaling that the consumer should stop
			if ctx.Err() != nil {
				return
			}
			c.ready = make(chan bool)
		}
	}()

	<-c.ready // Await till the consumer has been set up	r.m.Lock()
	r.m.Lock()
	i := 0
	if r.subscribeCancelMap[topic] == nil {
		r.subscribeCancelMap[topic] = make([]*pubsub.CancelFunc, 0)
	}
	l := len(r.subscribeCancelMap[topic])
	for ; i <= l; i++ {
		if i == l {
			r.subscribeCancelMap[topic] = append(r.subscribeCancelMap[topic], &cancel)
		} else if r.subscribeCancelMap[topic][i] == nil {
			r.subscribeCancelMap[topic][i] = &cancel
			break
		}
	}
	r.m.Unlock()
	return func() {
		r.m.Lock()
		cancel()
		if r.subscribeCancelMap[topic][i] == &cancel {
			r.subscribeCancelMap[topic][i] = nil
		}
		r.m.Unlock()
	}, nil
}

// Consumer represents a Sarama consumer group consumer
type Consumer struct {
	ready chan bool
	cb    func(event []byte) error
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (consumer *Consumer) Setup(sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	close(consumer.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (consumer *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (consumer *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {

	// NOTE:
	// Do not move the code below to a goroutine.
	// The `ConsumeClaim` itself is called within a goroutine, see:
	// https://github.com/Shopify/sarama/blob/master/consumer_group.go#L27-L29
	for message := range claim.Messages() {
		if err := consumer.cb(message.Value); err != nil {
			return err
		}
		session.MarkMessage(message, "")
	}
	return nil
}
