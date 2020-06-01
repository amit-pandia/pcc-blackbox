package main

import (
	"fmt"
	"github.com/go-test/deep"
	pcc "github.com/platinasystems/pcc-blackbox/lib"
	"strings"
	"testing"
	"time"
)

// Check if server is reaching the PCC through the tunnel
func checkInvaderTunnels(t *testing.T) {
	fmt.Println("\nTUNNEL: checking the tunnel addresses for the invaders")
	if nodes, err := pcc.Pcc.GetInvaders(); err == nil {

		for i := range *nodes {
			node := (*nodes)[i]
			nodeId := node.Id

			checkEnvironmentForNode(t, &node) // Check for the environment address

			if node, err := pcc.Pcc.GetNodeFromDB(nodeId); err == nil { // Check the DB
				address := node.TunnelServerAddress
				if len(strings.TrimSpace(address)) == 0 {
					t.Fatal(fmt.Sprintf("The tunnel address for the invader %d:%s is blank", nodeId, node.Name))
				} else {
					fmt.Println(fmt.Sprintf("The invader %d:%s is reaching the PCC through the tunnel %s", nodeId, node.Name, address))
				}
			} else {
				t.Fatal(err)
			}
		}
	} else {
		t.Fatal(err)
	}
}

// Check if server is reaching the PCC through the cluster-head tunnel
func checkServerTunnels(t *testing.T) {
	fmt.Println("\nTUNNEL: checking the tunnel addresses for the servers")
	if invaders, err := pcc.Pcc.GetInvadersFromDB(); err == nil {
		if nodes, err := pcc.Pcc.GetServers(); err == nil {
		loopServer:
			for i := range *nodes {
				node := (*nodes)[i]
				nodeId := node.Id
				address := checkEnvironmentForNode(t, &node) // Check for the environment address

				if node, err := pcc.Pcc.GetNodeFromDB(nodeId); err == nil { // Check the DB. The address should be blank
					address := node.TunnelServerAddress
					if len(strings.TrimSpace(address)) != 0 {
						t.Fatal(fmt.Sprintf("The tunnel address for the server %d:%s should be blank instead of %s", nodeId, node.Name, node.TunnelServerAddress))
					}
				} else {
					t.Fatal(err)
				}

				for j := range *invaders { // Check if the address is associated to one Invader
					invader := (*invaders)[j]
					if strings.Compare(address, invader.Host) == 0 {
						fmt.Println(fmt.Sprintf("The node %d:%s is reaching the pcc through the invader %d:%s:%s", nodeId, node.Name, invader.Id, invader.Name, invader.Host))
						continue loopServer
					}
				}

				t.Fatal(fmt.Sprintf("Unable to find the invader associated to the server %d:%s", nodeId, node.Name))
			}
		} else {
			t.Fatal(err)
		}
	} else {
		t.Fatal(err)
	}
}

// Check data received from the environment endpoint
func checkEnvironmentForNode(t *testing.T, node *pcc.NodeWithKubernetes) (address string) {
	fmt.Println(fmt.Sprintf("Tunnel: checking the tunnel address for the node %d:%s:%s", node.Id, node.Name, node.Host))

	if defaultEnv, err := pcc.Pcc.GetEnvironment(nil); err == nil {
		defaultAddress := defaultEnv["servicePublicHost"]
		nodeId := node.Id

		if env, err := pcc.Pcc.GetEnvironment(&nodeId); err == nil { // Check the environment
			hAddress := env["servicePublicHost"]
			if diff := deep.Equal(defaultAddress, hAddress); diff == nil {
				t.Fatal(fmt.Sprintf("The tunnel address should be different from the PCC address %v", defaultAddress))
			} else {
				fmt.Println(fmt.Sprintf("The node %d:%s is getting the address %v", nodeId, node.Name, hAddress))
				address = fmt.Sprintf("%v", hAddress)
			}
		} else {
			t.Fatal(err)
		}
	} else {
		t.Fatal(err)
	}

	return
}

// Check if invader is reaching the PCC through the tunnel
func checkTunnelConnection(t *testing.T) {
	fmt.Println("\nTUNNEL: checking invaders connection")
	if nodes, err := pcc.Pcc.GetInvadersFromDB(); err == nil {
		for i := range *nodes {
			node := (*nodes)[i]

			if err = tunnelPing(&node, true); err != nil {
				t.Fatal(err)
			}
		}
	} else {
		t.Fatal(err)
	}
}

func tunnelPing(node *pcc.NodeWithKubernetes, log bool) error {
	nodeId := node.Id
	var ssh pcc.SSHHandler
	if stdout, stderr, err := ssh.Run(node.Host, fmt.Sprintf("ping -c 3 %s", node.TunnelServerAddress)); err == nil {
		if log {
			fmt.Println(fmt.Sprintf("The node %d:%s is pinging the address %v\n%s", nodeId, node.Name, node.TunnelServerAddress, stdout))
		}
		return nil
	} else {
		if log {
			fmt.Println(fmt.Sprintf("Error pinging from the node %d:%s %v\n%s", nodeId, node.Name, err, stderr))
		}
		return err
	}
}

// Check if invader is running the iptables rules
func checkTunnelForwardingRules(t *testing.T) {
	fmt.Println("\nTUNNEL: checking forwarding rules")
	var ssh pcc.SSHHandler

	if nodes, err := pcc.Pcc.GetInvadersFromDB(); err == nil {
		for i := range *nodes {
			node := (*nodes)[i]
			nodeId := node.Id

			if stdout, stderr, err := ssh.Run(node.Host, "sudo iptables --list-rules PREROUTING -t nat"); err == nil {
				lines := strings.Split(stdout, `-A`)
			loopPort:
				for _, port := range []string{"8081", "9092", "9999"} {
					rule := fmt.Sprintf("PREROUTING -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s", port, node.TunnelServerAddress, port)
					for _, line := range lines {
						if strings.Compare(strings.TrimSpace(line), rule) == 0 {
							fmt.Println(fmt.Sprintf("Found the forward rule for port %s on invader %d:%s", port, nodeId, node.Name))
							continue loopPort
						}
					}

					t.Fatal(fmt.Sprintf("Unable to find the forward rule for port %s on invader %d:%s. \nRules are:%s", port, nodeId, node.Name, stdout))
				}
			} else {
				fmt.Println(fmt.Sprintf("Error getting iptables tule from the node %d:%s %v\n%s", nodeId, node.Name, err, stderr))
				t.Fatal(err)
			}
		}
	} else {
		t.Fatal(err)
	}
}

// Check if the PCC is able to restore the tunnel
func checkTunnelRestore(t *testing.T) {
	fmt.Println("\nTUNNEL: checking the restore for the invaders")
	var ssh pcc.SSHHandler

	if nodes, err := pcc.Pcc.GetInvadersFromDB(); err == nil {
		for i := range *nodes {
			node := (*nodes)[i]
			nodeId := node.Id

			tun := fmt.Sprintf("tun%d", node.Id)
			if _, _, err := ssh.Run(node.Host, fmt.Sprintf("sudo ip link delete tun%d", node.Id)); err == nil {
				fmt.Println(fmt.Sprintf("The node %d:%s tun device %s has been removed. Waiting for the restore", nodeId, node.Name, tun))

			restoreLoop:
				for i := 1; i <= 10; i++ {
					time.Sleep(time.Second * time.Duration(15))
					if err = tunnelPing(&node, false); err == nil {
						fmt.Println(fmt.Sprintf("The PCC restored the tunnel %s on the invader %d:%s", tun, nodeId, node.Name))
						break restoreLoop
					}
				}

				if err != nil {
					fmt.Println(fmt.Sprintf("The PCC was not able to restore the tunnel for the invader %d:%s", nodeId, node.Name))
					t.Fatal(err)
				}
			} else {
				t.Fatal(err)
			}
		}
	} else {
		t.Fatal(err)
	}
}
