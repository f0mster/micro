package memory

import (
	"sync"
	"sync/atomic"

	"github.com/f0mster/micro/pubsub"
)

type data struct {
	lastEl    int64
	callbacks map[int64]func(ctx, event []byte) error
	sync.Mutex
	done int32
}

type Events struct {
	subscribeMap   map[string]map[string]*data
	subscribeMutex sync.Mutex
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
		subscribeMap: map[string]map[string]*data{},
	}
}

func (r *Events) Close() {
	// todo
}

func (r *Events) initSubscribeMap(namespace string, eventName string) {
	if _, ok := r.subscribeMap[namespace]; !ok {
		r.subscribeMap[namespace] = map[string]*data{}
	}
	if _, ok := r.subscribeMap[namespace][eventName]; !ok {
		r.subscribeMap[namespace][eventName] = &data{callbacks: map[int64]func(ctx []byte, event []byte) error{}}
	}
}

func (r *Events) Publish(namespace string, eventName string, ctx []byte, eventData []byte) (err error) {
	go func() {
		r.subscribeMutex.Lock()
		r.initSubscribeMap(namespace, eventName)

		tmp := r.subscribeMap[namespace][eventName]
		r.subscribeMutex.Unlock()
		tmp.Lock()
		for _, cb := range tmp.callbacks {
			// Need new variable cause callback will be run in go routine
			f := cb
			go f(ctx, eventData)
		}
		tmp.Unlock()
	}()
	return
}

func (r *Events) Unsubscribe(namespace string, eventName string) {
	r.subscribeMutex.Lock()
	defer r.subscribeMutex.Unlock()

	el := r.subscribeMap[namespace][eventName]
	el.Lock()
	//TODO: handle error?
	atomic.AddInt32(&el.done, 1)
	delete(r.subscribeMap[namespace], eventName)
	el.Unlock()
}

func (r *Events) Subscribe(namespace string, eventName string, callback func(ctx, event []byte) error) (cancel pubsub.CancelFunc, err error) {
	r.subscribeMutex.Lock()
	defer r.subscribeMutex.Unlock()
	r.initSubscribeMap(namespace, eventName)
	el := r.subscribeMap[namespace][eventName]
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
			delete(r.subscribeMap[namespace], eventName)
		}
		el.Unlock()
	}, nil
}
