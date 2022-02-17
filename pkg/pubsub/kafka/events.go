package kafka

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/Shopify/sarama"

	"github.com/f0mster/micro/pkg/pubsub"
)

var ErrNoActiveBrokers = errors.New("failed to find active brokers")

type Events struct {
	consumerGroup      string
	syncProducer       sarama.SyncProducer
	client             sarama.Client
	subscribeCancelMap map[string]*cancelMap
	m                  sync.RWMutex
	config             *sarama.Config
	topics             map[string]bool
	prefix             string
	consumer           Consumer
}

type cancelMap struct {
	lastNumber uint
	data       map[uint]*pubsub.CancelFunc
}

var _ pubsub.PubSub = (*Events)(nil)

func New(config *sarama.Config, brokers []string, consumerGroup string) (*Events, error) {
	return NewWithPrefix(config, brokers, consumerGroup, "")
}

func NewWithPrefix(config *sarama.Config, brokers []string, consumerGroup string, prefix string) (*Events, error) {
	client, err := sarama.NewClient(brokers, config)
	if err != nil {
		return nil, err
	}
	producer, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		return nil, err
	}
	return &Events{
		consumerGroup:      consumerGroup,
		client:             client,
		syncProducer:       producer,
		subscribeCancelMap: map[string]*cancelMap{},
		config:             config,
		topics:             map[string]bool{},
		prefix:             prefix,
		consumer: Consumer{
			topics:        map[string]func(event []byte) error{},
			cancelConsume: nil,
		},
	}, nil
}

func (r *Events) Close() {
	// todo
}

func (r *Events) PublishToTopic(topic string, eventData []byte) (err error) {
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
		brokers := r.client.Brokers()
		if len(brokers) < 1 {
			r.m.Unlock()
			return ErrNoActiveBrokers
		} else if ok, err := brokers[0].Connected(); err != nil {
			r.m.Unlock()
			return err
		} else if !ok {
			err = brokers[0].Open(r.config)
			if err != nil {
				r.m.Unlock()
				return err
			}
		}

		admin, err := sarama.NewClusterAdminFromClient(r.client)
		topics, err := admin.ListTopics()
		if _, ok := topics[topic]; !ok {
			err = admin.CreateTopic(topic, topicDetail, true)
			var se *sarama.TopicError
			if err != nil && !(errors.As(err, &se) && se.Err == sarama.ErrTopicAlreadyExists) {
				r.m.Unlock()
				return err
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

func (r *Events) UnsubscribeFromTopic(topic string) {
	r.m.Lock()
	for i := range r.subscribeCancelMap[topic].data {
		(*r.subscribeCancelMap[topic].data[i])()
		delete(r.subscribeCancelMap[topic].data, i)
		delete(r.topics, topic)
		r.restartConsume()
	}
	r.m.Unlock()
}

func (r *Events) restartConsume() {
	log.Println("obtaining lock consumer")
	r.consumer.mu.Lock()
	defer r.consumer.mu.Unlock()
	log.Println("got lock consumer")
	if r.consumer.cg != nil {
		log.Println("close consumergroup")
		r.consumer.cg.Close()
		log.Println("wait from cancel")
		r.consumer.startWG.Wait()
		return
	}
	ctx, ctxcancel := context.WithCancel(context.Background())
	r.consumer.cancelConsume = ctxcancel
	log.Println("ctx cancel func", &r.consumer.cancelConsume)
	r.consumer.startWG.Add(1)
	go func(ctx context.Context) {
		for {
			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			log.Println("consuming topic:", r.consumer.topics)
			topics := make([]string, 0, len(r.consumer.topics))
			for name := range r.consumer.topics {
				topics = append(topics, name)
			}
			log.Println("going to consume", topics)
			var err error
			r.consumer.cg, err = sarama.NewConsumerGroupFromClient(r.consumerGroup, r.client)
			if err != nil {
				log.Println("can't connect")
				return
			}
			if err := r.consumer.cg.Consume(ctx, topics, &r.consumer); err != nil {
				panic(fmt.Sprintf("Error from consumer: %v", err))
			}
			// check if context was cancelled, signaling that the consumer should stop
			log.Println("stop consuming topic:", topics)
			if ctx.Err() != nil {
				r.consumer.cg = nil
				return
			}
			r.consumer.startWG.Add(1)

		}
	}(ctx)
	r.consumer.startWG.Wait() // Await till the consumer has been set up
	log.Println("wait done")
}

func (r *Events) SubscribeForTopic(topic string, callback func(event []byte) error) (cancel pubsub.CancelFunc, err error) {
	log.Println("obtaining lock")
	r.m.Lock()
	defer r.m.Unlock()
	log.Println("got lock")

	r.consumer.topics[topic] = callback
	wg := &sync.WaitGroup{}
	wg.Add(1)
	if r.subscribeCancelMap[topic] == nil {
		r.subscribeCancelMap[topic] = &cancelMap{data: map[uint]*pubsub.CancelFunc{}}
	}
	r.subscribeCancelMap[topic].lastNumber++
	i := r.subscribeCancelMap[topic].lastNumber
	r.subscribeCancelMap[topic].data[i] = &cancel
	log.Println("restarting consume")
	r.restartConsume()
	// TODO: fix cancel func
	return func() {
		r.m.Lock()
		//cancel()
		if r.subscribeCancelMap[topic].data[i] == &cancel {
			delete(r.subscribeCancelMap[topic].data, i)
		}
		left := len(r.subscribeCancelMap[topic].data)
		r.m.Unlock()
		if left == 0 {
			r.UnsubscribeFromTopic(topic)
		}
	}, nil
}

// Consumer represents a Sarama consumer group consumer
type Consumer struct {
	mu            sync.Mutex
	startWG       sync.WaitGroup
	topics        map[string]func(event []byte) error
	cancelConsume context.CancelFunc
	cg            sarama.ConsumerGroup
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (consumer *Consumer) Setup(sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	fmt.Printf("setup ok for len %v topic %+v\n", len(consumer.topics), consumer.topics)
	consumer.startWG.Done()
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
		log.Println("got message", message.Topic, message.Value)
		if err := consumer.topics[message.Topic](message.Value); err != nil {
			return err
		}
		session.MarkMessage(message, "")
	}
	return nil
}

func (r *Events) Subscribe(namespace string, eventName string, callback func(event []byte) error) (pubsub.CancelFunc, error) {
	topic := r.prefix + namespace + "." + eventName
	return r.SubscribeForTopic(topic, callback)
}

func (r *Events) Unsubscribe(namespace string, eventName string) {
	topic := r.prefix + namespace + "." + eventName
	r.UnsubscribeFromTopic(topic)
}

func (r *Events) Publish(namespace string, eventName string, event []byte) error {
	topic := r.prefix + namespace + "." + eventName
	return r.PublishToTopic(topic, event)
}
