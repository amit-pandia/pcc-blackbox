package main

import (
	"fmt"
	"github.com/lib/pq"
	"github.com/platinasystems/pcc-blackbox/lib"
	"github.com/platinasystems/test"
	"github.com/platinasystems/tiles/pccserver/models"
	"testing"
	"time"
	"strings"
)

var (
	startTime  time.Time
	cephConfig = &pcc.CephConfiguration{}
	isCephDeploy   = true
	isCephUndeploy = true
	cephClusterAlreadyExists = false
)

func testCeph(t *testing.T) {
	if t.Run("parseCephConfig", parseCephConfig) {
		if isCephDeploy {
			if run, ok := cephConfig.Tests[pcc.TestCreateCephCluster]; ok && run {
				if t.Run("createCephCluster", testCreateCephCluster) {
					if ! cephClusterAlreadyExists {
						t.Run("verifyCephInstallation", testVerifyCephInstallation)
					}
				}
			} else {
				fmt.Println("Ceph Cluster creation test is skipped")
			}
			if run, ok := cephConfig.Tests[pcc.TestCreateCephPools]; ok && run {
				if t.Run("createCephPool", testCreateCephPool) {
					t.Run("verifyCephPoolCreation", testVerifyCephPoolCreation)
				}
			} else {
				fmt.Println("Ceph Pools creation test is skipped")
			}
			if run, ok := cephConfig.Tests[pcc.TestCreateCephFS]; ok && run {
				if t.Run("createCephFS", testCreateCephFS) {
					t.Run("verifyCephFSCreation", testVerifyCephFSCreation)
				}
			} else {
				fmt.Println("Ceph FS creation test is skipped")
			}
		}
		if isCephUndeploy {
			if run, ok := cephConfig.Tests[pcc.TestDeleteCephFS]; ok && run {
				if t.Run("deleteCephFS", testDeleteCephFS) {
					t.Run("verifyCephFSDeletion", testVerifyCephFSDeletion)
				}
			} else {
				fmt.Println("Ceph FS deletion test is skipped")
			}
			if run, ok := cephConfig.Tests[pcc.TestDeleteCephPools]; ok && run {
				if t.Run("deleteCephPool", testDeleteCephPool) {
					t.Run("verifyCephPoolDeletion", testVerifyCephPoolDeletion)
				}
			} else {
				fmt.Println("Ceph Pools deletion test is skipped")
			}
			if run, ok := cephConfig.Tests[pcc.TestDeleteCephCluster]; ok && run {
				if t.Run("deleteCephCluster", testDeleteCephCluster) {
					t.Run("verifyCephUninstallation", testVerifyCephUninstallation)
				}
			} else {
				fmt.Println("Ceph Cluster deletion test is skipped")
			}
		}
	}
}

func parseCephConfig(t *testing.T) {
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
		*cephConfig = Env.CephConfiguration
		cephConfig.PccClient = Pcc
		if err = Pcc.ValidateCephConfig(cephConfig, identifier); err != nil {
			err = fmt.Errorf("Failed to validate Ceph Test config..ERROR:%v", err)
		}
	} else {
		err = fmt.Errorf("No unique identifier found")
	}
	if err != nil {
		assert.Fatalf("%v", err)
	}
}

func testCreateCephCluster(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	startTime = time.Now()
	err := createCephCluster(cephConfig)
	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		fmt.Println(errMsg)
		assert.Fatalf(errMsg)
		return
	}
}

func createCephCluster(cephConfig *pcc.CephConfiguration) (err error) {
	var (
		createRequest pcc.CreateCephClusterRequest
		clusterId     uint64
	)
	fmt.Println("Ceph cluster installation is starting")
	if createRequest, err = getCephCreateClusterRequest(cephConfig); err == nil {
		//TODO: Delete existing ceph cluster with same name if any
		clusterId, err = cephConfig.PccClient.CreateCephCluster(createRequest)
		if err != nil {
			if ! strings.Contains(err.Error(), "already exist") {
				errMsg := fmt.Sprintf("Ceph cluster deployment failed..ERROR:%v", err)
				err = fmt.Errorf(errMsg)
			} else {
				cephClusterAlreadyExists = true
				fmt.Printf("Ceph Cluster[%v] already exists\n", cephConfig.ClusterName)
				err = nil
			}
		} else {
			fmt.Println("Ceph cluster installation has started. Cluster id:", clusterId)
		}
		if clusterId != 0 {
			cephConfig.SetCephClusterId(clusterId)
		}
	}
	return
}

func getCephCreateClusterRequest(cephConfig *pcc.CephConfiguration) (createRequest pcc.CreateCephClusterRequest, err error) {
	var sNodes []pcc.CephNodes
	if sNodes, err = getNodesList(cephConfig); err != nil {
		return
	}
	createRequest = pcc.CreateCephClusterRequest{
		Name:  cephConfig.ClusterName,
		Nodes: sNodes,
		Tags:  pq.StringArray{"ROTATIONAL", "SATA", "PCIe"},
	}
	return
}

func testCreateCephPool(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	startTime = time.Now()
	err := createCephPool(cephConfig)
	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		fmt.Println(errMsg)
		assert.Fatalf(errMsg)
		return
	}
}

func createCephPool(cephConfig *pcc.CephConfiguration) (errAggr error) {
	var (
		poolId uint64
	)
	if clusterId := cephConfig.GetCephClusterId(); clusterId != 0 {
		for t, pools := range pcc.CephPools {
			for pool, _ := range pools {
				if createRequest, err := getCephPoolCreateRequest(pool, clusterId); err == nil {
					fmt.Printf("Ceph pool [%v] creation is starting\n", pool)
					poolId, err = cephConfig.PccClient.CreateCephPool(createRequest)
					if err != nil {
						errMsg := fmt.Sprintf("Ceph pool [%v] creation failed..ERROR:%v", pool, err)
						fmt.Println(errMsg)
						err = fmt.Errorf(errMsg)
						errAggr = fmt.Errorf(fmt.Sprint(errAggr) + fmt.Sprintf("%v\n", err))
						if poolId != 0 {
							pcc.CephPools[t][pool] = poolId
						}
					} else {
						pcc.CephPools[t][pool] = poolId
						fmt.Printf("Ceph pool [%v] creation has started. poolId: %v\n", pool, poolId)
					}
				}
			}
		}
	} else {
		errAggr = fmt.Errorf("No Ceph Cluster found")
	}
	return
}

func getCephPoolCreateRequest(name string, clusterId uint64) (createRequest pcc.CreateCephPoolRequest, err error) {
	createRequest = pcc.CreateCephPoolRequest{
		Name:          name,
		CephClusterId: clusterId,
		Size:          2,
		Quota:         "10000",
		QuotaUnit:     "MiB",
		PoolType:      models.CEPH_POOL_PROFILE_TYPE_REPLICATED.String(),
	}
	return
}

func testCreateCephFS(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	startTime = time.Now()
	err := createCephFS(cephConfig)
	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		fmt.Println(errMsg)
		assert.Fatalf(errMsg)
		return
	}
}

func createCephFS(cephConfig *pcc.CephConfiguration) (err error) {
	var (
		fsId uint64
	)
	fmt.Printf("Ceph FS [%v] creation is starting\n", pcc.CEPH_FS_NAME)
	if clusterId := cephConfig.GetCephClusterId(); clusterId != 0 {
		if createRequest, err := getCephFSCreateRequest(cephConfig, pcc.CEPH_FS_NAME, clusterId); err == nil {
			fsId, err = cephConfig.PccClient.CreateCephFS(createRequest)
			if err != nil {
				errMsg := fmt.Sprintf("Ceph FS [%v] creation failed..ERROR:%v", pcc.CEPH_FS_NAME, err)
				fmt.Println(errMsg)
				err = fmt.Errorf(errMsg)
			} else {
				fmt.Printf("Ceph FS [%v] creation has started. fsId: %v\n", pcc.CEPH_FS_NAME, fsId)
			}
		}
	} else {
		err = fmt.Errorf("No Ceph Cluster found")
	}
	return
}

func getCephFSCreateRequest(cephConfig *pcc.CephConfiguration, name string, clusterId uint64) (createRequest pcc.CreateCephFSRequest, err error) {
	var metadataPool, defaultPool, dataPool1, dataPool2 *models.CephPool
	metadataPool, err = cephConfig.PccClient.GetCephPool(pcc.CEPH_POOL_METADATA, clusterId)
	if err != nil {
		return
	}
	defaultPool, err = cephConfig.PccClient.GetCephPool(pcc.CEPH_POOL_DEFAULT, clusterId)
	if err != nil {
		return
	}
	dataPool1, err = cephConfig.PccClient.GetCephPool(pcc.CEPH_POOL_DATA_1, clusterId)
	if err != nil {
		return
	}
	dataPool2, err = cephConfig.PccClient.GetCephPool(pcc.CEPH_POOL_DATA_2, clusterId)
	if err != nil {
		return
	}
	createRequest = pcc.CreateCephFSRequest{
		Name:          name,
		CephClusterId: clusterId,
		MetadataPool:  metadataPool,
		DefaultPool:   defaultPool,
		DataPools:     []*models.CephPool{dataPool1, dataPool2},
	}
	return
}

func testDeleteCephFS(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	startTime = time.Now()
	err := deleteCephFS(cephConfig)
	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		fmt.Println(errMsg)
		assert.Fatalf(errMsg)
		return
	}
}

func deleteCephFS(cephConfig *pcc.CephConfiguration) (err error) {
	fmt.Printf("Ceph FS [%v] deletion is starting\n", pcc.CEPH_FS_NAME)
	time.Sleep(time.Second * 5)

	if clusterId := cephConfig.GetCephClusterId(); clusterId != 0 {
		cephFS, errGet := cephConfig.PccClient.GetCephFS(pcc.CEPH_FS_NAME, clusterId)
		if errGet != nil {
			err = fmt.Errorf("%v", errGet)
		} else {
			fsId := cephFS.Id
			if fsId != 0 {
				err = cephConfig.PccClient.DeleteCephFS(cephFS.Id)
				if err != nil {
					err = fmt.Errorf("Ceph FS [%v] deletion failed..ERROR: %v", pcc.CEPH_FS_NAME, err)
				} else {
					fmt.Printf("Ceph FS [%v] deletion has started. fsId: %v\n", pcc.CEPH_FS_NAME, fsId)
				}
			} else {
				err = fmt.Errorf("Ceph FS [%v] not found to delete", fsId)
			}
		}
	} else {
		err = fmt.Errorf("No Ceph Cluster found")
	}
	return
}

func testDeleteCephPool(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	startTime = time.Now()
	err := deleteCephPool(cephConfig)
	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		fmt.Println(errMsg)
		assert.Fatalf(errMsg)
		return
	}
}

func deleteCephPool(cephConfig *pcc.CephConfiguration) (errAggr error) {
	fmt.Println("Ceph pools deletion is starting")
	time.Sleep(time.Second * 5)

	if clusterId := cephConfig.GetCephClusterId(); clusterId != 0 {
		for _, pools := range pcc.CephPools {
			for pool, id := range pools {
				fmt.Printf("Ceph pool [%v] deletion is starting\n", pool)
				err := cephConfig.PccClient.DeleteCephPool(id)
				if err != nil {
					err = fmt.Errorf("Ceph pool [%v] deletion failed..ERROR: %v", pool, err)
					errAggr = fmt.Errorf(fmt.Sprint(errAggr) + fmt.Sprintf("%v\n", err))
				} else {
					fmt.Printf("Ceph pool [%v] deletion has started. poolId: %v\n", pool, id)
				}
			}
		}
	} else {
		errAggr = fmt.Errorf("No Ceph Cluster found")
	}
	return
}

func testDeleteCephCluster(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	startTime = time.Now()
	err := deleteCephCluster(cephConfig)
	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		fmt.Println(errMsg)
		assert.Fatalf(errMsg)
		return
	}
}

func deleteCephCluster(cephConfig *pcc.CephConfiguration) (err error) {
	fmt.Printf("Ceph cluster [%v] uninstallation is starting\n", cephConfig.ClusterName)
	time.Sleep(time.Second * 5)

	if clusterId := cephConfig.GetCephClusterId(); clusterId != 0 {
		err = cephConfig.PccClient.DeleteCephCluster(clusterId)
		if err != nil {
			err = fmt.Errorf("Ceph cluster deletion failed..ERROR: %v", err)
		} else {
			fmt.Println("Ceph cluster uninstallation has started. Cluster id:", clusterId)
		}
	} else {
		err = fmt.Errorf("No Ceph Cluster found")
	}
	return
}

func testVerifyCephFSCreation(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	err := verifyCephFSCreation(cephConfig)
	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		fmt.Println(errMsg)
		assert.Fatalf(errMsg)
		return
	}
}

func verifyCephFSCreation(cephConfig *pcc.CephConfiguration) (err error) {
	fmt.Printf("Verifying Ceph FS [%v] creation...Timeout:[%v sec]\n", pcc.CEPH_FS_NAME, pcc.CEPH_FS_CREATION_TIMEOUT)

	s, err := cephConfig.VerifyCeph(startTime, pcc.CEPH_FS_CREATE_EVENT, pcc.CEPH_FS_NAME)
	if err != nil {
		errMsg := fmt.Sprintf("Ceph FS [%v] creation verification failed...ERROR: %v", pcc.CEPH_FS_NAME, err)
		err = fmt.Errorf("%v", errMsg)
	} else {
		fmt.Printf("Ceph FS [%v] created properly..[%v]\n", pcc.CEPH_FS_NAME, s.Msg)
	}
	return
}

func testVerifyCephFSDeletion(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	err := verifyCephFSDeletion(cephConfig)
	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		fmt.Println(errMsg)
		assert.Fatalf(errMsg)
		return
	}
}

func verifyCephFSDeletion(cephConfig *pcc.CephConfiguration) (err error) {
	fmt.Printf("Verifying Ceph FS [%v] deletion...Timeout:[%v sec]\n", pcc.CEPH_FS_NAME, pcc.CEPH_FS_DELETION_TIMEOUT)

	s, err := cephConfig.VerifyCeph(startTime, pcc.CEPH_FS_DELETE_EVENT, pcc.CEPH_FS_NAME)
	if err != nil {
		errMsg := fmt.Sprintf("Ceph FS [%v] deletion verification failed...ERROR: %v", pcc.CEPH_FS_NAME, err)
		err = fmt.Errorf("%v", errMsg)
	} else {
		fmt.Printf("Ceph FS [%v] deleted properly..[%v]\n", pcc.CEPH_FS_NAME, s.Msg)
	}
	return
}

func testVerifyCephPoolCreation(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	err := verifyCephPoolCreation(cephConfig)
	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		fmt.Println(errMsg)
		assert.Fatalf(errMsg)
		return
	}
}

func verifyCephPoolCreation(cephConfig *pcc.CephConfiguration) (err error) {
	var errs []error
	for _, pools := range pcc.CephPools {
		for pool, _ := range pools {
			fmt.Printf("Verifying ceph pool [%v] creation...Timeout:[%v sec]\n", pool, pcc.CEPH_POOL_CREATION_TIMEOUT)
			_, errP := cephConfig.PccClient.GetCephPool(pool, cephConfig.ClusterId)
			if errP != nil {
				errMsg := fmt.Sprintf("Ceph pool [%v] creation failed..ERROR: %v", pool, errP)
				fmt.Println(errMsg)
				errs = append(errs, errP)
			} else {
				s, errT := cephConfig.VerifyCeph(startTime, pcc.CEPH_POOL_CREATE_EVENT, pool)
				if errT != nil {
					errMsg := fmt.Sprintf("Ceph pool [%v] creation verification failed...ERROR: %v", pool, errT)
					fmt.Println(errMsg)
					errs = append(errs, errT)
				} else {
					fmt.Printf("Ceph pool [%v] created successfully..[%v]\n", pool, s.Msg)
				}
			}
		}
	}
	if len(errs) > 0 {
		errMsg := fmt.Sprintf("Ceph pool creation failed: %v", errs)
		err = fmt.Errorf("%v", errMsg)
	}
	return
}

func testVerifyCephPoolDeletion(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	err := verifyCephPoolDeletion(cephConfig)
	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		fmt.Println(errMsg)
		assert.Fatalf(errMsg)
		return
	}
}

func verifyCephPoolDeletion(cephConfig *pcc.CephConfiguration) (err error) {
	var errs []error
	for _, pools := range pcc.CephPools {
		for pool, _ := range pools {
			fmt.Printf("Verifying ceph pool [%v] deletion...Timeout:[%v sec]\n", pool, pcc.CEPH_POOL_CREATION_TIMEOUT)
			s, errT := cephConfig.VerifyCeph(startTime, pcc.CEPH_POOL_DELETE_EVENT, pool)
			if errT != nil {
				errMsg := fmt.Sprintf("Ceph pool [%v] deletion verification failed...ERROR: %v", pool, errT)
				fmt.Println(errMsg)
				errs = append(errs, errT)
			} else {
				fmt.Printf("Ceph pool [%v] deleted successfully..[%v]\n", pool, s.Msg)
			}
		}
	}
	if len(errs) > 0 {
		errMsg := fmt.Sprintf("Ceph pool deletion failed: %v", errs)
		err = fmt.Errorf("%v", errMsg)
	}
	return
}

func testVerifyCephInstallation(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	err := verifyCephInstallation(cephConfig)
	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		fmt.Println(errMsg)
		assert.Fatalf(errMsg)
		return
	}
}

func verifyCephInstallation(cephConfig *pcc.CephConfiguration) (err error) {
	fmt.Printf("Verifying ceph cluster[%v] installation...Timeout:[%v sec]\n", cephConfig.ClusterName, pcc.CEPH_3_NODE_INSTALLATION_TIMEOUT)
	_, err = cephConfig.PccClient.GetCephCluster(cephConfig.ClusterName)
	if err != nil {
		errMsg := fmt.Sprintf("Ceph cluster[%v] installation verification failed...ERROR: %v", cephConfig.ClusterName, err)
		err = fmt.Errorf("%v", errMsg)
	} else {
		s, err := cephConfig.VerifyCeph(startTime, pcc.CEPH_CLUSTER_INSTALL_EVENT, cephConfig.ClusterName)
		if err != nil {
			errMsg := fmt.Sprintf("Ceph cluster[%v] installation verification failed...ERROR: %v", cephConfig.ClusterName, err)
			err = fmt.Errorf("%v", errMsg)
		} else {
			fmt.Printf("Ceph cluster [%v] deployed properly..[%v]\n", cephConfig.ClusterName, s.Msg)
		}
	}
	return
}

func testVerifyCephUninstallation(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	err := verifyCephUninstallation(cephConfig)
	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		fmt.Println(errMsg)
		assert.Fatalf(errMsg)
		return
	}
}

func verifyCephUninstallation(cephConfig *pcc.CephConfiguration) (err error) {
	fmt.Printf("Verifying ceph cluster[%v] uninstallation...Timeout:[%v sec]\n", cephConfig.ClusterName, pcc.CEPH_3_NODE_UNINSTALLATION_TIMEOUT)
	_, err = cephConfig.PccClient.GetCephCluster(cephConfig.ClusterName)
	if err != nil {
		errMsg := fmt.Sprintf("Ceph cluster[%v] uninstallation verification failed...ERROR: %v", cephConfig.ClusterName, err)
		err = fmt.Errorf("%v", errMsg)
	} else {
		s, err := cephConfig.VerifyCeph(startTime, pcc.CEPH_CLUSTER_UNINSTALL_EVENT, cephConfig.ClusterName)
		if err != nil {
			errMsg := fmt.Sprintf("Ceph cluster[%v] uninstallation failed...ERROR: %v\n", cephConfig.ClusterName, err)
			err = fmt.Errorf("%v", errMsg)
		} else {
			fmt.Printf("Ceph cluster [%v] undeployed properly..[%v]\n", cephConfig.ClusterName, s.Msg)
		}
	}
	return
}

func getNodesList(cephConfig *pcc.CephConfiguration) ([]pcc.CephNodes, error) {
	var (
		DIM                    = cephConfig.NumberOfNodes
		sNodes                 = make([]pcc.CephNodes, DIM)
		nodesSetCompleted bool = false
	)
	var j = 0
	for _, i := range Env.Invaders {
		sNodes[j] = pcc.CephNodes{ID: NodebyHostIP[i.HostIp]}
		j++
		if j == DIM {
			nodesSetCompleted = true
		}
	}
	if cephConfig.HighAvailability && len(sNodes) < 2 {
		return sNodes, fmt.Errorf("Atleast 2 invaders are required for High Availability configuration..Found: %v invaders", len(sNodes))
	}
	for _, i := range Env.Servers {
		if nodesSetCompleted {
			continue
		}
		sNodes[j] = pcc.CephNodes{ID: NodebyHostIP[i.HostIp]}
		j++
		if j == DIM {
			nodesSetCompleted = true
			continue
		}
	}
	return sNodes, nil
}
