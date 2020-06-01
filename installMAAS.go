package main

import (
	"fmt"
	pcc "github.com/platinasystems/pcc-blackbox/lib"
	"github.com/platinasystems/test"
	"testing"
	"time"
)

func updateNodes_installMAAS(t *testing.T) {
	t.Run("installMAAS", installMAAS)
}

func installMAAS(t *testing.T) {
	test.SkipIfDryRun(t)

	var (
		err    error
		maasId uint64
		lldpId uint64
	)

	if maasId, err = pcc.Pcc.FindRoleId(pcc.ROLE_MAAS); err == nil {
		if lldpId, err = pcc.Pcc.FindRoleId(pcc.ROLE_LLDP); err == nil {
			if nodes, err := pcc.Pcc.GetInvaderIds(); err == nil {
				if err = setRolesToNodesAndCheck([]uint64{lldpId, maasId}, "MAAS", nodes, MAAS_INSTALL_TIMEOUT); err != nil {
					t.Fatal(err)
				}
			} else {
				t.Fatal(err)
			}
		} else {
			t.Fatal(err)
		}
	} else {
		t.Fatal(err)
	}
}

// FIXME move the wait to PccClient
func setRolesToNodesAndCheck(roles []uint64, app string, nodes []uint64, timeout time.Duration) (err error) {
	var (
		installed    []uint64
		nodesToCheck []uint64
		check        bool
	)

	fmt.Printf("installing %s on nodes:%v\n", app, nodes)
	if installed, nodesToCheck, err = pcc.Pcc.AddRolesToNodes(nodes, roles); err == nil {
		if len(installed) > 0 {
			fmt.Printf("%s already installed on nodes %d", app, installed)
		}
		//Check APP installation
		for i := 0; i < len(nodesToCheck); i++ {
			id := nodesToCheck[i]
			fmt.Printf("Checking %s installation for node:%v\n", app, id)

			start := time.Now()
			if check, err = pcc.Pcc.WaitForInstallation(id, timeout, app, "", &start); err != nil {
				err = fmt.Errorf("failed checking %s on %v: %v", app, id, err)
				return
			} else if check {
				fmt.Printf("%s correctly installed on nodeId:%v\n", app, id)
			}
		}
	}

	return
}
