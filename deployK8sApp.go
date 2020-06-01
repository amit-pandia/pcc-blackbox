package main

import (
	"fmt"
	"testing"
	"time"

	pcc "github.com/platinasystems/pcc-blackbox/lib"
	"github.com/platinasystems/test"
	"github.com/platinasystems/tiles/pccserver/models"
)

var (
	deployStartTime time.Time
	appConfig       = &pcc.K8sAppConfiguration{}
)

// starting point for k8s app deployment/undeployment testing
func testK8sApp(t *testing.T) {
	if t.Run("parseK8sAppConfig", parseK8sAppConfig) {
		if appConfig.K8sCluster == nil {
			if run, ok := appConfig.Tests[pcc.TestCreateK8sCluster]; ok && run {
				k8sname = appConfig.K8sClusterName
				if ! t.Run("createK8sCluster", createK8sCluster) {
					fmt.Println("Failed to create k8s cluster")
					return
				} else {
					cluster, err := appConfig.PccClient.GetKubernetesClusterByName(k8sname)
					if err != nil || cluster == nil {
						fmt.Printf("Failed to get K8s Cluster[%v] Object\n", k8sname)
						return
					}
					appConfig.K8sCluster = cluster
				}
			} else {
				fmt.Println("Create K8s Cluster test is skipped")
			}
		}
		if appConfig.K8sCluster != nil {
			if appConfig.CephStorageRequired {
				isCephUndeploy = false
				t.Run("testCeph", testCeph)
			}
			if run, ok := appConfig.Tests[pcc.TestCreateK8sStorageClass]; ok && run {
				if t.Run("createStorageClass", testCreateStorageClass) {
					t.Run("verifyStorageClassCreation", testVerifyStorageClassCreation)
				}
			} else {
				fmt.Println("Create K8s Storage Class test is skipped")
			}
			if run, ok := appConfig.Tests[pcc.TestDeployK8sApp]; ok && run {
				if t.Run("deployK8sApp", testDeployK8sApp) {
					t.Run("verifyK8sAppDeployment", testVerifyK8sAppDeployment)
				}
			} else {
				fmt.Println("Deploy K8s App test is skipped")
			}
			if run, ok := appConfig.Tests[pcc.TestUndeployK8sApp]; ok && run {
				if t.Run("undeployK8sApp", testUndeployK8sApp) {
					t.Run("verifyK8sAppUndeployment", testVerifyK8sAppUnDeployment)
				}
			} else {
				fmt.Println("Undeploy K8s App test is skipped")
			}
			if run, ok := appConfig.Tests[pcc.TestDeleteK8sStorageClass]; ok && run {
				if t.Run("deleteStorageClass", testDeleteStorageClass) {
					t.Run("verifyStorageClassDeletion", testVerifyStorageClassDeletion)
				}
			} else {
				fmt.Println("Delete K8s Storage Class test is skipped")
			}
			if appConfig.CephStorageRequired {
				isCephDeploy = false
				isCephUndeploy = true
				t.Run("testCeph", testCeph)
			}
			if run, ok := appConfig.Tests[pcc.TestDeleteK8sCluster]; ok && run {
				t.Run("deleteK8sCluster", deleteK8sCluster)
			} else {
				fmt.Println("Delete K8s Cluster test is skipped")
			}
		} else {
			fmt.Printf("No K8s cluster found to perform further tests\n")
			return
		}
	}

}

func parseK8sAppConfig(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}
	var (
		identifier string
		err error
	)
	if len(Env.Invaders) > 0 {
		identifier = Env.Invaders[0].HostIp
	} else if len(Env.Servers) > 0 {
		identifier = Env.Servers[0].HostIp
	}
	if identifier != "" {
		*appConfig = Env.K8sAppConfiguration
		appConfig.PccClient = pcc.Pcc
		if err = pcc.Pcc.ValidateAppConfig(appConfig, identifier); err != nil {
			err = fmt.Errorf("Failed to validate k8s app Test config..ERROR:%v", err)
		}
	} else {
		err = fmt.Errorf("No unique identifier found")
	}

	if err != nil {
		assert.Fatalf("%v", err)
	}
}

func testCreateStorageClass(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}
	deployStartTime = time.Now()
	err := createStorageClass(appConfig)
	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		fmt.Println(errMsg)
		assert.Fatalf(errMsg)
		return
	}
}

func createStorageClass(appConfig *pcc.K8sAppConfiguration) (err error) {
	if appConfig.CephStorageRequired {
		var (
			createStorageClassRequest = pcc.CephStorageClassConfig {
				CniPlugin: appConfig.K8sCluster.CniPlugin,
				K8sVersion: appConfig.K8sCluster.K8sVersion,
			}
		)
		fmt.Println("ceph storage class creation is starting")
		for _, pool := range []string{ pcc.CEPH_POOL_K8S_1, pcc.CEPH_POOL_DATA_1} {
			if err = getCreateCephStorageClassRequest(appConfig, pool, &createStorageClassRequest); err != nil {
				return err
			}
		}
		err = appConfig.PccClient.CreateCephStorageClass(createStorageClassRequest, appConfig.K8sCluster.ID)
		if err != nil {
			errMsg := fmt.Sprintf("StorageClass creation failed..ERROR:%v", err)
			fmt.Println(errMsg)
			err = fmt.Errorf(errMsg)
		}
	} else {
		err = fmt.Errorf("Storage Class creation without Ceph Storage is not implemented in PCC")
	}
	return
}

func getCreateCephStorageClassRequest(appConfig *pcc.K8sAppConfiguration, poolName string, createStorageClassRequest *pcc.CephStorageClassConfig) (err error) {
	var (
		pool *models.CephPool
		sc_name string
	)
	if poolName == pcc.CEPH_POOL_DATA_1 {
		sc_name = fmt.Sprintf(pcc.K8S_STORAGE_CLASS_NAME_CEPHFS, cephConfig.ClusterId, pcc.CEPH_POOL_DATA_1)
	}
	if poolName == pcc.CEPH_POOL_K8S_1 {
		sc_name = fmt.Sprintf(pcc.K8S_STORAGE_CLASS_NAME_CEPH_RBD, cephConfig.ClusterId, pcc.CEPH_POOL_K8S_1)
	}
	if _, ok := appConfig.StorageClasses[sc_name]; !ok {
		appConfig.StorageClasses[sc_name] = 0
	}
	pool, err = appConfig.PccClient.GetCephPool(poolName, cephConfig.ClusterId)
	if err == nil {
		createStorageClassRequest.PoolIds = append(createStorageClassRequest.PoolIds, pool.Id)
	}
	return
}

func testDeployK8sApp(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}
	deployStartTime = time.Now()
	err := deployK8sApp(appConfig)
	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		fmt.Println(errMsg)
		assert.Fatalf(errMsg)
		return
	}
}

func deployK8sApp(appConfig *pcc.K8sAppConfiguration) (err error) {
	var (
		deployRequest pcc.DeployAppRequest
	)
	fmt.Println("K8s App deployment is starting")
	if deployRequest, err = getAppDeployRequest(appConfig); err == nil {
		err = appConfig.PccClient.DeployK8sApp(deployRequest, appConfig.K8sCluster.ID)
		if err != nil {
			errMsg := fmt.Sprintf("K8s App deployment failed..ERROR:%v", err)
			err = fmt.Errorf(errMsg)
		}
	}
	return
}

func getAppDeployRequest(appConfig *pcc.K8sAppConfiguration) (deployRequest pcc.DeployAppRequest, err error) {
	deployRequest = pcc.DeployAppRequest{}
	base64EncodedFileData := ""
	for _, app := range appConfig.Apps {
		if app.HelmFilePath != "" {
			base64EncodedFileData, err = appConfig.ParseAndEncode(app)
		}
		tmp := &models.KApp{
			AppName:          app.AppName,
			AppNamespace:     app.AppNamespace,
			Label:            app.Label,
			GitBranch:        app.GitBranch,
			GitRepoPath:      app.GitRepoPath,
			GitUrl:           app.GitUrl,
			HelmValuesBase64: base64EncodedFileData,
		}
		deployRequest.Apps = append(deployRequest.Apps, tmp)
	}

	return
}

func testUndeployK8sApp(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	deployStartTime = time.Now()
	err := undeployK8sApp(appConfig)
	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		fmt.Println(errMsg)
		assert.Fatalf(errMsg)
		return
	}
}

func undeployK8sApp(appConfig *pcc.K8sAppConfiguration) (err error) {
	fmt.Printf("K8s Apps undeploying on [%v] cluster  is starting\n", appConfig.K8sClusterName)
	time.Sleep(time.Second * 10)
	if len(appConfig.AppIds) > 0 {
		appUndeployRequest := pcc.UndeployAppRequest{
			AppIds: appConfig.AppIds,
		}
		err = appConfig.PccClient.UnDeployK8sApp(appUndeployRequest, appConfig.GetK8sClusterId())
		if err != nil {
			err = fmt.Errorf("K8s App undeployment failed..ERROR: %v", err)
		} else {
			fmt.Println("K8s App undeployment started. AppIds:", appConfig.AppIds)
		}
	} else {
		err = fmt.Errorf("No k8s apps found")
	}
	return
}

func testDeleteStorageClass(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}
	err := deleteStorageClass(appConfig)
	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		fmt.Println(errMsg)
		assert.Fatalf(errMsg)
		return
	}
}

func deleteStorageClass(appConfig *pcc.K8sAppConfiguration) (err error) {
	time.Sleep(time.Second * 10)
	var request pcc.DeleteStorageClassRequest
	request, err = getDeleteStorageClassRequest(appConfig)
	if err == nil {
		err = appConfig.PccClient.DeleteStorageClass(request, appConfig.K8sCluster.ID)
		if err != nil {
			err = fmt.Errorf("Deletion of Storage Class has failed..ERROR: %v", err)
		} else {
			fmt.Printf("Deletion of Storage classes[%v] has started\n", appConfig.StorageClasses)
		}
	}
	return
}

func getDeleteStorageClassRequest(appConfig *pcc.K8sAppConfiguration) (request pcc.DeleteStorageClassRequest, err error) {
	for _, id := range appConfig.StorageClasses {
		if id != 0 {
			request.StorageclassIds = append(request.StorageclassIds, id)
		}
	}
	if len(request.StorageclassIds) == 0 {
		err = fmt.Errorf("No Storage Classes found to delete")
	}
	return
}

func testVerifyK8sAppDeployment(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	err := verifyK8sAppDeployment(appConfig)
	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		fmt.Println(errMsg)
		assert.Fatalf(errMsg)
		return
	}
}

func verifyK8sAppDeployment(appConfig *pcc.K8sAppConfiguration) (err error) {
	fmt.Printf("Verifying K8s apps deployment on cluster [%v]...Timeout:[%v sec]\n", appConfig.K8sClusterName, pcc.K8S_APP_DEPLOYMENT_TIMEOUT)
	_, err = appConfig.VerifyK8sApp(deployStartTime, pcc.K8S_APP_DEPLOY_EVENT, appConfig.K8sClusterName)
	if err != nil {
		err = fmt.Errorf("Apps deployment verification on cluster [%v] failed...ERROR: %v\n", appConfig.K8sClusterName, err)
	} else {
		for _, app := range appConfig.Apps {
			id, errGet := appConfig.PccClient.GetK8sAppId(app.Label, appConfig.GetK8sClusterId())
			if errGet != nil {
				err = fmt.Errorf("Failed to get AppId of K8s app[%v] ERROR: %v", app.Label, errGet)
				return
			}
			appConfig.AppIds = append(appConfig.AppIds, id)
			fmt.Printf("K8s App[%v] deployed on cluster[%v]. AppId: [%d]\n", app.Label, appConfig.K8sClusterName, id)
		}
	}
	return
}

func testVerifyK8sAppUnDeployment(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	err := verifyK8sAppUnDeployment(appConfig)
	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		fmt.Println(errMsg)
		assert.Fatalf(errMsg)
		return
	}
}

func verifyK8sAppUnDeployment(appConfig *pcc.K8sAppConfiguration) (err error) {
	fmt.Printf("Verifying K8s app undeployment on cluster [%v]...Timeout:[%v sec]\n", appConfig.K8sClusterName, pcc.K8S_APP_DEPLOYMENT_TIMEOUT)
	s, err := appConfig.VerifyK8sApp(deployStartTime, pcc.K8S_APP_UNDEPLOY_EVENT, appConfig.K8sClusterName)
	if err != nil {
		errMsg := fmt.Sprintf("K8s App undeployment verification on cluster [%v] failed...ERROR: %v", appConfig.K8sClusterName, err)
		err = fmt.Errorf("%v", errMsg)
	} else {
		fmt.Printf("K8s Apps undeployed on cluster [%v] properly..[%v]\n", appConfig.K8sClusterName, s.Msg)
	}
	return
}

func testVerifyStorageClassCreation(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}
	err := verifyStorageClassCreation(appConfig)
	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		fmt.Println(errMsg)
		assert.Fatalf(errMsg)
		return
	}
}

func verifyStorageClassCreation(appConfig *pcc.K8sAppConfiguration) (err error) {
	fmt.Printf("Verifying storage class creation on cluster [%v]...Timeout:[%v sec]\n", appConfig.K8sClusterName, pcc.K8S_STORAGE_CLASS_CREATION_TIMEOUT)
	_, err = appConfig.VerifyK8sApp(deployStartTime, pcc.K8S_STORAGE_CLASS_CREATION_EVENT, appConfig.K8sClusterName)
	if err != nil {
		errMsg := fmt.Sprintf("Storage class creation verification on cluster [%v] failed...ERROR: %v", appConfig.K8sClusterName, err)
		err = fmt.Errorf("%v", errMsg)
	} else {
		scFailed := []string{}
		for name, _ := range appConfig.StorageClasses {
			id, errGet := appConfig.PccClient.GetStorageClassId(appConfig.K8sCluster.ID, name)
			if errGet == nil {
				appConfig.StorageClasses[name] = id
			} else {
				scFailed = append(scFailed, fmt.Sprintf("%s;", errGet.Error()))
			}
		}
		if len(scFailed) > 0{
			err = fmt.Errorf("Failed to create StorageClasses[%v]", scFailed)
		}else {
			fmt.Printf("Successfully verified Storage Classes %v creation on cluster[%v]\n", appConfig.StorageClasses, appConfig.K8sClusterName)
		}
	}
	return
}

func testVerifyStorageClassDeletion(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}
	err := verifyStorageClassDeletion(appConfig)
	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		fmt.Println(errMsg)
		assert.Fatalf(errMsg)
		return
	}
}

func verifyStorageClassDeletion(appConfig *pcc.K8sAppConfiguration) (err error) {
	fmt.Printf("Verifying storage class Deletion on cluster [%v]...Timeout:[%v sec]\n", appConfig.K8sClusterName, pcc.K8S_STORAGE_CLASS_DELETION_TIMEOUT)
	_, err = appConfig.VerifyK8sApp(deployStartTime, pcc.K8S_STORAGE_CLASS_DELETION_EVENT, appConfig.K8sClusterName)
	if err != nil {
		errMsg := fmt.Sprintf("Storage class deletion verification on cluster [%v] failed...ERROR: %v", appConfig.K8sClusterName, err)
		err = fmt.Errorf("%v", errMsg)
	} else {
		fmt.Printf("Successfully verified Storage classes[%v] deletion on cluster [%v] properly\n", appConfig.StorageClasses, appConfig.K8sClusterName)
	}
	return
}