// Copyright © 2020 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package pcc

import (
	"fmt"
	"github.com/platinasystems/tiles/pccserver/models"
)

type ProvisionedApp struct {
	models.ProvisionedApp
}

func (p *PccClient) GetApps(nodeId uint64) (apps []ProvisionedApp, err error) {
	var endpoint string
	endpoint = fmt.Sprintf("pccserver/node/%d/apps", nodeId)
	err = p.Get(endpoint, &apps)

	return
}

func (p *PccClient) IsAppInstalled(nodeId uint64, appId string) (isInstalled bool, err error) {
	var apps []ProvisionedApp

	if apps, err = p.GetApps(nodeId); err == nil {
		for i := range apps {
			if apps[i].ID == appId && apps[i].Local.Installed {
				isInstalled = true
				return
			}
		}
	} else {
		fmt.Printf("Failed to GetApps: %v\n", err)
	}
	return
}

func (p *PccClient) IsRoleInstalled(nodeId uint64, roleId uint64) (isInstalled bool, err error) {
	return p.AreRoleInstalled(nodeId, []uint64{roleId})
}

func (p *PccClient) AreRoleInstalled(nodeId uint64, roles []uint64) (areInstalled bool, err error) {
	var node *NodeWithKubernetes

	if node, err = p.GetNode(nodeId); err == nil {
		areInstalled = true
	l1:
		for _, desidered := range roles {
			for _, role := range node.RoleIds {
				if desidered == role {
					continue l1
				}
			}
			return false, nil
		}
	}
	return
}
