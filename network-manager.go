package main

import (
	"fmt"
	pcc "github.com/platinasystems/pcc-blackbox/lib"
	"github.com/platinasystems/test"
	"strings"
	"testing"
	"time"
)

var (
	networkConfig               = &pcc.NetworkConfiguration{}
	isNetworkingDeploy          = true
	isNetworkingUndeploy        = true
	networkClusterAlreadyExists = false
)

func testNetworking(t *testing.T) {
	if t.Run("parseNetworkingConfig", parseNetworkingConfig) {
		if isNetworkingDeploy {
			if run, ok := networkConfig.Tests[pcc.TestCreateNetworkCluster]; ok && run {
				if t.Run("createNetworkCluster", testCreateNetworkCluster) {
					if !networkClusterAlreadyExists {
						t.Run("verifyNetworkInstallation", testVerifyNetworkingInstallation)
					}
				}
			} else {
				fmt.Println("Network Cluster creation test is skipped")
			}

			if isNetworkingUndeploy {
				if run, ok := networkConfig.Tests[pcc.TestDeleteNetworkCluster]; ok && run {
					if t.Run("deleteNetworkCluster", testDeleteNetworkCluster) {
						t.Run("verifyNetworkUninstallation", testVerifyNetworkingUninstallation)
					}
				} else {
					fmt.Println("Network Cluster deletion test is skipped")
				}
			}
		}
	}
}

func parseNetworkingConfig(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	var (
		identifier string
		err        error
	)
	if len(Env.Invaders) > 0 {
		identifier = Env.Invaders[0].HostIp
	} else if len(Env.Servers) > 0 {
		identifier = Env.Servers[0].HostIp
	}
	if identifier != "" {
		*networkConfig = Env.NetworkConfiguration
		networkConfig.PccClient = Pcc
		if err = Pcc.ValidateNetworkConfig(networkConfig, identifier); err != nil {
			err = fmt.Errorf("Failed to validate Networking Test config..ERROR:%v", err)
		}
	} else {
		err = fmt.Errorf("No unique identifier found")
	}
	if err != nil {
		assert.Fatalf("%v", err)
	}
}

func testCreateNetworkCluster(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	startTime = time.Now()
	err := createNetworkCluster(networkConfig)
	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		fmt.Println(errMsg)
		assert.Fatalf(errMsg)
		return
	}
}

func testVerifyNetworkingInstallation(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	err := verifyNetworkingInstallation(networkConfig)
	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		fmt.Println(errMsg)
		assert.Fatalf(errMsg)
		return
	}
}

func testDeleteNetworkCluster(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	startTime = time.Now()
	err := deleteNetworkCluster(networkConfig)
	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		fmt.Println(errMsg)
		assert.Fatalf(errMsg)
		return
	}
}

func testVerifyNetworkingUninstallation(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	err := verifyNetworkingUninstallation(networkConfig)
	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		fmt.Println(errMsg)
		assert.Fatalf(errMsg)
		return
	}
}

func createNetworkCluster(networkConfig *pcc.NetworkConfiguration) (err error) {
	var (
		createRequest pcc.CreateNetworkClusterRequest
		clusterId     uint64
	)
	fmt.Println("Network cluster installation is starting")
	if createRequest, err = getNetworkCreateClusterRequest(networkConfig); err == nil {
		//TODO: Delete existing network cluster with same name if any
		clusterId, err = networkConfig.PccClient.CreateNetworkCluster(createRequest)
		if err != nil {
			if ! strings.Contains(err.Error(), "already exist") {
				errMsg := fmt.Sprintf("Network cluster deployment failed..ERROR:%v", err)
				err = fmt.Errorf(errMsg)
			} else {
				networkClusterAlreadyExists = true
				fmt.Printf("Network Cluster[%v] already exists\n", networkConfig.ClusterName)
				err = nil
			}
		} else {
			fmt.Println("Network cluster installation has started. Cluster id:", clusterId)
		}
		if clusterId != 0 {
			networkConfig.SetNetworkClusterId(clusterId)
		}
	}
	return
}

func verifyNetworkingInstallation(networkConfig *pcc.NetworkConfiguration) (err error) {
	fmt.Printf("Verifying network cluster[%v] installation...Timeout:[%v sec]\n", networkConfig.ClusterName, pcc.NETWORK_3_NODE_INSTALLATION_TIMEOUT)
	_, err = networkConfig.PccClient.GetNetworkCluster(networkConfig.ClusterName)
	if err != nil {
		errMsg := fmt.Sprintf("Network cluster[%v] installation verification failed...ERROR: %v", networkConfig.ClusterName, err)
		err = fmt.Errorf("%v", errMsg)
	} else {
		s, err := networkConfig.VerifyNetwork(startTime, pcc.NETWORK_CLUSTER_INSTALL_EVENT, networkConfig.ClusterName)
		if err != nil {
			errMsg := fmt.Sprintf("Network cluster[%v] installation verification failed...ERROR: %v", networkConfig.ClusterName, err)
			err = fmt.Errorf("%v", errMsg)
		} else {
			fmt.Printf("Network cluster [%v] deployed properly..[%v]\n", networkConfig.ClusterName, s.Msg)
		}
	}
	return
}

func deleteNetworkCluster(networkConfig *pcc.NetworkConfiguration) (err error) {
	fmt.Printf("Network cluster [%v] uninstallation is starting\n", networkConfig.ClusterName)
	time.Sleep(time.Second * 5)

	if clusterId := networkConfig.GetNetworkClusterId(); clusterId != 0 {
		err = networkConfig.PccClient.DeleteNetworkCluster(clusterId)
		if err != nil {
			err = fmt.Errorf("Network cluster deletion failed..ERROR: %v", err)
		} else {
			fmt.Println("Network cluster uninstallation has started. Cluster id:", clusterId)
		}
	} else {
		err = fmt.Errorf("No Network Cluster found")
	}
	return
}

func verifyNetworkingUninstallation(networkConfig *pcc.NetworkConfiguration) (err error) {
	fmt.Printf("Verifying network cluster[%v] uninstallation...Timeout:[%v sec]\n", networkConfig.ClusterName, pcc.NETWORK_3_NODE_UNINSTALLATION_TIMEOUT)
	_, err = networkConfig.PccClient.GetNetworkCluster(networkConfig.ClusterName)
		if strings.Contains(err.Error(), "not found") {err = nil}
	if err != nil {
		errMsg := fmt.Sprintf("network cluster[%v] uninstallation verification failed...ERROR: %v", networkConfig.ClusterName, err)
		err = fmt.Errorf("%v", errMsg)
	} else {
		s, err := networkConfig.VerifyNetwork(startTime, pcc.NETWORK_CLUSTER_UNINSTALL_EVENT, networkConfig.ClusterName)
		if err != nil {
			errMsg := fmt.Sprintf("Network cluster[%v] uninstallation failed...ERROR: %v\n", networkConfig.ClusterName, err)
			err = fmt.Errorf("%v", errMsg)
		} else {
			fmt.Printf("Network cluster [%v] undeployed properly..[%v]\n", networkConfig.ClusterName, s.Msg)
		}
	}
	return
}

func getNetworkCreateClusterRequest(networkConfig *pcc.NetworkConfiguration) (createRequest pcc.CreateNetworkClusterRequest, err error) {
	var sNodes []pcc.NetworkNodes
	if sNodes, err = getNetworkNodesList(networkConfig); err != nil {
		return
	}
	createRequest = pcc.CreateNetworkClusterRequest{
		Name:  networkConfig.ClusterName,
		Nodes: sNodes,
	}
	createRequest.ControlCIDR = networkConfig.ControlCIDR
	createRequest.IgwPolicy = networkConfig.IgwPolicy
	return
}

func getNetworkNodesList(networkConfig *pcc.NetworkConfiguration) ([]pcc.NetworkNodes, error) {
	var (
		DIM                    = networkConfig.NumberOfNodes
		sNodes                 = make([]pcc.NetworkNodes, DIM)
		nodesSetCompleted bool = false
	)
	var j = 0
	for _, i := range Env.Invaders {
		sNodes[j] = pcc.NetworkNodes{ID: NodebyHostIP[i.HostIp]}
		j++
		if j == DIM {
			nodesSetCompleted = true
		}
	}
	if networkConfig.HighAvailability && len(sNodes) < 2 {
		return sNodes, fmt.Errorf("Atleast 2 invaders are required for High Availability configuration..Found: %v invaders", len(sNodes))
	}
	for _, i := range Env.Servers {
		if nodesSetCompleted {
			continue
		}
		sNodes[j] = pcc.NetworkNodes{ID: NodebyHostIP[i.HostIp]}
		j++
		if j == DIM {
			nodesSetCompleted = true
			continue
		}
	}
	return sNodes, nil
}
