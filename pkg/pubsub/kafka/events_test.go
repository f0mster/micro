package kafka_test

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/require"

	"github.com/f0mster/micro/pkg/pubsub"
	"github.com/f0mster/micro/pkg/pubsub/kafka"
	"github.com/f0mster/micro/pkg/rnd"
)

/*
  Внимание!

  Для запуска необходим docker.

  Адрес указывается через переменную окружения DOCKER_HOST.

*/

const kafkaContainerName = "micro-docker-test-kafka"

var cgCancel context.CancelFunc

var appIndex int32

func generateRandomString(l int) string {
	a, _ := rnd.GenerateRandomString(12)
	return a
}

type testData struct {
	NameSpace string
	EventName string
	EventData string
	cnt       int
}

func TestKafkaPubSub(t *testing.T) {
	callbackPerEvent := 1
	numOfNameSpaces := 10
	numOfEventNames := 10
	numOfMessagesPerTopic := 1000

	broker, err := initKafka()
	require.NoError(t, err)
	r, err := kafka.New(config(), []string{broker}, "cg1")
	require.NoError(t, err)
	NumGoroutine := runtime.NumGoroutine()

	done := make([]func(), 0)
	mu := sync.Mutex{}
	datas := map[string]map[string]*testData{}
	wg := sync.WaitGroup{}
	var topics = make([]string, 0, 100)
	mu.Lock()
	runs := int64(0)
	for i := 0; i < numOfNameSpaces; i++ {
		ns := strconv.Itoa(i) + generateRandomString(20)
		for z := 0; z < numOfEventNames; z++ {
			eventName := strconv.Itoa(z) + generateRandomString(20)
			topic := ns + ":" + eventName
			topics = append(topics, topic)
			datas[topic] = map[string]*testData{}
			for j := 0; j < callbackPerEvent; j++ {
				var cancel pubsub.CancelFunc
				log.Println("let's try to subscribe")
				cancel, err = r.Subscribe(ns, eventName, func(arguments []byte) error {
					vaL := atomic.AddInt64(&runs, 1)
					fmt.Println("func run", vaL)
					mu.Lock()
					defer mu.Unlock()
					defer wg.Done()
					if _, ok := datas[topic]; !ok {
						t.Fatal("no element", topic)
					}
					if _, ok := datas[topic][string(arguments)]; !ok {
						t.Fatal("no element", topic, string(arguments))
					}
					el := datas[topic][string(arguments)]
					require.Equal(t, el.EventData, string(arguments))
					el.cnt++
					if el.cnt >= callbackPerEvent {
						delete(datas[topic], el.EventData)
						if len(datas[topic]) == 0 {
							delete(datas, topic)
						}
					}
					return nil
				})
				require.NoError(t, err)
				log.Println("subscribe complete")
				done = append(done, cancel)
			}
		}
	}
	mu.Unlock()
	for i := range topics {
		parts := strings.Split(topics[i], ":")
		for j := 0; j < numOfMessagesPerTopic; j++ {
			wg.Add(1 * callbackPerEvent)
			eventData := ""
			for {
				eventData = generateRandomString(24)
				if _, ok := datas[topics[i]][eventData]; !ok {
					break
				}
			}
			td := testData{
				NameSpace: parts[0],
				EventName: parts[1],
				EventData: generateRandomString(22),
			}
			datas[topics[i]][td.EventData] = &td
			go func() {
				err := r.Publish(td.NameSpace, td.EventName, []byte(td.EventData))
				require.NoError(t, err)
			}()
		}
	}

	wg.Wait()
	require.Equal(t, 0, len(datas), "something left")
	for i := range done {
		done[i]()
	}

	time.Sleep(5 * time.Second)
	if NumGoroutine < runtime.NumGoroutine() {
		//TODO: AID-139 find out why sarama produce so many goroutines and doesn't close them
	}
}

func initKafka() (broker string, err error) {
	dockerPool, err := dockertest.NewPool("")
	resource, ok := dockerPool.ContainerByName(kafkaContainerName)
	if ok {
		resource.Close()
	}
	if err != nil {
		return "", fmt.Errorf("could not start kafka: %s", err)
	}
	kafkaResource, err := dockerPool.RunWithOptions(&dockertest.RunOptions{
		Name:       kafkaContainerName,
		Repository: "johnnypark/kafka-zookeeper",
		Tag:        "2.6.0",
		Hostname:   "kafka",
		Env: []string{
			"ADVERTISED_HOST=127.0.0.1",
			"NUM_PARTITIONS=1",
		},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"9092/tcp": {{HostIP: "localhost", HostPort: "9092/tcp"}},
		},
	}, removeAndRestart)
	if err != nil {
		return "", fmt.Errorf("could not start kafka: %s", err)
	}
	addr := kafkaResource.GetHostPort("9092/tcp")
	err = dockerPool.Retry(func() error {

		fmt.Println("retry", addr)
		_, err = sarama.NewClient([]string{addr}, config())
		if err != nil {
			fmt.Println("client error")
			return err
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("could not connect to kafka: %s", err)
	}
	defer func() {
		if cgCancel != nil {
			cgCancel()
			cgCancel = nil
		}
	}()

	if err != nil {
		return "", err
	}
	fmt.Println("kafka connected")
	return addr, nil
}

func removeAndRestart(config *docker.HostConfig) {
	// Set AutoRemove to true so that stopped container goes away by itself.
	config.AutoRemove = true
	config.RestartPolicy = docker.RestartPolicy{
		Name: "no",
	}
}

func config() *sarama.Config {
	config := sarama.NewConfig()
	config.Admin.Retry.Max = 10
	config.Admin.Retry.Backoff = 10 * time.Second
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Partitioner = sarama.NewRandomPartitioner
	config.Producer.Return.Successes = true
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRange
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	return config
}

type t struct {
	cancel context.CancelFunc
}

func TestSsss(t *testing.T) {
	ctx1, ctxcancel1 := context.WithCancel(context.WithValue(context.Background(), "1", generateRandomString(5)))
	ctx2, ctxcancel2 := context.WithCancel(context.WithValue(context.Background(), "1", generateRandomString(5)))
	ctx3, ctxcancel3 := context.WithCancel(context.WithValue(context.Background(), "1", generateRandomString(5)))
	go func(ctx1 context.Context, ctx2 context.Context, ctx3 context.Context) {
		for {
			select {
			case <-ctx1.Done():
				fmt.Println("ctx1")
			case <-ctx2.Done():
				fmt.Println("ctx2")
			case <-ctx3.Done():
				fmt.Println("ctx3")
			}
		}
	}(ctx1, ctx2, ctx3)
	_, _, _ = ctxcancel1, ctxcancel2, ctxcancel3
	ctxcancel2()
	time.Sleep(1 * time.Second)
}
