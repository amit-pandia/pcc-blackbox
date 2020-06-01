package main

import (
	"fmt"
	pcc "github.com/platinasystems/pcc-blackbox/lib"
	"github.com/platinasystems/test"
	"testing"
	"time"
)

func delAllNodes(t *testing.T) {
	t.Run("delNodes", delNodes)
	t.Run("validateDeleteNodes", validateDeleteNodes)
}

func delNodes(t *testing.T) {
	test.SkipIfDryRun(t)
	if _, err := pcc.Pcc.DeleteNodes(true); err != nil {
		t.Fatal(err)
	}
}

func validateDeleteNodes(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	// wait for node to be removed
	time.Sleep(5 * time.Second)
	start := time.Now()
	timeout := 60 * 8 * time.Second // FIXME move to config
	for true {
		if nodes, err := pcc.Pcc.GetNodes(); err == nil {
		l1:
			for k := range Nodes { // clear the map or remove deleted nodes
				for _, node := range *nodes {
					if node.Id == k {
						continue l1
					}
				}
				delete(Nodes, k)
			}
		} else {
			fmt.Printf("error getting nodes %v\n", err)
		}
		if len(Nodes) == 0 {
			fmt.Println("all nodes have been deleted")
			return
		} else if len(Nodes) > 0 {
			time.Sleep(5 * time.Second)
		}
		if time.Since(start) > timeout {
			fmt.Printf("delAllNodes timeout\n")
			for _, node := range Nodes {
				assert.Fatalf("node %v provisionStatus = %v was not "+
					"deleted\n", node.Name, node.ProvisionStatus)
			}

			return
		}
	}
}
