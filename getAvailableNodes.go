package main

import (
	pcc "github.com/platinasystems/pcc-blackbox/lib"
	"github.com/platinasystems/test"
	"testing"
)

func getNodes(t *testing.T) {
	t.Run("getAvailableNodes", getAvailableNodes)
}

func getAvailableNodes(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}
	if nodes, err := pcc.Pcc.GetNodes(); err == nil {
		for i := 0; i < len(*nodes); i++ {
			node := (*nodes)[i]
			id := node.Id
			Nodes[id] = &node
			NodebyHostIP[node.Host] = id
			Env.setNodeId(node.Host, id)
		}
	} else {
		assert.Fatalf("Error getting nodes: %v\n", err)
	}
}
