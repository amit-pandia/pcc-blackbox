package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	pcc "github.com/platinasystems/pcc-blackbox/lib"
	"github.com/platinasystems/test"
)

var nodeIntfMap = make(map[uint64][]int64)

func configServerInterfaces(t *testing.T) {
	mayRun(t, "interfaces", func(t *testing.T) {
		mayRun(t, "configNetworkInterfaces", configNetworkInterfaces)
		mayRun(t, "verifyNetworkInterfaces", verifyNetworkInterfaces)
		mayRun(t, "verifyNetworkConfig", verifyNetworkConfig)
		mayRun(t, "verifyNetworkUp", verifyNetworkUp)
	})
}

// FIXME lump the functions in an single one
func configNetworkInterfaces(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	configureNode := func(node node, sever bool) {
		var (
			err       error
			ifaces    []*pcc.InterfaceDetail
			nodeIntfs []int64
		)

		id := NodebyHostIP[node.HostIp]
		if ifaces, err = pcc.Pcc.GetIfacesByNodeId(id); err != nil {
			assert.Fatalf("Error retrieving node %s id[%d] interfaces\n %v", node.HostIp, id, err)
			return
		}
	l2:
		for j := range node.NetInterfaces { // skip the check of the interfaces are not declared in Env
			iface := node.NetInterfaces[j]
			for _, intf := range ifaces {
				if iface.Name == intf.Interface.Name || iface.MacAddr == intf.Interface.MacAddress {
					nodeIntfs = append(nodeIntfs, intf.Interface.Id)
					continue l2
				}
			}

			fmt.Printf("SKIP interface %v for node %d\n", iface, id)
		}
		nodeIntfMap[id] = nodeIntfs
		configNodeInterfaces(t, sever, id, node.HostIp, node.NetInterfaces, ifaces)
	}

	for i := range Env.Invaders {
		configureNode(Env.Invaders[i].node, false)
	}
	for i := range Env.Servers {
		configureNode(Env.Servers[i].node, true)
	}
}

func prepIfaceRequest(nodeId uint64, iface *pcc.InterfaceDetail, configIface netInterface) (ifaceRequest pcc.InterfaceRequest) {

	ifaceRequest.NodeId = nodeId
	ifaceRequest.InterfaceId = iface.Interface.Id
	ifaceRequest.Name = iface.Interface.Name
	ifaceRequest.Ipv4Addresses = configIface.Cidrs
	ifaceRequest.MacAddress = configIface.MacAddr
	ifaceRequest.ManagedByPcc = configIface.ManagedByPcc
	ifaceRequest.Gateway = configIface.Gateway
	ifaceRequest.Autoneg = configIface.Autoneg
	if ifaceRequest.Autoneg == "off" {
		ifaceRequest.Speed =
			json.Number(configIface.Speed)
	} else {
		ifaceRequest.Speed = ""
	}
	ifaceRequest.Mtu = json.Number(configIface.Mtu)
	ifaceRequest.AdminStatus = pcc.INTERFACE_STATUS_UP
	if configIface.IsManagement {
		ifaceRequest.IsManagement = "true"
	} else {
		ifaceRequest.IsManagement = "false"
	}
	return
}

func configNodeInterfaces(t *testing.T, skipManagement bool, nodeId uint64, HostIp string,
	serverInterfaces []netInterface, ifaces []*pcc.InterfaceDetail) {

	assert := test.Assert{t}
	var (
		iface        *pcc.InterfaceDetail
		ifaceRequest pcc.InterfaceRequest
		err          error
	)

	for j := 0; j < len(serverInterfaces); j++ {
		mac := serverInterfaces[j].MacAddr
		if serverInterfaces[j].IsManagement && !skipManagement {
			continue // don't mess with management
		}
		iface, err = pcc.Pcc.GetIfaceByMacAddress(mac, ifaces)
		if err != nil {
			assert.Fatalf("Error in retrieving interface having "+
				"MacAddress: %v for node %v id[%v]",
				mac, HostIp, nodeId)
			return
		}

		ifaceRequest = prepIfaceRequest(nodeId, iface,
			serverInterfaces[j])

		fmt.Printf("Configuring node %v interface %v %v\n", nodeId,
			iface.Interface.Name, ifaceRequest)

		if err := pcc.Pcc.SetIface(ifaceRequest); err != nil {
			assert.Fatalf("Error setting interface %v for node "+
				"%v id[%v]: %v\n", ifaceRequest, HostIp,
				nodeId, err)
			return
		}
	}

	fmt.Printf("Apply interface changes for node %d\n", nodeId)
	if err = pcc.Pcc.ApplyIface(nodeId); err != nil {
		assert.Fatalf("Interface apply failed: %v\n", err)
		return
	}
}

func validateIfaceConfig(intfReq pcc.InterfaceRequest) (err error) {

	iface, err := pcc.Pcc.GetIfaceById(intfReq.NodeId, intfReq.InterfaceId)
	if err != nil {
		return
	}

	err = fmt.Errorf("config mismatch for %v", intfReq.Name)

	fmt.Printf("  Validating config for %v\n", intfReq.Name)

	// chop off ",<metric>"
	rGateway := strings.Split(intfReq.Gateway, ",")
	cGateway := strings.Split(iface.Interface.Gateway, ",")
	if rGateway[0] != cGateway[0] {
		fmt.Printf("    gateway mismatch %v %v\n",
			intfReq.Gateway, iface.Interface.Gateway)
		return
	}
	if iface.Interface.Autoneg {
		if intfReq.Autoneg != "on" {
			fmt.Printf("    autoneg mismatch %v %v\n",
				intfReq.Autoneg, iface.Interface.Autoneg)
			return
		}
	} else {
		if intfReq.Autoneg != "off" {
			fmt.Printf("    autoneg mismatch %v %v\n",
				intfReq.Autoneg, iface.Interface.Autoneg)
			return
		}
	}
	if intfReq.Autoneg == "off" {
		if intfReq.Speed != json.Number(iface.Interface.Speed) {
			fmt.Printf("    speed mismatch %v %v\n",
				intfReq.Speed, iface.Interface.Speed)
			return
		}
	}
	mtu := fmt.Sprintf("%v", iface.Interface.Mtu)
	if intfReq.Mtu != json.Number(mtu) {
		fmt.Printf("    mtu mismatch %v %v\n", intfReq.Mtu, mtu)
		return
	}
	if intfReq.AdminStatus != iface.Interface.AdminStatus {
		fmt.Printf("    adminStatus mismatch %v %v\n",
			intfReq.AdminStatus, iface.Interface.AdminStatus)
		return
	}
	if iface.Interface.IsManagement {
		if intfReq.IsManagement != "true" {
			fmt.Printf("    IsManagement mismatch %v %v\n",
				intfReq.IsManagement,
				iface.Interface.IsManagement)
			return
		}
	} else {
		if intfReq.IsManagement != "false" {
			fmt.Printf("    IsManagement mismatch %v %v\n",
				intfReq.IsManagement,
				iface.Interface.IsManagement)
			return
		}
	}
	if intfReq.ManagedByPcc != iface.Interface.ManagedByPcc {
		fmt.Printf("    ManagedByPcc mismatch %v %v\n",
			intfReq.ManagedByPcc, iface.Interface.ManagedByPcc)
		return
	}

	var desireIpMap = make(map[string]int, len(intfReq.Ipv4Addresses))
	for _, addr := range intfReq.Ipv4Addresses {
		desireIpMap[addr] = 1
	}
	for _, addr := range iface.Interface.Ipv4Addresses {
		if _, ok := desireIpMap[addr]; ok {
			delete(desireIpMap, addr)
		}
	}
	if len(desireIpMap) != 0 {
		fmt.Printf("    Ipv4 mismatch ")
		for k, _ := range desireIpMap {
			fmt.Printf("  %v\n", k)
		}
		return
	}

	var desire6IpMap = make(map[string]int,
		len(iface.Interface.Ipv6Addresses))

	for _, addr := range iface.Interface.Ipv6Addresses {
		if strings.HasPrefix(strings.ToLower(addr), "fe80") {
			continue // skip link local
		}
		desire6IpMap[addr] = 1
	}

	for _, addr := range intfReq.Ipv6Addresses {
		if _, ok := desire6IpMap[addr]; ok {
			delete(desire6IpMap, addr)
		}
	}
	if len(desire6IpMap) != 0 {
		fmt.Printf("    Ipv6 mismatch ")
		for k, _ := range desire6IpMap {
			fmt.Printf("  %v\n", k)
		}
		return
	}

	err = nil
	return
}

func serverConfigLoop(id uint64, serverIntfs []netInterface) (done bool, err error) {
	var ifaces []*pcc.InterfaceDetail

	done = false
	if ifaces, err = pcc.Pcc.GetIfacesByNodeId(id); err != nil {
		return
	}

	fmt.Printf("Validating config on server %v\n", id)

	var intfsToCheck = make(map[string]netInterface, len(serverIntfs))
	for _, intf := range serverIntfs {
		if intf.IsManagement {
			continue // don't mess with management
		}
		intfsToCheck[intf.MacAddr] = intf
	}

	for _, intf := range serverIntfs {
		var (
			intfConfig netInterface
			mac        string
			ok         bool
			iface      *pcc.InterfaceDetail
		)

		mac = intf.MacAddr
		if intfConfig, ok = intfsToCheck[mac]; !ok {
			continue
		}

		iface, err = pcc.Pcc.GetIfaceByMacAddress(mac, ifaces)
		if err != nil {
			return
		}

		ifaceRequest := prepIfaceRequest(id, iface, intfConfig)
		err = validateIfaceConfig(ifaceRequest)
		if err == nil {
			delete(intfsToCheck, mac)
		} else {
			pcc.Pcc.SetIfaceApply(ifaceRequest)
		}
	}
	err = nil
	if len(intfsToCheck) == 0 {
		done = true
		return
	}
	return
}

func verifyNetworkConfig(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	var serverMap = make(map[uint64][]netInterface)
	for _, i := range Env.Servers {
		id := NodebyHostIP[i.HostIp]
		serverMap[id] = i.NetInterfaces
	}
	for _, i := range Env.Invaders {
		id := NodebyHostIP[i.HostIp]
		serverMap[id] = i.NetInterfaces
	}

	allDone := false
	loop := 0
	loopLimit := 50
	for !allDone {
		loop++
		fmt.Printf("Interface config validation, loop %v\n", loop)
		for _, node := range Nodes {
			var (
				done bool
				err  error
			)
			ifaces, ok := serverMap[node.Id]
			if !ok {
				continue
			}
			done, err = serverConfigLoop(node.Id, ifaces)
			if err != nil {
				assert.Fatalf("Failed serverConfigLoop: %v\n",
					err)
				return
			}
			if done {
				delete(serverMap, node.Id)
			}
		}
		if len(serverMap) == 0 {
			allDone = true
		} else {
			time.Sleep(10 * time.Second)
		}
		if loop >= loopLimit {
			assert.Fatal("Timed out verifying intferface config\n")
			return
		}
	}

}

func verifyNetworkInterfaces(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}
	var nodesToCheck = make(map[uint64]uint64, len(nodeIntfMap))

	for id := range nodeIntfMap {
		nodesToCheck[id] = 1
	}

	timeout := time.After(10 * time.Minute)
	tick := time.Tick(5 * time.Second)
	for {
		select {
		case <-timeout:
			for id, _ := range nodesToCheck {
				intfs, found := nodeIntfMap[id]
				if !found {
					assert.Fatalf("map lookup failed %v\n",
						id)
					return
				}
				for _, i := range intfs {
					intf, _ := pcc.Pcc.GetIfaceById(id, i)
					if intf.Interface.IsManagement {
						continue
					}
					state := intf.Interface.IntfState
					if state != pcc.Ready {
						fmt.Printf("failed to update"+
							" %v %v %v\n", id,
							intf.Interface.Name,
							state)
					}
				}
			}
			assert.Fatalf("time out updating interfaces\n")
			return
		case <-tick:
			for id, intfs := range nodeIntfMap {
				if _, found := nodesToCheck[id]; !found {
					continue
				}
				intf_count := len(intfs)
				intf_up := 0
				for _, i := range intfs {
					intf, err := pcc.Pcc.GetIfaceById(id, i)
					if err != nil {
						return
					}
					if intf.Interface.IsManagement {
						intf_count--
						continue
					}

					state := intf.Interface.IntfState
					switch state {
					case pcc.Ready:
						intf_up++
					case pcc.Queued:
					case pcc.Updating:
					case pcc.Unknown:
					case pcc.Offline:
					default:
						assert.Fatalf("unexpected "+
							"IntfState %v\n",
							state)
						return
					}
				}
				if intf_count == intf_up {
					fmt.Printf("Node %v interfaces "+
						"updated\n", id)
					delete(nodesToCheck, id)
				}
			}
			if len(nodesToCheck) == 0 {
				return
			}
		}
	}
}

func verifyNetworkUp(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}
	var nodesToCheck = make(map[uint64]uint64, len(nodeIntfMap))

	for id := range nodeIntfMap {
		nodesToCheck[id] = 1
	}

	timeout := time.After(5 * time.Minute)
	tick := time.Tick(5 * time.Second)
	for {
		select {
		case <-timeout:
			for id, _ := range nodesToCheck {
				intfs, found := nodeIntfMap[id]
				if !found {
					assert.Fatalf("map lookup failed %v\n",
						id)
					return
				}
				for _, i := range intfs {
					intf, err := pcc.Pcc.GetIfaceById(id, i)
					if err != nil {
						assert.Fatalf("getIfaceById: %v",
							err)
					}
					if !intf.Interface.ManagedByPcc {
						continue
					}
					ifName := intf.Interface.Name
					carrier := intf.Interface.CarrierStatus
					admin := intf.Interface.AdminStatus
					fmt.Printf("  %v %v admin %v carrier "+
						"%v\n",
						id, ifName, admin, carrier)
				}
			}
			assert.Fatalf("time out updating interfaces\n")
			return
		case <-tick:
			for id, intfs := range nodeIntfMap {
				if _, found := nodesToCheck[id]; !found {
					continue
				}
				intf_count := len(intfs)
				intf_up := 0
				admin_down := 0
				for _, i := range intfs {
					intf, _ := pcc.Pcc.GetIfaceById(id, i)
					if !intf.Interface.ManagedByPcc {
						intf_count--
						continue
					}
					status := intf.Interface.AdminStatus
					if status == pcc.INTERFACE_STATUS_DOWN {
						admin_down++
					}
				}

				for _, i := range intfs {
					intf, _ := pcc.Pcc.GetIfaceById(id, i)
					if !intf.Interface.ManagedByPcc {
						continue
					}
					ifName := intf.Interface.Name
					carrier := intf.Interface.CarrierStatus
					admin := intf.Interface.AdminStatus
					if admin == pcc.INTERFACE_STATUS_DOWN {
						fmt.Printf("  %v %v admin "+
							"down\n", id, ifName)
					} else {
						fmt.Printf("  %v %v carrier "+
							"%v\n", id, ifName,
							carrier)
						if carrier == pcc.INTERFACE_STATUS_UP {
							intf_up++
						}
					}

				}
				if intf_up+admin_down == intf_count {
					fmt.Printf("Node %v interfaces "+
						"all UP\n", id)
					delete(nodesToCheck, id)
				}
			}
			if len(nodesToCheck) == 0 {
				return
			}
		}
	}
}
