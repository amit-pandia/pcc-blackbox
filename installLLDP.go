package main

import (
	pcc "github.com/platinasystems/pcc-blackbox/lib"
	"github.com/platinasystems/test"
	"testing"
)

func updateNodes_installLLDP(t *testing.T) {
	t.Run("installLLDP", installLLDP)
}

// Install LLDP on all nodes
func installLLDP(t *testing.T) {
	test.SkipIfDryRun(t)
	if nodes, err := pcc.Pcc.GetNodeIds(); err == nil {
		installLLDPOnNodes(nodes)
	} else {
		panic(err)
	}
}

// Install LLDP on all invaders
func installLLDPOnInvaders(t *testing.T) {
	if nodes, err := pcc.Pcc.GetInvaderIds(); err == nil {
		if err = installLLDPOnNodes(nodes); err != nil {
			t.Fatal(err)
		}
	} else {
		t.Fatal(err)
	}
}

// Install LLDP on nodes
func installLLDPOnNodes(nodes []uint64) (err error) {
	var roleId uint64
	if roleId, err = pcc.Pcc.FindRoleId(pcc.ROLE_LLDP); err == nil {
		err = setRolesToNodesAndCheck([]uint64{roleId}, pcc.ROLE_LLDPD, nodes, LLDP_TIMEOUT)
	}
	return
}
