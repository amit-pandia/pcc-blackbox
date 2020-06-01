// Copyright © 2020 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package pcc

import (
	"fmt"
	"github.com/platinasystems/go-common/http"
)

type Credential struct {
	UserName string `json:"username"`
	Password string `json:"password"`
}

type PccClient struct {
	pccIp      string
	bearer     string
	RestClient *http.PlatinaRestService
}

var Pcc *PccClient

func (pcc *PccClient) getClient() (rc *http.PlatinaRestService) {
	if pcc.RestClient == nil {
		var prs http.PlatinaRestService
		prc := http.PlatinaRestClient{Address: pcc.pccIp, Token: pcc.bearer, Port: 9999}
		prs = &prc
		rc = &prs
	} else {
		rc = pcc.RestClient
	}

	return
}

// GET
func (pcc *PccClient) Get(endPoint string, out interface{}) (err error) {
	err = (*pcc.getClient()).Get(endPoint, out)
	return
}

// DELETE
func (pcc *PccClient) Delete(endPoint string, data interface{}, out interface{}) (err error) {
	err = (*pcc.getClient()).Delete(endPoint, data, out)
	return
}

// POST
func (pcc *PccClient) Post(endPoint string, data interface{}, out interface{}) (err error) {
	err = (*pcc.getClient()).Post(endPoint, data, out)
	return
}

// PUT
func (pcc *PccClient) Put(endPoint string, data interface{}, out interface{}) (err error) {
	err = (*pcc.getClient()).Put(endPoint, data, out)
	return
}

// PUT
func (pcc *PccClient) PutFile(endPoint string, filePath string, fields map[string]string, out interface{}) (err error) { // FIXME all services should share the same structure
	err = (*pcc.getClient()).PutFile(endPoint, filePath, fields, out)
	return
}

func (pcc *PccClient) GetFile(endPoint string) (content string, err error) {
	content, err = (*pcc.getClient()).GetFile(endPoint)
	return
}

func Authenticate(PccIp string, cred Credential) (pcc *PccClient, err error) {
	var out struct{ Token string }
	rc := http.PlatinaRestClient{Address: PccIp, Port: 9999}
	client := PccClient{pccIp: PccIp}
	if err = rc.Post("security/auth", cred, &out); err == nil {
		client.bearer = fmt.Sprintf("Bearer %s", out.Token)
	}
	pcc = &client
	return
}
