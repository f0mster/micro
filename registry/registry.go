package registry

// Do not call any of this methods directly or from client.
// Methods must be wrapped inside service server/client

type Registerer interface {
	Register(namespace string, instanceId InstanceId)
	Unregister(namespace string, instanceId InstanceId)
}

type InstanceId string
type RPCAddress string
type CancelFunc func()

type Watcher interface {
	// WatchUnregistered will call onchange func until CancelFunc called
	WatchUnregistered(namespace string, onchange func()) CancelFunc
	WatchRegistered(namespace string, onchange func()) CancelFunc
	WatchInstanceUnregistered(namespace string, onchange func(instanceId InstanceId)) CancelFunc
	WatchInstanceRegistered(namespace string, onchange func(instanceId InstanceId)) CancelFunc
	// Instances enumerates registered instances by namespace.
	// Returns map of InstanceId to bool.
	Instances(namespace string) map[InstanceId]bool
}

type Registry interface {
	Registerer
	Watcher
}
