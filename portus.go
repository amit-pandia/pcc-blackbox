package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	pcc "github.com/platinasystems/pcc-blackbox/lib"
	"github.com/platinasystems/test"
)

const (
	KEYMANAGER_ENDPOINT  = "key-manager"
	PORTUS_KEY_FILENAME  = "test_portus_key"
	PORTUS_CERT_FILENAME = "test_portus_crt"
)

var PortusSelectedNodeIds []uint64

func AddPortus(t *testing.T) {
	t.Run("addPortus", installPortus)
}

func CheckPortusInstallation(t *testing.T) {
	t.Run("checkPortus", checkPortus)
}

func UploadSecurityPortusKey(t *testing.T) {
	t.Run("uploadPortusKey", uploadSecurityKey_Portus)
}

func UploadSecurityPortusCert(t *testing.T) {
	t.Run("uploadSecurityPortusCert", uploadCertificate_Portus)
}

func uploadSecurityKey_Portus(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}
	err := CreateFileAndUpload(PORTUS_KEY_FILENAME, PORTUS_KEY,
		pcc.PRIVATE_KEY)
	if err != nil {
		assert.Fatalf(err.Error())
	}
}

func uploadCertificate_Portus(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}
	err := CreateFileAndUpload(PORTUS_CERT_FILENAME, PORTUS_CERT, pcc.CERT)
	if err != nil {
		assert.Fatalf(err.Error())
	}
}

func installPortus(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}
	var (
		portusConfiguration pcc.PortusConfiguration
	)

	for id, node := range Nodes {
		if pcc.Pcc.IsNodeOnline(node.Id) {
			portusConfiguration = Env.PortusConfiguration
			portusConfiguration.NodeID = id
			portusConfiguration.Name = fmt.Sprintf("portus_%v", id)

			if Env.AuthenticationProfile.Name == "" {
				fmt.Println("Authenticatiom Profile is not defined in the configuration file, Portus will be installed without it")
			} else {
				authProfile, err := pcc.Pcc.GetAuthProfileByName(CurrentAuthProfileName)
				if err == nil {
					portusConfiguration.AuthenticationProfileId = &authProfile.ID
				} else {
					fmt.Printf("Missing authentication profile %s\n, Portus will be installed without it", CurrentAuthProfileName)
				}
			}

			exist, certificate, err := pcc.Pcc.FindCertificate(PORTUS_CERT_FILENAME)
			if err != nil {
				fmt.Printf("Get certificate %s failed\n%v\n", PORTUS_CERT_FILENAME, err)
			} else if exist {
				portusConfiguration.RegistryCertId = &certificate.Id
			}

			exist, privateKey, err := pcc.Pcc.FindSecurityKey(PORTUS_KEY_FILENAME)
			if err != nil {
				fmt.Printf("Get private key %s failed\n%v\n", PORTUS_KEY_FILENAME, err)
			} else if exist {
				portusConfiguration.RegistryKeyId = &privateKey.Id
			}

			fmt.Printf("Installing Portus on Node with id %v\n",
				node.Id)

			err = pcc.Pcc.InstallPortusNode(portusConfiguration)
			if err != nil {
				fmt.Printf("Portus installation in %v failed\n%v\n", node.Host, err)
				fmt.Printf("Trying in another node\n")
			} else {
				PortusSelectedNodeIds = append(PortusSelectedNodeIds, node.Id)
				break
			}
		}
	}
	if len(PortusSelectedNodeIds) == 0 {
		assert.Fatal("Failed to install Portus: No available nodes\n")
	}
}

func checkPortus(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	for id, node := range Nodes {
		if idInSlice(node.Id, PortusSelectedNodeIds) {
			check, err := pcc.Pcc.WaitForInstallation(id, PORTUS_TIMEOUT, PORTUS_NOTIFICATION, "", nil)
			if err != nil {
				assert.Fatalf("Portus installation has failed\n%v\n", err)
			}
			if check {
				fmt.Printf("Portus correctly installed on nodeId:%v\n", id)
			}
		}
	}
}

func delAllPortus(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	var (
		portusConfigs []pcc.PortusConfiguration
		err           error
		id            uint64
	)

	portusConfigs, err = pcc.Pcc.GetPortusNodes()
	if err != nil {
		assert.Fatalf("Failed to get portus nodes: %v\n", err)
		return
	}

	for _, p := range portusConfigs {
		fmt.Printf("Deleting Portus %v\n", p.Name)
		id = p.ID
		err = pcc.Pcc.DelPortusNode(id, true)
		if err != nil {
			assert.Fatalf("Failed to delete Portus %v: %v\n",
				p.Name, err)
			return
		}
		// wait till deleted
		done := false
		timeout := time.After(10 * time.Minute)
		tick := time.Tick(30 * time.Second)
		for !done {
			select {
			case <-tick:
				_, err = pcc.Pcc.GetPortusNodeById(id)
				if err != nil {
					mesg := err.Error()
					if strings.Contains(mesg,
						"record not found") {
						done = true
						continue
					}
					assert.Fatalf("Failed Get Portus: %v\n",
						err)
					return
				}
			case <-timeout:
				assert.Fatal("Timeout deleting Portus\n")
			}
		}
	}
}
