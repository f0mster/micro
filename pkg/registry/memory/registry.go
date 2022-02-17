package memory

import (
	"sync"

	"github.com/f0mster/micro/pkg/registry"
)

type data struct {
	instances                          map[registry.InstanceId]bool
	watchRegisteredCallbacks           map[int64]func()
	watchUnregisteredCallbacks         map[int64]func()
	WatchInstanceRegisteredCallbacks   map[int64]func(instanceId registry.InstanceId)
	WatchInstanceUnregisteredCallbacks map[int64]func(instanceId registry.InstanceId)
}

type memRegistry struct {
	reg      map[string]*data
	regMutex sync.RWMutex
}

func New() *memRegistry {
	return &memRegistry{
		reg: map[string]*data{},
	}
}

func (m *memRegistry) Register(namespace string, instanceId registry.InstanceId) {
	m.regMutex.Lock()
	defer m.regMutex.Unlock()
	m.initNamespace(namespace)
	rns := m.reg[namespace]
	m.reg[namespace].instances[registry.InstanceId(instanceId)] = true
	for _, v := range rns.watchRegisteredCallbacks {
		go v()
	}
	for _, v := range rns.WatchInstanceRegisteredCallbacks {
		go v(instanceId)
	}
}

func (m *memRegistry) Unregister(namespace string, instanceId registry.InstanceId) {
	m.regMutex.Lock()
	defer m.regMutex.Unlock()
	m.initNamespace(namespace)
	rns := m.reg[namespace]
	lenOld := len(m.reg[namespace].instances)
	delete(m.reg[namespace].instances, registry.InstanceId(instanceId))

	for _, v := range rns.WatchInstanceUnregisteredCallbacks {
		go v(instanceId)
	}

	if lenOld == 1 && len(m.reg[namespace].instances) == 0 {
		for _, v := range rns.watchUnregisteredCallbacks {
			go v()
		}
	}

}

func (m *memRegistry) Instances(namespace string) map[registry.InstanceId]bool {
	m.regMutex.RLock()
	defer m.regMutex.RUnlock()
	m.initNamespace(namespace)
	resp := map[registry.InstanceId]bool{}
	for k, v := range m.reg[namespace].instances {
		resp[k] = v
	}
	return resp
}

func (m *memRegistry) WatchUnregistered(namespace string, onchange func()) registry.CancelFunc {
	m.regMutex.Lock()
	defer m.regMutex.Unlock()
	m.initNamespace(namespace)
	data := m.reg[namespace].watchUnregisteredCallbacks
	i := int64(0)
	for {
		if _, ok := data[i]; !ok {
			break
		}
		i++
	}
	data[i] = onchange
	deleted := false
	return func() {
		if deleted {
			return
		}
		m.regMutex.Lock()
		defer m.regMutex.Unlock()
		delete(data, i)
		deleted = true
	}
}

func (m *memRegistry) WatchRegistered(namespace string, onchange func()) registry.CancelFunc {
	m.regMutex.Lock()
	defer m.regMutex.Unlock()
	m.initNamespace(namespace)
	data := m.reg[namespace].watchRegisteredCallbacks
	i := int64(0)
	for {
		if _, ok := data[i]; !ok {
			break
		}
		i++
	}

	data[i] = onchange
	deleted := false
	return func() {
		if deleted {
			return
		}
		m.regMutex.Lock()
		defer m.regMutex.Unlock()
		delete(data, i)
		deleted = true
	}
}

func (m *memRegistry) WatchInstanceUnregistered(namespace string, onchange func(instanceId registry.InstanceId)) registry.CancelFunc {
	m.regMutex.Lock()
	defer m.regMutex.Unlock()
	m.initNamespace(namespace)
	data := m.reg[namespace].WatchInstanceUnregisteredCallbacks
	i := int64(0)
	for {
		if _, ok := data[i]; !ok {
			break
		}
		i++
	}
	data[i] = onchange
	deleted := false
	return func() {
		if deleted {
			return
		}
		m.regMutex.Lock()
		defer m.regMutex.Unlock()
		delete(data, i)
		deleted = true
	}
}

func (m *memRegistry) WatchInstanceRegistered(namespace string, onchange func(instanceId registry.InstanceId)) registry.CancelFunc {
	m.regMutex.Lock()
	m.initNamespace(namespace)
	data := m.reg[namespace].WatchInstanceRegisteredCallbacks
	i := int64(0)
	for {
		if _, ok := data[i]; !ok {
			break
		}
		i++
	}
	insts := map[registry.InstanceId]bool{}
	for inst := range m.reg[namespace].instances {
		insts[inst] = true
	}
	m.regMutex.Unlock()
	for inst, addr := range insts {
		insts[inst] = addr
		go onchange(inst)
	}
	data[i] = onchange
	deleted := false
	return func() {
		if deleted {
			return
		}
		m.regMutex.Lock()
		defer m.regMutex.Unlock()
		delete(data, i)
		deleted = true
	}
}

func (m *memRegistry) initNamespace(namespace string) {
	if _, ok := m.reg[namespace]; !ok {
		m.reg[namespace] = &data{
			instances:                          map[registry.InstanceId]bool{},
			watchRegisteredCallbacks:           map[int64]func(){},
			watchUnregisteredCallbacks:         map[int64]func(){},
			WatchInstanceRegisteredCallbacks:   map[int64]func(instanceId registry.InstanceId){},
			WatchInstanceUnregisteredCallbacks: map[int64]func(instanceId registry.InstanceId){},
		}
	}
}
