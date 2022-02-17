package memory

import (
	"sync"
	"sync/atomic"

	"github.com/f0mster/micro/pkg/pubsub"
)

type data struct {
	lastEl    int64
	callbacks map[int64]func(event []byte) error
	sync.Mutex
	done int32
}

type Events struct {
	subscribeMap   map[string]*data
	subscribeMutex sync.Mutex
}

func (r *Events) PublishToTopic(topic string, eventData []byte) error {
	go func() {
		r.subscribeMutex.Lock()
		r.initSubscribeMapTopic(topic)

		tmp := r.subscribeMap[topic]
		r.subscribeMutex.Unlock()
		tmp.Lock()
		for _, cb := range tmp.callbacks {
			// Need new variable cause callback will be run in go routine
			f := cb
			go f(eventData)
		}
		tmp.Unlock()
	}()
	return nil

}

func (r *Events) SubscribeForTopic(topic string, callback func(event []byte) error) (pubsub.CancelFunc, error) {
	r.subscribeMutex.Lock()
	defer r.subscribeMutex.Unlock()
	r.initSubscribeMapTopic(topic)
	el := r.subscribeMap[topic]
	i := el.lastEl
	el.lastEl++
	el.callbacks[i] = callback

	return func() {
		r.subscribeMutex.Lock()
		defer r.subscribeMutex.Unlock()
		el.Lock()
		delete(el.callbacks, i)
		if len(el.callbacks) == 0 {
			atomic.AddInt32(&el.done, 1)
			delete(r.subscribeMap, topic)
		}
		el.Unlock()
	}, nil
}

func (r *Events) UnsubscribeFromTopic(topic string) {
	r.subscribeMutex.Lock()
	defer r.subscribeMutex.Unlock()

	el := r.subscribeMap[topic]
	el.Lock()
	//TODO: handle error?
	atomic.AddInt32(&el.done, 1)
	delete(r.subscribeMap, topic)
	el.Unlock()
}

type ResponseMsg struct {
	Response []byte
	Err      string
}

type RequestMsg struct {
	EventData []byte
	Context   []byte
}

func New() (inst *Events) {
	return &Events{
		subscribeMap: map[string]*data{},
	}
}

func (r *Events) Close() {
	// todo
}

func (r *Events) initSubscribeMap(namespace string, eventName string) {
	r.initSubscribeMapTopic(getTopic(namespace, eventName))
}

func (r *Events) initSubscribeMapTopic(topic string) {
	if _, ok := r.subscribeMap[topic]; !ok {
		r.subscribeMap[topic] = &data{callbacks: map[int64]func(event []byte) error{}}
	}
}

func (r *Events) Publish(namespace string, eventName string, eventData []byte) (err error) {
	return r.PublishToTopic(getTopic(namespace, eventName), eventData)
}

func getTopic(namespace string, eventName string) string {
	return namespace + "." + eventName
}

func (r *Events) Unsubscribe(namespace string, eventName string) {
	r.UnsubscribeFromTopic(getTopic(namespace, eventName))
}

func (r *Events) Subscribe(namespace string, eventName string, callback func(event []byte) error) (cancel pubsub.CancelFunc, err error) {
	return r.SubscribeForTopic(getTopic(namespace, eventName), callback)
}
