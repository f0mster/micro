package tests

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/f0mster/micro/pkg/registry"
)

/*
  Внимание!

  Для запуска необходим docker.

  Адрес указывается через переменную окружения DOCKER_HOST.

*/

func Registry_Test(reg1 registry.Registry, reg2 registry.Registry, t *testing.T) {
	var stop registry.CancelFunc
	var stop1 registry.CancelFunc
	var stop2 registry.CancelFunc
	var stop3 registry.CancelFunc
	var stop4 registry.CancelFunc
	var stop5 registry.CancelFunc
	var stop6 registry.CancelFunc
	var stop7 registry.CancelFunc

	wg := sync.WaitGroup{}
	stop = reg1.WatchRegistered("ns1", func() {
		wg.Done()
	})
	stop7 = reg1.WatchRegistered("nsa", func() {
		t.Fatal("wrong behavior")
	})

	wg.Add(1)
	reg2.Register("ns1", "inst:1")
	wg.Wait()
	wg.Add(1)

	stop1 = reg1.WatchInstanceRegistered("ns1", func(instanceId registry.InstanceId) {
		require.Equal(t, registry.InstanceId("inst:1"), instanceId)
		wg.Done()
		stop1()
	})
	stop2 = reg1.WatchInstanceRegistered("nsa", func(instanceId registry.InstanceId) {
		t.Fatal("wrong behavior")
	})
	time.Sleep(1 * time.Second)
	stop3 = reg1.WatchUnregistered("ns2", func() {
		t.Fatal("wrong behavior")
	})
	stop4 = reg1.WatchInstanceUnregistered("ns2", func(instanceId registry.InstanceId) {
		t.Fatal("bad behavior")
	})

	wg.Add(2)
	stop5 = reg1.WatchUnregistered("ns1", func() {
		wg.Done()
	})
	stop6 = reg1.WatchInstanceUnregistered("ns1", func(instanceId registry.InstanceId) {
		require.Equal(t, registry.InstanceId("inst:1"), instanceId)
		wg.Done()
	})
	reg2.Unregister("ns1", "inst:1")
	wg.Wait()
	stop()
	stop1()
	stop2()
	stop3()
	stop4()
	stop5()
	stop6()
	stop7()
	reg2.Register("ns1", "inst:1")
	reg2.Unregister("ns1", "inst:1")

	time.Sleep(2 * time.Second)
	fmt.Println("all ok")
}
