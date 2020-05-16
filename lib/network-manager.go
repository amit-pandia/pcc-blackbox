package pcc

import (
	"fmt"
	"strings"
	"time"
)

const (
	NETWORK_CLUSTER_NAME_PREFIX           = "networktest"

	TestCreateNetworkCluster = "createCluster"
	TestDeleteNetworkCluster = "deleteCluster"

	NETWORK_3_NODE_INSTALLATION_TIMEOUT   = 1000
	NETWORK_3_NODE_UNINSTALLATION_TIMEOUT = 1000

	// For Verification
	NETWORK_CLUSTER_INSTALL_EVENT   = "networkClusterInstall"
	NETWORK_CLUSTER_UNINSTALL_EVENT = "networkClusterUninstall"
)

type NetworkNodes struct {
	ID uint64
}

type NetworkConfiguration struct {
	ClusterName      string          `json:"clusterName"`
	ClusterId        uint64          `json:"clusterId"`
	HighAvailability bool            `json:"highAvailability"`
	NumberOfNodes    int             `json:"numberOfNodes"`
	IgwPolicy        string          `json:"igwPolicy"`
	ControlCIDR      string          `json:"controlCIDR"`
	Tests            map[string]bool `json:"tests"`
	PccClient        *PccClient
}

type CreateNetworkClusterRequest struct {
	Name        string         `json:"name"`
	Nodes       []NetworkNodes `json:"nodes"`
	IgwPolicy   string         `json:"igwPolicy"`
	ControlCIDR string         `json:"controlCIDR"`
}

type NetworkClusterUI struct {
	Id                 uint64                `json:"id" gorm:"id"`
	Name               string                `json:"name" gorm:"name"`
	ControlCIDR        string                `json:"controlCIDR" gorm:"column:control_cidr" validate:"required,cidrv4"`
	DeployStatus       string                `json:"deploy_status" gorm:"deploy_status"`
	IgwPolicy          string                `json:"igwPolicy" validate:"required"`
	ProgressPercentage int8                  `json:"progressPercentage" gorm:"-"`
	Nodes              []*NetworkClusterNode `json:"nodes" gorm:"foreignkey:network_cluster_id"`
}

type NetworkCluster struct {
	NetworkClusterUI
	State           string                                `json:"state, omitempty" gorm:"state"`
	InventoryPath   string                                `json:"inventoryPath, omitempty" gorm:"-"`
	HostVarsPath    string                                `json:"hostVarsPath, omitempty" gorm:"-"`
	GroupVarsPath   string                                `json:"groupVarsPath, omitempty" gorm:"-"`
	ClusterTopology map[uint64]NetworkClusterNodeTopology `json:"clusterTopology, omitempty" gorm:"-"`
	GroupVars       NetworkClusterGroupVars               `json:"groupVars, omitempty" gorm:"-"`
	Inventory       NetworkClusterInventory               `json:"inventory, omitempty" gorm:"-"`
	HostVars        []NetworkClusterHostVars              `json:"hostVars, omitempty" gorm:"-"`
	Phase           string                                `json:"phase, omitempty" gorm:"-"`
	UpdateRequest   *NetworkClusterUpdateRequest          `json:"updateRequest, omitempty" gorm:"-"`
}

type NetworkClusterNode struct {
	Id               uint64 `json:"-" gorm:"id"`
	NodeId           uint64 `json:"id,omitempty" validate:"required" gorm:"node_id"`
	NetworkClusterId uint64 `validate:"required" gorm:"column:network_cluster_id" json:"network_cluster_id,omitempty"`
	ControlIP        string `gorm:"control_ip" json:"controlIP"`
	NodeName         string `gorm:"-"`
	IsInvader        bool   `gorm:"-"`
	MgmtIPv4Address  string `gorm:"-"`
}

type NetworkClusterNodeTopology struct {
	NodeId        uint64
	NodeName      string
	NodeGroupId   uint64
	NodeGroupName string
	Links         map[string]NetworkClusterNodeLink `json:"links"`
}

type NetworkClusterNodeLink struct {
	IPv4Addresses       map[string]string `json:"ipv4_addresses"`
	IPv6Addresses       map[string]string `json:"ipv6_addresses"`
	InterfaceName       string            `json:"interface_name"`
	MacAddress          string            `json:"mac_address"`
	RemoteNodeId        uint64            `json:"remote_node_id"`
	RemoteNodeName      string            `json:"remote_node_name"`
	RemoteIPv4Addresses map[string]string `json:"remote_ipv4_ipv4_addresses"`
	RemoteIPv6Address   map[string]string `json:"remote_ipv6_ipv4_addresses"`
	RemoteInterfaceName string            `json:"remote_interface_name"`
	RemoteMacAddress    string            `json:"remote_mac_address"`
	RemoteGroupName     string            `json:"remote_group_name"`
	Ready               bool              `json:"ready,omitempty"`
	Tags                map[string]string `json:"tags"`
	CarrierStatus       string            `json:"carrier_status"`
	AdminStatus         string            `json:"admin_status"`
}

type NetworkClusterGroupVars struct {
	OspfArea    int
	ControlCIDR string
	IgwInUse    bool
	ControlIPs  map[string]string
}

type NetworkClusterInventory struct {
	SshUser               string
	SshUserPrivateKeyFile string
	Nodes                 []NetworkClusterNode
	Invaders              string
	Servers               string
}

type NetworkClusterHostVars struct {
	Nodeid            uint64
	NodeName          string
	ControlIP         string
	NetworkInterfaces map[string]string
	SrcToDst          map[string]string
	IsInvader         bool
	IsLeaf            bool
	IsSpine           bool
	AlterClusterRole  bool
	NodeAction        string
}

type NetworkClusterUpdateRequest struct {
	ExistingCluster    NetworkCluster
	NodesToBeAdded     []*NetworkClusterNode
	NodesToBeRemoved   []*NetworkClusterNode
	NodesToBeUpdated   []*NetworkClusterNode
	ClusterToBeUpdated NetworkCluster
	UpdateAction       string
}

var (
	NETWORK_INSTALLATION_SUCCESS_NOTIFICATION        = "Successfully created Network Cluster: [%+v]."

	NETWORK_INSTALLATION_INTERMEDIATE_NOTIFICATION_1       = "Create network failed for network cluster: [%+v] due to playbook error."
	NETWORK_INSTALLATION_INTERMEDIATE_NOTIFICATION_2       = "Reachability check failed for Network cluster: [%+v]. Rolling back."

	NETWORK_INSTALLATION_FAILED_NOTIFICATION_1             = "Rollback Successful for Network Cluster [%s]."
	NETWORK_INSTALLATION_FAILED_NOTIFICATION_2             = "Rollback failed for Network Cluster [%s]."

	NETWORK_UNINSTALLATION_SUCCESS_NOTIFICATION        = "Successfully deleted Network Cluster: [%+v]."

	NETWORK_CLUSTER_NOT_FOUND_ERROR = "Network cluster[%v] not found"
)

type NetworkVerifier struct {
	timeout   time.Duration
	events    EventsToCheck
	breakLoop chan bool
}

func (v *NetworkVerifier) GetTimeout() time.Duration {
	return v.timeout
}

func (v *NetworkVerifier) GetEventsToCheck() EventsToCheck {
	return v.events
}

func (v *NetworkVerifier) GetBreakLoopChan() chan bool {
	return v.breakLoop
}

func (p *PccClient) CreateNetworkCluster(request CreateNetworkClusterRequest) (id uint64, err error) {
	endpoint := fmt.Sprintf("pccserver/network/cluster")

	if err = p.Post(endpoint, &request, nil); err == nil {
		time.Sleep(time.Second * 5)
		cluster, errGet := p.GetNetworkCluster(request.Name)
		if errGet == nil {
			if cluster != nil {
				id = cluster.Id
			} else {
				err = fmt.Errorf("Failed to get cluster")
			}
		} else {
			err = fmt.Errorf("Failed to get cluster")
		}
	}
	return
}

func (config *NetworkConfiguration) SetNetworkClusterId(id uint64) {
	config.ClusterId = id
}

func (p *PccClient) GetNetworkCluster(clusterName string) (cluster *NetworkCluster, err error) {
	var clusterList []*NetworkCluster
	if clusterList, err = p.GetAllNetworkClusters(); err == nil {
		for _, cluster = range clusterList {
			if cluster.Name == clusterName {
				return
			}
		}
		if cluster == nil {
			err = fmt.Errorf(NETWORK_CLUSTER_NOT_FOUND_ERROR, clusterName)
		}
	}
	return
}

func (config *NetworkConfiguration) GetNetworkClusterName() string {
	return config.ClusterName
}

func (config *NetworkConfiguration) GetNetworkClusterId() uint64 {
	return config.ClusterId
}

func (p *PccClient) DeleteNetworkCluster(id uint64) (err error) {
	endpoint := fmt.Sprintf("pccserver/network/cluster/%v", id)
	err = p.Delete(endpoint, nil, nil)
	return
}

func (p *PccClient) GetAllNetworkClusters() (clusterList []*NetworkCluster, err error) {
	err = p.Get("pccserver/network/cluster", &clusterList)
	return
}

func (p *PccClient) ValidateNetworkConfig(config *NetworkConfiguration, identifier string) (err error) {
	var cluster *NetworkCluster
	if config.ClusterName != "" {
		cluster, err = p.GetNetworkCluster(config.ClusterName)
		if err == nil {
			config.ClusterId = cluster.Id
		} else if err.Error() == fmt.Sprintf(NETWORK_CLUSTER_NOT_FOUND_ERROR, config.ClusterName) {
			if run, ok := config.Tests[TestCreateNetworkCluster]; ok && run {
				err = nil
			}
		}
	} else {
		config.ClusterName = fmt.Sprintf("%s_%s", NETWORK_CLUSTER_NAME_PREFIX, identifier)
	}
	return
}

func (config *NetworkConfiguration) VerifyNetwork(startTime time.Time, action string, name string) (s EventStatus, err error) {
	s = config.PccClient.Verify(startTime, config.getNetworkVerifier(action, name))

	failed := !(strings.Contains(s.Msg, fmt.Sprintf(NETWORK_INSTALLATION_SUCCESS_NOTIFICATION, name)) ||
		strings.Contains(s.Msg, fmt.Sprintf(NETWORK_UNINSTALLATION_SUCCESS_NOTIFICATION, name)))
	if s.IsError || failed {
		err = fmt.Errorf("%v", s.Msg)
	}
	return
}

func (config *NetworkConfiguration) getNetworkVerifier(action string, name string) (v *NetworkVerifier) {
	switch action {
	case NETWORK_CLUSTER_INSTALL_EVENT:
		v = &NetworkVerifier{
			timeout: NETWORK_3_NODE_INSTALLATION_TIMEOUT,
			events: EventsToCheck{
				fmt.Sprintf(NETWORK_INSTALLATION_SUCCESS_NOTIFICATION, config.GetNetworkClusterName()):        true,
				fmt.Sprintf(NETWORK_INSTALLATION_INTERMEDIATE_NOTIFICATION_1, config.GetNetworkClusterName()):       true,
				fmt.Sprintf(NETWORK_INSTALLATION_INTERMEDIATE_NOTIFICATION_2, config.GetNetworkClusterName()):       true,
				fmt.Sprintf(NETWORK_INSTALLATION_FAILED_NOTIFICATION_1, config.GetNetworkClusterName()):       true,
				fmt.Sprintf(NETWORK_INSTALLATION_FAILED_NOTIFICATION_2, config.GetNetworkClusterName()):       true,
			},
			breakLoop: make(chan bool),
		}
	case NETWORK_CLUSTER_UNINSTALL_EVENT:
		v = &NetworkVerifier{
			timeout: NETWORK_3_NODE_UNINSTALLATION_TIMEOUT,
			events: EventsToCheck{
				fmt.Sprintf(NETWORK_UNINSTALLATION_SUCCESS_NOTIFICATION, config.GetNetworkClusterName()): true,
			},
			breakLoop: make(chan bool),
		}
	}
	return
}
