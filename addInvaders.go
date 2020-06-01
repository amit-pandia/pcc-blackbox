package main

import (
	"fmt"
	pcc "github.com/platinasystems/pcc-blackbox/lib"
	"github.com/platinasystems/test"
	"strings"
	"sync"
	"testing"
	"time"
)

func addClusterHeads(t *testing.T) {
	t.Run("addInvaders", addInvaders)
}

func addInvaders(t *testing.T) {
	var nodes []node
	for i := range Env.Invaders {
		nodes = append(nodes, Env.Invaders[i].node)
	}

	fmt.Printf("adding %d invaders\n", len(nodes))
	addNodesAndCheckStatus(t, nodes)
}

// add nodes fom list
func addNodesAndCheckStatus(t *testing.T, nodes []node) {
	var err error
	test.SkipIfDryRun(t)

	//Check Agent and collector installation function. FIXME add a channel for stopping on error
	waitInstallation := func(timeout time.Duration, app string, nodeId uint64, from *time.Time) {
		fmt.Printf("Checking %s installation for nodeId:%v from %s \n", app, nodeId, from.String())
		check, waitErr := pcc.Pcc.WaitForInstallation(nodeId, timeout, app, "", from)
		if waitErr != nil {
			fmt.Printf("\n%v\n", waitErr)
			err = waitErr
		}
		if check {
			fmt.Printf("%s correctly installed on nodeId:%v\n", app, nodeId)
		}
	}

	var wg sync.WaitGroup
	wg.Add(len(nodes))

	addNodeAndWait := func(n node, nodeNumbers int) { // add the node and wait for the services. FIXME add a channel for stopping on error
		defer wg.Done()
		var (
			node         pcc.NodeWithKubernetes
			routineError error
		)
		node.Host = n.HostIp
		node.Managed = new(bool)
		*node.Managed = true

		if Nodes[NodebyHostIP[node.Host]] == nil { // add the node
			fmt.Printf("adding the node %s\n", node.Host)
			if routineError = pcc.Pcc.AddNode(&node); routineError == nil {
				n.Id = node.Id
				node.Invader = true
				Nodes[node.Id] = &node
				NodebyHostIP[node.Host] = node.Id
				fmt.Printf("Add id %d to Nodes. Mapping hostIP %v to id %d\n", node.Id, node.Host, node.Id)

				eventsFrom := time.Now()
				waitInstallation(AGENT_TIMEOUT*time.Duration(nodeNumbers), AGENT_NOTIFICATION, node.Id, &eventsFrom)
				waitInstallation(COLLECTOR_TIMEOUT*time.Duration(nodeNumbers), COLLECTOR_NOTIFICATION, node.Id, &eventsFrom)
				start := time.Now()
				timeout := time.Duration(180*len(nodes)) * time.Second
				var (
					connection         string
					previousConnection string
					ignore             error
					status             string
				)
				for true {
					time.Sleep(10 * time.Second)
					if status, ignore = pcc.Pcc.GetProvisionStatus(node.Id); ignore == nil { // early check for add fail
						if strings.Contains(status, "Add node failed") {
							err = fmt.Errorf("%s for node %d", status, node.Id)
							t.Fatal(err)
						}
					}
					if connection, ignore = pcc.Pcc.GetNodeConnectionStatus(node.Id); ignore == nil {
						switch connection { // FIXME use models
						case "online":
							fmt.Printf("the node %d:%s is online\n", node.Id, node.Host)
							return
						case "", "NoRunningService": // wait for the next cycle
						default:
							err = fmt.Errorf(fmt.Sprintf("Unable to add the node %s", connection))
							t.Fatal(err)
						}
						if previousConnection != "" && previousConnection != connection {
							fmt.Printf("the node %d:%s connection status switched from %s to %s\n", node.Id, node.Host, previousConnection, connection)
						}

						previousConnection = connection
					} else {
						fmt.Printf("error getting the connection status for node %d:%s %v\n", node.Id, node.Host, err)
					}

					if time.Since(start) > timeout {
						err = fmt.Errorf("timeout for node addition %d", node.Id)
						t.Fatal(err)
					}

					if err != nil {
						return
					}
				}
			} else {
				err = fmt.Errorf("add node %s failed\n%v\n", node.Host, routineError)
				t.Fatal(err)
			}
		} else {
			fmt.Printf("the node %s was already added\n", node.Host)
		}
	}

	for i := range nodes { // add nodes in parallel
		go addNodeAndWait(nodes[i], len(nodes))
	}

	wg.Wait() // wait for all addition

	if err != nil {
		fmt.Println(fmt.Sprintf("Error adding nodes %v", err))
		t.Fatal(err)
	}

}
