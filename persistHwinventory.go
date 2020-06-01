package main

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	pcc "github.com/platinasystems/pcc-blackbox/lib"
	"github.com/platinasystems/test"
)

var PxeBootSelectedNodeId uint64
var hwInventory []pcc.HardwareInventory

func testHardwareInventory(t *testing.T) {
	t.Run("pxebootNode", pxebootNode)
	t.Run("checkNodeAdd", checkNodeAdd)
	t.Run("checkHardwareInventory", checkHardWareInventory)
	t.Run("checkStorage", checkStorage)
	t.Run("powerCycleNode", powerCycleNode)
}

func pxebootNode(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}
	var (
		err     error
		pxeboot []byte
	)
	if len(Env.Servers) != 0 {
		pxeboot, err = exec.Command("/bin/bash", "-c", "for cmd in 'chassis bootdev pxe' 'chassis power cycle'; do ipmitool -I lanplus -H "+Env.Servers[0].BMCIp+" -U ADMIN -P ADMIN $cmd; done").Output()
		if err != nil {
			assert.Fatalf("%v\n%v\n", string(pxeboot), err)
			fmt.Printf("pxeboot failed %v\n%v\n", string(pxeboot), err)
			return
		}
	}
}

func checkNodeAdd(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	from := time.Now()
	err := verifyAddNode(from, "nodeAdd")
	if err != nil {
		fmt.Println("Node additon failed..ERROR:", err)
		assert.Fatalf("Node addition failed")
		return
	}
}

func checkHardWareInventory(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}
	var (
		flag bool
		err  error
	)

	hwInventory, err = pcc.Pcc.GetHardwareInventory()
	if err != nil {
		assert.Fatalf("GetHardwareInventory failed: %v\n", err)
		return
	}

	for _, server := range Env.Servers {
		for _, hw := range hwInventory {
			if server.BMCIp == hw.Bus.Bmc.Ipcfg.Ipaddress {
				PxeBootSelectedNodeId = hw.NodeID
				flag = true
				fmt.Printf("Hardware inventory with node id %v persisted succesfully", PxeBootSelectedNodeId)
				break
			}
		}
	}
	if !flag {
		assert.Fatalf("inventory for node with id %v not persisted in db", PxeBootSelectedNodeId)
		return
	}
}
func checkStorage(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	var (
		storage pcc.StorageChildrenTO
		err     error
	)

	storage, err = pcc.Pcc.GetStorageNode(PxeBootSelectedNodeId)
	if err != nil {
		assert.Fatalf("GetStorageNode failed: %v\n", err)
		return
	}

	if len(storage.Children) != 0 {
		fmt.Printf("inventory for node with id %v persisted in "+
			"storage inventory", PxeBootSelectedNodeId)
	} else {
		assert.Fatalf("inventory for node with id %v not persisted "+
			"in storage inventory", PxeBootSelectedNodeId)
		return
	}
}

func powerCycleNode(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}
	var (
		err        error
		powerCycle []byte
	)
	if len(Env.Servers) != 0 {
		powerCycle, err = exec.Command("/bin/bash", "-c", "ipmitool -I lanplus -H "+Env.Servers[0].BMCIp+" -U ADMIN -P ADMIN chassis power cycle").Output()
		if err != nil {
			assert.Fatalf("%v\n%v\n", string(powerCycle), err)
			fmt.Printf("power cycle failed %v\n%v\n", string(powerCycle), err)
			return
		}
	}

}

func verifyAddNode(from time.Time, action string) (err error) {
	done := make(chan status)
	breakLoop := make([]chan bool, 2)

	if action == "nodeAdd" {
		go func() {
			breakLoop[0] = make(chan bool)
			syncCheckGenericInstallation(0, PXEBOOT_TIMEOUT, PXEBOOT_NODE_ADD_NOTIFICATION, from, done, breakLoop[0])
		}()
		go func() {
			breakLoop[1] = make(chan bool)
			syncCheckGenericInstallation(0, PXEBOOT_TIMEOUT, PXEBOOT_NODE_ADD_FAILED_NOTIFICATION, from, done, breakLoop[1])
		}()
	}
	s := <-done
	go func() {
		for i := 0; i < 2; i++ {
			breakLoop[i] <- true
		}
	}()
	if !s.isError {
		if strings.Contains(s.msg, PXEBOOT_NODE_ADD_NOTIFICATION) {
			fmt.Println("Node is added succesfully..\n", s.msg)
		} else if strings.Contains(s.msg, PXEBOOT_NODE_ADD_FAILED_NOTIFICATION) {
			err = fmt.Errorf("%v", s.msg)
		}
	} else {
		err = fmt.Errorf("%v", s.msg)
	}
	return
}
