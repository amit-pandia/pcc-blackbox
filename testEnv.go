package main

import pcc "github.com/platinasystems/pcc-blackbox/lib"

type testEnv struct {
	Env                   string
	PccIp                 string
	Invaders              []invader
	Servers               []server
	DockerStats           pcc.DockerStatsConfig
	AuthenticationProfile pcc.AuthenticationProfile
	PortusConfiguration   pcc.PortusConfiguration
	DBConfiguration       *pcc.DBConfiguration
	SshConfiguration      *pcc.SshConfiguration
	CephConfiguration     pcc.CephConfiguration
	NetworkConfiguration     pcc.NetworkConfiguration
	K8sAppConfiguration   pcc.K8sAppConfiguration
	Availability          Availability
}

type node struct {
	Id            uint64
	HostIp        string
	BMCIp         string
	BMCUser       string
	BMCUsers      []string
	BMCPass       string
	KeyAlias      []string
	NetInterfaces []netInterface
}

type netInterface struct {
	Name         string
	Cidrs        []string
	Gateway      string
	MacAddr      string
	IsManagement bool
	ManagedByPcc bool
	Speed        string
	Autoneg      string
	Fec          string
	Media        string
	Mtu          string
}

type invader struct {
	node
}

type server struct {
	node
}

type Availability struct {
	FakeAddress  string
	Inaccessible string
}

var exampleEnv = testEnv{
	PccIp:        "172.17.2.238",
	Availability: Availability{"ImNotWorking", "172.17.0.0"},
	Invaders: []invader{
		invader{
			node{
				HostIp:   "172.17.2.60",
				BMCIp:    "172.17.3.60",
				BMCUser:  "ADMIN",
				BMCUsers: []string{"ADMIN"},
				BMCPass:  "ADMIN",
				NetInterfaces: []netInterface{
					netInterface{
						Name:  "eth0",
						Cidrs: []string{"172.17.2.60/23"},
					},
				},
			},
		},
	},
}

func (te *testEnv) setNodeId(host string, id uint64) (found bool) {
	found = false

	for i := range te.Invaders {
		if te.Invaders[i].HostIp == host {
			te.Invaders[i].Id = id
			found = true
			break
		}
	}

	if !found {
		for i := range te.Servers {
			if te.Servers[i].HostIp == host {
				te.Servers[i].Id = id
				break
			}
		}
	}
	return
}

func (te *testEnv) GetNodeByHost(host string) *node {
	for i := range te.Invaders {
		if te.Invaders[i].HostIp == host {
			return &te.Invaders[i].node
		}
	}

	for i := range te.Servers {
		if te.Servers[i].HostIp == host {
			return &te.Servers[i].node
		}
	}
	return nil
}

func (te *testEnv) IsNodeAlreadyAdded(host string) bool {
	if n := te.GetNodeByHost(host); n != nil {
		return n.Id > 0
	}

	return false
}
