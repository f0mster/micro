package redis

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/f0mster/micro/pubsub"
)
import "github.com/mediocregopher/radix/v3"

type data struct {
	lastEl    int64
	callbacks map[int64]func(ctx, event []byte) error
	done      int32
	msgChan   chan radix.PubSubMessage
}
type Event struct {
	pool           *radix.Pool
	pubsub         radix.PubSubConn
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

func New(network, addr string) (inst *Event, err error) {
	inst = &Event{
		subscribeMap: map[string]map[string]*data{},
	}
	inst.pubsub, err = radix.PersistentPubSubWithOpts(network, addr)
	if err != nil {
		return nil, fmt.Errorf("radix pubsub create error: %w", err)
	}
	inst.pool, err = radix.NewPool(network, addr, 8)
	if err != nil {
		return nil, fmt.Errorf("radix pool create error: %w", err)
	}
	return
}

func (r *Event) Close() {
	r.pool.Close()
	r.pubsub.Close()
}

func (r *Event) Publish(namespace string, eventName string, ctx []byte, eventData []byte) (err error) {
	req := RequestMsg{
		EventData: eventData,
		Context:   ctx,
	}
	sreq, err := json.Marshal(req)
	if err != nil {
		return
	}
	err = r.pool.Do(radix.Cmd(nil, "PUBLISH", namespace+":"+eventName, string(sreq)))
	//	fmt.Println("sent to", namespace+":"+eventName)
	if err != nil {
		return
	}
	return
}

func (r *Event) Unsubscribe(namespace string, eventName string) {
	r.subscribeMutex.Lock()
	el := r.subscribeMap[namespace][eventName]
	//TODO: handle error?
	r.pubsub.Unsubscribe(el.msgChan, namespace+":"+eventName)
	atomic.AddInt32(&el.done, 1)
	close(el.msgChan)
	delete(r.subscribeMap[namespace], eventName)
	r.subscribeMutex.Unlock()
}

func (r *Event) Subscribe(namespace string, eventName string, callback func(ctx, event []byte) error) (cancel pubsub.CancelFunc, err error) {
	r.subscribeMutex.Lock()
	defer r.subscribeMutex.Unlock()
	if _, ok := r.subscribeMap[namespace]; !ok {
		r.subscribeMap[namespace] = map[string]*data{}
	}
	if _, ok := r.subscribeMap[namespace][eventName]; !ok {
		r.subscribeMap[namespace][eventName] = &data{
			callbacks: map[int64]func(ctx []byte, event []byte) error{},
			msgChan:   make(chan radix.PubSubMessage, 10000),
		}
	}
	el := r.subscribeMap[namespace][eventName]
	i := el.lastEl
	el.lastEl++

	if len(el.callbacks) == 0 {
		err = r.pubsub.Subscribe(el.msgChan, namespace+":"+eventName)
		if err != nil {
			return
		}

		go func() {
			for atomic.LoadInt32(&el.done) == 0 {
				evt := <-el.msgChan
				req := RequestMsg{}
				if len(evt.Message) == 0 {
					continue
				}
				err = json.Unmarshal(evt.Message, &req)
				if err != nil {
					fmt.Println(err)
				}
				r.subscribeMutex.Lock()
				for i := range el.callbacks {
					go el.callbacks[i](req.Context, req.EventData)
				}
				r.subscribeMutex.Unlock()
			}
		}()
	}
	el.callbacks[i] = callback

	return func() {
		r.subscribeMutex.Lock()
		defer r.subscribeMutex.Unlock()
		delete(el.callbacks, i)
		if len(el.callbacks) == 0 {
			atomic.AddInt32(&el.done, 1)
			_ = r.pubsub.Unsubscribe(el.msgChan, namespace+":"+eventName)
			if _, ok := r.subscribeMap[namespace][eventName]; ok {
				delete(r.subscribeMap[namespace], eventName)
				close(el.msgChan)
			}
		}
	}, nil
}
