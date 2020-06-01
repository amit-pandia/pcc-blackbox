package main

import (
	"fmt"
	"github.com/platinasystems/tiles/pccserver/executor"
	"strings"
	"testing"
	"time"

	pcc "github.com/platinasystems/pcc-blackbox/lib"
	"github.com/platinasystems/test"
)

var k8sname string = "k8stest"

func createK8sCluster(t *testing.T) {
	t.Run("CreateK8sCluster", createK8s_3nodes)
	t.Run("ValidateK8sCluster", validateK8sCluster)
}

func createK8s_3nodes(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}
	const DIM = 3
	var (
		err               error
		k8sRequest        pcc.K8sClusterRequest
		k8sNodes               = make([]pcc.K8sNodes, DIM)
		nodesSetCompleted bool = false
	)

	if (len(Env.Invaders) + len(Env.Servers)) < DIM {
		assert.Fatalf("Needed at least %d nodes for creating a cluster", DIM)
	}

	var j = 0
	for _, i := range Env.Invaders {
		k8sNodes[j] = pcc.K8sNodes{ID: NodebyHostIP[i.HostIp]}
		j++
		if j == DIM {
			nodesSetCompleted = true
			continue
		}
	}
	for _, i := range Env.Servers {
		if nodesSetCompleted {
			continue
		}
		k8sNodes[j] = pcc.K8sNodes{ID: NodebyHostIP[i.HostIp]}
		j++
		if j == DIM {
			nodesSetCompleted = true
			continue
		}
	}
	k8sRequest = pcc.K8sClusterRequest{
		ID:         0,         //todo dynamic counter
		Name:       k8sname,   //todo dynamic
		K8sVersion: "v1.14.3", //todo dynamic
		CniPlugin:  "kube-router",
		Nodes:      k8sNodes,
	}
	err = pcc.Pcc.CreateKubernetes(k8sRequest)
	if err != nil {
		assert.Fatalf("%v", err)
		return
	}
}

func validateK8sCluster(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	var (
		id  uint64
		err error
	)

	id, err = pcc.Pcc.FindKubernetesId(k8sname)
	if err != nil {
		assert.Fatalf("Failed to find cluster %v: %v", k8sname, err)
		return
	}

	timeout := time.After(45 * time.Minute)
	tick := time.Tick(1 * time.Minute)
	done := false
	var last_percent int8 = -1
	for !done {
		select {
		case <-timeout:
			assert.Fatalf("Timed out waiting for Kubernetes")
			return
		case <-tick:
			status, percent, err := pcc.Pcc.GetKubernetesDeployStatus(id)
			if err != nil {
				assert.Fatalf("Failed to get deploy status "+
					"%v\n", err)
				return
			}

			switch status {
			case pcc.K8S_DEPLOY_STATUS_PROGRESS:
				if percent != last_percent {
					fmt.Printf("Cluster %v = %v  %v%%\n",
						id, status, percent)
					last_percent = percent
				}
			case pcc.K8S_DEPLOY_STATUS_COMPLETED:
				fmt.Println("Kubernetes cluster installed")
				done = true
			case pcc.K8S_DEPLOY_STATUS_FAILED:
				assert.Fatalf("Kubernetes cluster install " +
					"failed")
			case pcc.K8S_DEPLOY_APP_STATUS_PROGRESS:
				fmt.Println("Kubernetes app progress")
			default:
				assert.Fatalf("Unexpected status - %v\n",
					status)
				return
			}
		}
	}

	timeout = time.After(5 * time.Minute)
	tick = time.Tick(5 * time.Second)
	done = false
	for !done {
		select {
		case <-timeout:
			assert.Fatalf("health check timed out\n")
			return
		case <-tick:
			health, err := pcc.Pcc.GetKubernetesHealth(id)
			if err != nil {
				assert.Fatalf("Error geting K8s health\n")
				return
			}
			fmt.Printf("Kubernetes health = %v\n", health)
			if health == "good" {
				done = true
				return
			}

		}
	}
}

func deleteK8sCluster(t *testing.T) {
	t.Run("DeleteK8sCluster", deleteAllK8sCluster)
}

func deleteAllK8sCluster(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	clusters, err := pcc.Pcc.GetKubernetes()
	if err != nil {
		assert.Fatalf("Failed to get kubernetes clusters: %v\n", err)
		return
	}

	for _, c := range clusters {
		err := pcc.Pcc.DeleteKubernetes(c.ID, false)
		if err != nil {
			fmt.Printf("delete K8s cluster failed, try force: %v",
				err)
			err := pcc.Pcc.DeleteKubernetes(c.ID, true)
			if err != nil {
				assert.Fatalf("force delete failed: %v", err)
				return
			}
		}

		timeout := time.After(10 * time.Minute)
		tick := time.Tick(5 * time.Second)
		var last_percent int8 = -1
		for {
			select {
			case <-timeout:
				assert.Fatalf("Time out deleting Kubernetes")
				return
			case <-tick:
				cluster, err := pcc.Pcc.GetKubernetesId(c.ID)
				if err != nil && strings.Contains(err.Error(),
					"doesn't exist") {
					fmt.Printf("K8s delete OK\n")
					return
				}
				if err != nil {
					assert.Fatalf("get cluster failed: %v",
						err)
					return
				}
				var percent int8
				if clusterTask, ok := (cluster.Task).(*executor.Task); ok{
					percent = clusterTask.Progress
				}
				if percent != last_percent {
					fmt.Printf("delete status: %v  %v%%\n",
						cluster.DeployStatus, percent)
					last_percent = percent
				}
			}
		}
	}
}
