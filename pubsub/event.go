package pubsub

// Do not call any of this methods directly or from client.
// Methods must be wrapped inside service server/client

type Publisher interface {
	Publish(namespace string, eventName string, event []byte) error
}

type CancelFunc func()

type Subscriber interface {
	Subscribe(namespace string, eventName string, callback func(event []byte) error) (CancelFunc, error)
	Unsubscribe(namespace string, eventName string)
	//TODO: SubscribeWithGroup(namespace string, groupName string, eventName string, callback func(event []byte)) CancelFunc
}

type PubSub interface {
	Subscribe(namespace string, eventName string, callback func(event []byte) error) (CancelFunc, error)
	Unsubscribe(namespace string, eventName string)
	Publish(namespace string, eventName string, event []byte) error
}
