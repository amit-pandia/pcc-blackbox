package main

import (
	"fmt"
	pcc "github.com/platinasystems/pcc-blackbox/lib"
	"strings"
	"sync"
	"testing"
	"time"
)

// Add UNREACHABLE NODE
func addUnreachableNode(t *testing.T) {
	address := Env.Availability.FakeAddress
	if address == "" {
		address = "IamNotWorking"
	}
	if err := checkNodeConnectionStatus("unreachable", address); err != nil {
		t.Fatal(err)
	}
}

// Add INACCESSIBLE NODE
func addInaccessibleNode(t *testing.T) {
	address := Env.Availability.Inaccessible
	if address == "" {
		address = Env.PccIp
	}
	if err := checkNodeConnectionStatus("inaccessible", address); err != nil {
		t.Fatal(err)
	}
}

// add a node and wait for the connection status
func checkNodeConnectionStatus(status string, host string) (err error) {
	fmt.Println(fmt.Sprintf("\nAVAILABILITY: add %s node", status))
	var node *pcc.NodeWithKubernetes

	node = &pcc.NodeWithKubernetes{}
	node.Host = host

	if err = pcc.Pcc.AddNode(node); err == nil {
		defer func() { // delete the node
			pcc.Pcc.DeleteNode(node.Id)
		}()

		fmt.Printf("node [%s] added with id [%d]. Waiting for connection status [%s]\n", node.Host, node.Id, status)
		for i := 1; i <= 20; i++ { // wait for the status
			time.Sleep(time.Second * time.Duration(10))
			if node, err = pcc.Pcc.GetNode(node.Id); err == nil && node.NodeAvailabilityStatus != nil {
				connectionStatus := node.NodeAvailabilityStatus.ConnectionStatus
				fmt.Printf("Connection status for node %s is %s\n", host, connectionStatus)
				if strings.Compare(strings.ToLower(connectionStatus), status) == 0 {
					return
				}
			}
		}
	}

	if err == nil {
		err = fmt.Errorf("unable to get the fake status for the node %s", node.Host)
	}

	return
}

// Delete  both agent and collector and wait for the restore
func checkAgentAndCollectorRestore(t *testing.T) {
	fmt.Println("\nAVAILABILITY: checking the agent/collector restore")
	var wg sync.WaitGroup

	if nodes, err := pcc.Pcc.GetNodes(); err == nil {
		if len(*nodes) > 0 {
			node := (*nodes)[0] // get the first node
			wg.Add(2)

			f := func(service string, path string) {
				defer wg.Done()
				if err = checkRestore(service, path, &node); err != nil {
					t.Fatal(err)
				}
			}

			go f(AGENT_NOTIFICATION, "pccagent")            // Delete the agent and wait for the restore
			go f(COLLECTOR_NOTIFICATION, "systemCollector") // Delete the collector and wait for the restore

			wg.Wait()
		} else {
			t.Fatal("No nodes were found")
		}
	} else {
		t.Fatal(err)
	}
}

// Remove the service and wait for the restore
func checkRestore(service string, path string, node *pcc.NodeWithKubernetes) error {
	var ssh pcc.SSHHandler
	fmt.Println(fmt.Sprintf("Stopping and removing the service %s from node %d %s %s ", service, node.Id, node.Name, node.Host))
	if _, stderr, err := ssh.Run(node.Host, fmt.Sprintf("sudo rm -f /opt/platina/pcc/bin/%s && ps auxww | grep %s | grep -v grep | grep -v ansible | awk '{print $2}' | xargs sudo kill -9", path, path)); err == nil {
		fmt.Println(fmt.Sprintf("The %s:%s was correctly killed and removed from node %d:%s", service, path, node.Id, node.Name))

		if check, _ := pcc.Pcc.WaitForInstallation(node.Id, 60*10, service, "", nil); check {
			fmt.Println(fmt.Sprintf("The PCC restored the [%s] on the node [%d:%s]", service, node.Id, node.Name))
			return nil
		} else {
			return fmt.Errorf("the PCC was not able to restore the [%s] on the node [%d:%s]", service, node.Id, node.Name)
		}
	} else {
		fmt.Println(fmt.Sprintf("Error deleting the service %s\n%s", service, stderr))
		return err
	}
}
