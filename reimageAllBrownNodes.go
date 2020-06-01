package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"
	pcc "github.com/platinasystems/pcc-blackbox/lib"
	"github.com/platinasystems/test"
)

func reimageAllBrownNodes(t *testing.T) {
	t.Run("updateBmcInfo", updateBmcInfo)
	t.Run("reimageAllBrown", reimageAllBrown)
}

func updateBmcInfo(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	for _, i := range Env.Servers {
		var (
			addReq pcc.NodeWithKubernetes
			err    error
			key    pcc.SecurityKey
		)
		if key, err = getFirstKey(); err == nil {
			keys := pq.Int64Array{int64(key.Id)}

			pBool := new(bool)
			*pBool = true

			addReq.Host = i.HostIp
			addReq.Id = NodebyHostIP[i.HostIp]
			addReq.Bmc = i.BMCIp
			addReq.BmcUser = i.BMCUser
			addReq.BmcUsers = i.BMCUsers
			addReq.BmcPassword = i.BMCPass
			addReq.AdminUser = "admin"
			addReq.SSHKeys = keys
			addReq.Managed = pBool
			addReq.Console = "ttyS1"

			if err = pcc.Pcc.UpdateNode(&addReq); err != nil {
				assert.Fatalf("Failed to update BMC info: %v\n", err)
				return
			}
		} else {
			assert.Fatalf("Failed to get the key %v\n", err)
			return
		}
	}
}

func reimageAllBrown(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	if key, err := getFirstKey(); err == nil {
		keys := []string{key.Alias}

		nodesList := make([]uint64, len(Env.Servers))
		for i, s := range Env.Servers {
			nodesList[i] = NodebyHostIP[s.HostIp]
		}

		var request pcc.MaasRequest
		request.Nodes = nodesList
		request.Image = "centos76"
		request.Locale = "en-US"
		request.Timezone = "PDT"
		request.AdminUser = "admin"
		request.SSHKeys = keys

		fmt.Println(request)
		if err = pcc.Pcc.MaasDeploy(request); err != nil {
			assert.Fatalf("MaasDeploy failed: %v\n", err)
		}

		fmt.Println("Sleep for 8 minutes")
		time.Sleep(8 * time.Minute)

		for {
			for i, id := range nodesList {
				status, err := pcc.Pcc.GetProvisionStatus(id)
				if err != nil {
					fmt.Printf("Node %v error: %v\n", id, err)
					continue
				}
				if strings.Contains(status, "Ready") {
					fmt.Printf("Node %v has gone Ready\n", id)
					nodesList = removeIndex(i, nodesList)
					continue
				} else if strings.Contains(status, "reimage failed") {
					fmt.Printf("Node %v has failed reimage\n", id)
					nodesList = removeIndex(i, nodesList)
					continue
				}
				fmt.Printf("Node %v: %v\n", id, status)
			}
			if len(nodesList) == 0 {
				fmt.Printf("Brownfield re-image done\n")
				return
			}
			time.Sleep(60 * time.Second)
		}
	} else {
		assert.Fatalf("Failed to get the key %v\n", err)
	}
}

func removeIndex(i int, n []uint64) []uint64 {
	if len(n) > 1 {
		n = append(n[:i], n[i+1:]...)
		return n
	}
	return nil
}
