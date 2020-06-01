package main

import (
	"fmt"
	pcc "github.com/platinasystems/pcc-blackbox/lib"
	"github.com/platinasystems/tiles/pccserver/models"
	"strings"
	"time"
)

const (
	FREQUENCY                            = 10
	AGENT_TIMEOUT                        = 180
	COLLECTOR_TIMEOUT                    = 180
	LLDP_TIMEOUT                         = 300
	MAAS_INSTALL_TIMEOUT                 = 300
	K8S_INSTALL_TIMEOUT                  = 1800
	PORTUS_TIMEOUT                       = 600
	PXEBOOT_TIMEOUT                      = 400
	PORTUS_NOTIFICATION                  = "PORTUS"
	COLLECTOR_NOTIFICATION               = "COLLECTOR"
	AGENT_NOTIFICATION                   = "AGENT"
	PXEBOOT_NODE_ADD_NOTIFICATION        = "new node added successfully"
	PXEBOOT_NODE_ADD_FAILED_NOTIFICATION = "add node at failed"
)

type status struct {
	msg     string
	isError bool
}

// Synchronize checking for installation
func syncCheckGenericInstallation(id uint64, appTimeout time.Duration, str2Check string, from time.Time, found chan status, breakLoop chan bool) {
	s := status{}
	timeout := appTimeout * time.Second
	for time.Since(from) < timeout {
		select {
		case <-breakLoop:
			return
		default:
			var (
				events []models.Notification
				err    error
			)
			events, err = pcc.Pcc.GetEvents()
			if err != nil {
				s.msg = fmt.Sprintf("failed to getEvents ERROR: %v", err)
				s.isError = true
				found <- s
			} else {
				for i := 0; i < len(events); i++ {
					if events[i].CreatedAt < ConvertToMillis(from) {
						continue
					}
					if strings.Contains(events[i].Message, str2Check) {
						s.msg = fmt.Sprintf("\"%v\" notification found in events", str2Check)
						s.isError = false
						found <- s
						break
					}
				}
			}
		}
		time.Sleep(FREQUENCY * time.Second)
	}

	s.msg = fmt.Sprint("timeout exceeded")
	s.isError = true
	found <- s
}
