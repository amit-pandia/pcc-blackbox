package pcc

import (
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/platinasystems/tiles/pccserver/models"
	"github.com/platinasystems/tiles/pccserver/storage/ceph"
)

type cephPoolTypes string
type cephPoolMeta struct {
	name string
	id   uint64
}

const (
	CEPH_CLUSTER_NAME_PREFIX           = "cephtest"
	CEPH_3_NODE_INSTALLATION_TIMEOUT   = 1000
	CEPH_3_NODE_UNINSTALLATION_TIMEOUT = 300

	CEPH_POOL_CREATION_TIMEOUT = 300
	CEPH_POOL_DELETION_TIMEOUT = 300

	CEPH_FS_CREATION_TIMEOUT = 300
	CEPH_FS_DELETION_TIMEOUT = 300

	CEPH_POOL_FOR_K8S_TEST    cephPoolTypes = "k8sTest"
	CEPH_POOL_FOR_RBD_TEST    cephPoolTypes = "rbdTest"
	CEPH_POOL_FOR_CEPHFS_TEST cephPoolTypes = "cephFSTest"

	CEPH_POOL_K8S_1    = "k8spool1"
	CEPH_POOL_K8S_2    = "k8spool2"
	CEPH_POOL_RBD_1    = "rbdpool1"
	CEPH_POOL_RBD_2    = "rbdpool2"
	CEPH_POOL_METADATA = "metadatapool"
	CEPH_POOL_DEFAULT  = "defaultpool"
	CEPH_POOL_DATA_1   = "datapool1"
	CEPH_POOL_DATA_2   = "datapool2"

	CEPH_FS_NAME = CEPH_CLUSTER_NAME_PREFIX + "-fs"

	// For Verification
	CEPH_CLUSTER_INSTALL_EVENT   = "cephClusterInstall"
	CEPH_CLUSTER_UNINSTALL_EVENT = "cephClusterUninstall"
	CEPH_POOL_CREATE_EVENT       = "cephPoolCreate"
	CEPH_POOL_DELETE_EVENT       = "cephPoolDelete"
	CEPH_FS_CREATE_EVENT         = "cephFSCreate"
	CEPH_FS_DELETE_EVENT         = "cephFSDelete"
)

var (
	CEPH_INSTALLATION_SUCCESS_NOTIFICATION        = "Ceph cluster has been deployed"
	CEPH_INSTALLATION_FAILED_NOTIFICATION_1       = "Ceph cluster [%+v] installation failed"
	CEPH_INSTALLATION_FAILED_NOTIFICATION_2       = "Unable to create ceph cluster "
	CEPH_INSTALLATION_FAILED_NOTIFICATION_3       = "Unable to store ceph cluster "
	CEPH_INSTALLATION_FAILED_NOTIFICATION_4       = "Unable to deploy ceph  cluster [%s] as there are no OSD nodes available"
	CEPH_INSTALLATION_INTERMEDIATE_NOTIFICATION_1 = "Ceph cluster installation begins"
	CEPH_INSTALLATION_INTERMEDIATE_NOTIFICATION_2 = "Successfully created network for ceph cluster"
	CEPH_INSTALLATION_INTERMEDIATE_NOTIFICATION_3 = "Create network failed for ceph cluster"
	CEPH_INSTALLATION_INTERMEDIATE_NOTIFICATION_4 = "Reachability check failed for ceph cluster"
	CEPH_INSTALLATION_INTERMEDIATE_NOTIFICATION_5 = "]. Cluster:[%+v]"
	CEPH_INSTALLATION_INTERMEDIATE_NOTIFICATION_6 = "Provisioning unused drives"
	CEPH_INSTALLATION_INTERMEDIATE_NOTIFICATION_7 = "Creating network for ceph cluster"
	CEPH_INSTALLATION_INTERMEDIATE_NOTIFICATION_8 = "Drive provisionig is finished"

	CEPH_UNINSTALLATION_SUCCESS_NOTIFICATION        = "Successfully deleted network for ceph cluster"
	CEPH_UNINSTALLATION_FAILED_NOTIFICATION_1       = "Ceph cluster [%+v] uninstallation failed"
	CEPH_UNINSTALLATION_FAILED_NOTIFICATION_2       = "Unable to remove ceph cluster [%+v]"
	CEPH_UNINSTALLATION_INTERMEDIATE_NOTIFICATION_1 = "Ceph un-installation started"
	CEPH_UNINSTALLATION_INTERMEDIATE_NOTIFICATION_2 = "Deleting network for ceph cluster"
	CEPH_UNINSTALLATION_INTERMEDIATE_NOTIFICATION_3 = "ceph cluster has been removed from DB"
	CEPH_UNINSTALLATION_INTERMEDIATE_NOTIFICATION_4 = "Ceph cluster has been uninstalled but unable to remove it from database"

	CEPH_POOL_CREATION_SUCCESS_NOTIFICATION        = fmt.Sprintf("Pool : [%%s] has been [%s] for cluster [%%s]", ceph.CEPH_POOL_DEPLOY_STATUS_COMPLETED)
	CEPH_POOL_CREATION_FAILED_NOTIFICATION         = fmt.Sprintf("Pool : [%%s] has been [%s] for cluster [%%s]", ceph.CEPH_POOL_DEPLOY_STATUS_FAILED)
	CEPH_POOL_CREATION_INTERMEDIATE_NOTIFICATION_1 = "Creating Pool : [%s] for cluster [%s]"
	CEPH_POOL_DELETION_SUCCESS_NOTIFICATION        = "Pool [%s] has been removed from DB"
	CEPH_POOL_DELETION_FAILED_NOTIFICATION         = "Unable to remove pool [%s]"
	CEPH_POOL_DELETION_INTERMEDIATE_NOTIFICATION_1 = "Removing Pool : [%s] from cluster [%s]"

	CEPH_FS_CREATION_SUCCESS_NOTIFICATION        = fmt.Sprintf("FS : [%%s] has been [%s] for cluster [%%s]", ceph.CEPH_FS_DEPLOY_STATUS_COMPLETED)
	CEPH_FS_CREATION_INTERMEDIATE_NOTIFICATION_1 = "Creating FS : [%s] for cluster [%s]"
	CEPH_FS_CREATION_FAILED_NOTIFICATION_1       = fmt.Sprintf("FS : [%%s] has been [%s] for cluster [%%s]", ceph.CEPH_FS_DEPLOY_STATUS_FAILED)
	CEPH_FS_CREATION_FAILED_NOTIFICATION_2       = "Unable to Create FS : [%s] for cluster [%s]"
	CEPH_FS_DELETION_SUCCESS_NOTIFICATION        = "Ceph FS [%s] has been removed from DB"
	CEPH_FS_DELETION_FAILED_NOTIFICATION_1       = "Unable to remove ceph FS [%s]"
	CEPH_FS_DELETION_FAILED_NOTIFICATION_2       = "Unable to uninstall FS : [%s]"
	CEPH_FS_DELETION_INTERMEDIATE_NOTIFICATION_1 = "Removing FS : [%s]"

	CephPools = map[cephPoolTypes]map[string]uint64{
		CEPH_POOL_FOR_K8S_TEST:    map[string]uint64{CEPH_POOL_K8S_1: 0, CEPH_POOL_K8S_2: 0},
		CEPH_POOL_FOR_RBD_TEST:    map[string]uint64{CEPH_POOL_RBD_1: 0, CEPH_POOL_RBD_2: 0},
		CEPH_POOL_FOR_CEPHFS_TEST: map[string]uint64{CEPH_POOL_METADATA: 0, CEPH_POOL_DEFAULT: 0, CEPH_POOL_DATA_1: 0, CEPH_POOL_DATA_2: 0},
	}

	CEPH_CLUSTER_NOT_FOUND_ERROR = "Ceph cluster[%v] not found"
)

const (
	TestCreateCephCluster = "createCluster"
	TestDeleteCephCluster = "deleteCluster"
	TestCreateCephPools   = "createPools"
	TestDeleteCephPools   = "deletePools"
	TestCreateCephFS      = "createFS"
	TestDeleteCephFS      = "deleteFS"
)

type CephConfiguration struct {
	ClusterName      string          `json:"clusterName"`
	ClusterId        uint64          `json:"clusterId"`
	HighAvailability bool            `json:"highAvailability"`
	NumberOfNodes    int             `json:"numberOfNodes"`
	Tests            map[string]bool `json:"tests"`
	PccClient        *PccClient
}

func (config *CephConfiguration) GetCephClusterName() string {
	return config.ClusterName
}

func (config *CephConfiguration) SetCephClusterName(name string) {
	config.ClusterName = name
}

func (config *CephConfiguration) GetCephClusterId() uint64 {
	return config.ClusterId
}

func (config *CephConfiguration) SetCephClusterId(id uint64) {
	config.ClusterId = id
}

func (p *PccClient) ValidateCephConfig(config *CephConfiguration, identifier string) (err error) {
	var cluster *models.CephCluster
	if config.ClusterName != "" {
		cluster, err = p.GetCephCluster(config.ClusterName)
		if err == nil {
			config.ClusterId = cluster.Id
		} else if err.Error() == fmt.Sprintf(CEPH_CLUSTER_NOT_FOUND_ERROR, config.ClusterName) {
			if run, ok := config.Tests[TestCreateCephCluster]; ok && run {
				err = nil
			}
		}
	} else {
		config.ClusterName = fmt.Sprintf("%s_%s", CEPH_CLUSTER_NAME_PREFIX, identifier)
	}
	return
}

type CreateCephClusterRequest struct {
	Name        string         `json:"name"`
	Nodes       []CephNodes    `json:"nodes"`
	Version     string         `json:"version"`
	Tags        pq.StringArray `json:"tags"`
	IgwPolicy   string         `json:"igwPolicy"`
	ControlCIDR string         `json:"controlCIDR"`

	models.CephClusterConfig `json:"config"`
}

type CephNodes struct {
	ID uint64
}

func (p *PccClient) CreateCephCluster(request CreateCephClusterRequest) (id uint64, err error) {
	endpoint := fmt.Sprintf("pccserver/storage/ceph/cluster")

	if err = p.Post(endpoint, &request, nil); err == nil {
		time.Sleep(time.Second * 5)
		cluster, errGet := p.GetCephCluster(request.Name)
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

func (p *PccClient) GetCephCluster(clusterName string) (cluster *models.CephCluster, err error) {
	var clusterList []*models.CephCluster
	if clusterList, err = p.GetAllCephClusters(); err == nil {
		for _, cluster = range clusterList {
			if cluster.Name == clusterName {
				return
			}
		}
		if cluster == nil {
			err = fmt.Errorf(CEPH_CLUSTER_NOT_FOUND_ERROR, clusterName)
		}
	}
	return
}

func (p *PccClient) GetAllCephClusters() (clusterList []*models.CephCluster, err error) {
	err = p.Get("pccserver/storage/ceph/cluster", &clusterList)
	return
}

func (p *PccClient) DeleteCephCluster(id uint64) (err error) {
	endpoint := fmt.Sprintf("pccserver/storage/ceph/cluster/%v", id)
	err = p.Delete(endpoint, nil, nil)
	return
}

type CreateCephPoolRequest struct {
	Name          string         `json:"name,omitempty"`
	PoolType      string         `json:"pool_type,omitempty"`
	Size          uint64         `json:"size,omitempty"`
	Quota         string         `json:"quota,omitempty"`
	QuotaUnit     string         `json:"quota_unit,omitempty"`
	Tags          pq.StringArray `json:"tags,omitempty"`
	CephClusterId uint64         `json:"ceph_cluster_id,omitempty"`
	FailureDomain int            `json:"failure_domain,omitempty"`
}

func (p *PccClient) GetCephPool(name string, cephClusterId uint64) (cephPool *models.CephPool, err error) {
	var pools []*models.CephPool
	pools, err = p.GetAllCephPools(cephClusterId)
	if err == nil {
		if len(pools) > 0 {
			for _, cephPool = range pools {
				if cephPool.Name != name {
					continue
				}
				return
			}
		} else {
			err = fmt.Errorf("No ceph pool found")
		}
	}
	return
}

func (p *PccClient) GetAllCephPools(id uint64) (cephPools []*models.CephPool, err error) {
	if id != 0 {
		endpoint := fmt.Sprintf("pccserver/storage/ceph/cluster/%v/pools", id)
		err = p.Get(endpoint, &cephPools)
	} else {
		err = fmt.Errorf("ERROR: No Ceph Cluster found")
	}
	return
}

func (p *PccClient) GetAvailableCephPools() (cephPools []*models.CephPool, err error) {
	return
}

func (p *PccClient) CreateCephPool(request CreateCephPoolRequest) (id uint64, err error) {
	endpoint := fmt.Sprintf("pccserver/storage/ceph/pool")
	if err = p.Post(endpoint, &request, nil); err == nil {
		time.Sleep(time.Second * 5)
		cephPool, errGet := p.GetCephPool(request.Name, request.CephClusterId)
		if errGet == nil {
			if cephPool != nil {
				id = cephPool.Id
			} else {
				err = fmt.Errorf("Failed to get ceph pool")
			}
		}
	}
	return
}

func (p *PccClient) DeleteCephPool(id uint64) (err error) {
	endpoint := fmt.Sprintf("pccserver/storage/ceph/pool/%v", id)
	err = p.Delete(endpoint, nil, nil)
	return
}

type CreateCephFSRequest struct {
	Name          string             `json:"name,omitempty"`
	Tags          pq.StringArray     `json:"tags"`
	CephClusterId uint64             `json:"ceph_cluster_id,omitempty"`
	MetadataPool  *models.CephPool   `json:"metadata_pool,omitempty"`
	DefaultPool   *models.CephPool   `json:"default_pool,omitempty"`
	DataPools     []*models.CephPool `json:"data_pools,omitempty"`
	MaxMDS        int                `json:"max_mds,omitempty"`
}

func (p *PccClient) GetCephFS(name string, cephClusterId uint64) (cephFS *models.CephFS, err error) {
	var cephFSList []*models.CephFS
	cephFSList, err = p.GetAllCephFS(cephClusterId)
	if err == nil {
		if len(cephFSList) > 0 {
			for _, cephFS = range cephFSList {
				if cephFS.Name != name {
					continue
				}
				return
			}
		} else {
			err = fmt.Errorf("No ceph pool found")
		}
	}
	return
}

func (p *PccClient) GetAllCephFS(id uint64) (cephFSList []*models.CephFS, err error) {
	if id != 0 {
		endpoint := fmt.Sprintf("pccserver/storage/ceph/cluster/%v/fs", id)
		err = p.Get(endpoint, &cephFSList)
	} else {
		err = fmt.Errorf("ERROR: No Ceph Cluster found")
	}
	return
}

func (p *PccClient) CreateCephFS(request CreateCephFSRequest) (id uint64, err error) {
	if err = p.Post("pccserver/storage/ceph/fs", &request, nil); err == nil {
		time.Sleep(time.Second * 5)
		cephFS, errGet := p.GetCephFS(request.Name, request.CephClusterId)
		if errGet == nil {
			if cephFS != nil {
				id = cephFS.Id
			} else {
				err = fmt.Errorf("Failed to get ceph fs")
			}
		}
	}
	return
}

func (p *PccClient) DeleteCephFS(id uint64) (err error) {
	endpoint := fmt.Sprintf("pccserver/storage/ceph/fs/%v", id)
	err = p.Delete(endpoint, nil, nil)
	return
}

func (config *CephConfiguration) VerifyCeph(startTime time.Time, action string, name string) (s EventStatus, err error) {
	s = config.PccClient.Verify(startTime, config.getCephVerifier(action, name))

	failed := !(strings.Contains(s.Msg, CEPH_INSTALLATION_SUCCESS_NOTIFICATION) ||
		strings.Contains(s.Msg, CEPH_UNINSTALLATION_SUCCESS_NOTIFICATION) ||
		strings.Contains(s.Msg, fmt.Sprintf(CEPH_POOL_CREATION_SUCCESS_NOTIFICATION, name, config.ClusterName)) ||
		strings.Contains(s.Msg, fmt.Sprintf(CEPH_POOL_DELETION_SUCCESS_NOTIFICATION, name)) ||
		strings.Contains(s.Msg, fmt.Sprintf(CEPH_FS_CREATION_SUCCESS_NOTIFICATION, name, config.ClusterName)) ||
		strings.Contains(s.Msg, fmt.Sprintf(CEPH_FS_DELETION_SUCCESS_NOTIFICATION, name)))
	if s.IsError || failed {
		err = fmt.Errorf("%v", s.Msg)
	}
	return
}

type CephVerifier struct {
	timeout   time.Duration
	events    EventsToCheck
	breakLoop chan bool
}

func (v *CephVerifier) GetTimeout() time.Duration {
	return v.timeout
}

func (v *CephVerifier) GetEventsToCheck() EventsToCheck {
	return v.events
}

func (v *CephVerifier) GetBreakLoopChan() chan bool {
	return v.breakLoop
}

func (config *CephConfiguration) getCephVerifier(action string, name string) (v *CephVerifier) {
	switch action {
	case CEPH_CLUSTER_INSTALL_EVENT:
		v = &CephVerifier{
			timeout: CEPH_3_NODE_INSTALLATION_TIMEOUT,
			events: EventsToCheck{
				CEPH_INSTALLATION_SUCCESS_NOTIFICATION:                                                  true,
				fmt.Sprintf(CEPH_INSTALLATION_FAILED_NOTIFICATION_1, config.GetCephClusterName()):       true,
				CEPH_INSTALLATION_FAILED_NOTIFICATION_2:                                                 true,
				CEPH_INSTALLATION_FAILED_NOTIFICATION_3:                                                 true,
				fmt.Sprintf(CEPH_INSTALLATION_FAILED_NOTIFICATION_4, config.GetCephClusterName()):       true,
				CEPH_INSTALLATION_INTERMEDIATE_NOTIFICATION_1:                                           false,
				CEPH_INSTALLATION_INTERMEDIATE_NOTIFICATION_2:                                           false,
				CEPH_INSTALLATION_INTERMEDIATE_NOTIFICATION_3:                                           false,
				CEPH_INSTALLATION_INTERMEDIATE_NOTIFICATION_4:                                           false,
				fmt.Sprintf(CEPH_INSTALLATION_INTERMEDIATE_NOTIFICATION_5, config.GetCephClusterName()): false,
				CEPH_INSTALLATION_INTERMEDIATE_NOTIFICATION_6:                                           false,
				CEPH_INSTALLATION_INTERMEDIATE_NOTIFICATION_7:                                           false,
				CEPH_INSTALLATION_INTERMEDIATE_NOTIFICATION_8:                                           false,
			},
			breakLoop: make(chan bool),
		}
	case CEPH_CLUSTER_UNINSTALL_EVENT:
		v = &CephVerifier{
			timeout: CEPH_3_NODE_UNINSTALLATION_TIMEOUT,
			events: EventsToCheck{
				CEPH_UNINSTALLATION_SUCCESS_NOTIFICATION:                                            true,
				fmt.Sprintf(CEPH_UNINSTALLATION_FAILED_NOTIFICATION_1, config.GetCephClusterName()): true,
				fmt.Sprintf(CEPH_UNINSTALLATION_FAILED_NOTIFICATION_2, config.GetCephClusterName()): true,
				CEPH_UNINSTALLATION_INTERMEDIATE_NOTIFICATION_1:                                     false,
				CEPH_UNINSTALLATION_INTERMEDIATE_NOTIFICATION_2:                                     false,
				CEPH_UNINSTALLATION_INTERMEDIATE_NOTIFICATION_3:                                     false,
				CEPH_UNINSTALLATION_INTERMEDIATE_NOTIFICATION_4:                                     false,
			},
			breakLoop: make(chan bool),
		}
	case CEPH_POOL_CREATE_EVENT:
		v = &CephVerifier{
			timeout: CEPH_POOL_CREATION_TIMEOUT,
			events: EventsToCheck{
				fmt.Sprintf(CEPH_POOL_CREATION_SUCCESS_NOTIFICATION, name, config.GetCephClusterName()):        true,
				fmt.Sprintf(CEPH_POOL_CREATION_FAILED_NOTIFICATION, name, config.GetCephClusterName()):         true,
				fmt.Sprintf(CEPH_POOL_CREATION_INTERMEDIATE_NOTIFICATION_1, name, config.GetCephClusterName()): false,
			},
			breakLoop: make(chan bool),
		}
	case CEPH_POOL_DELETE_EVENT:
		v = &CephVerifier{
			timeout: CEPH_POOL_DELETION_TIMEOUT,
			events: EventsToCheck{
				fmt.Sprintf(CEPH_POOL_DELETION_SUCCESS_NOTIFICATION, name):                                     true,
				fmt.Sprintf(CEPH_POOL_DELETION_FAILED_NOTIFICATION, name):                                      true,
				fmt.Sprintf(CEPH_POOL_DELETION_INTERMEDIATE_NOTIFICATION_1, name, config.GetCephClusterName()): false,
			},
			breakLoop: make(chan bool),
		}
	case CEPH_FS_CREATE_EVENT:
		v = &CephVerifier{
			timeout: CEPH_FS_CREATION_TIMEOUT,
			events: EventsToCheck{
				fmt.Sprintf(CEPH_FS_CREATION_SUCCESS_NOTIFICATION, name, config.GetCephClusterName()):        true,
				fmt.Sprintf(CEPH_FS_CREATION_FAILED_NOTIFICATION_1, name, config.GetCephClusterName()):       true,
				fmt.Sprintf(CEPH_FS_CREATION_FAILED_NOTIFICATION_2, name, config.GetCephClusterName()):       true,
				fmt.Sprintf(CEPH_FS_CREATION_INTERMEDIATE_NOTIFICATION_1, name, config.GetCephClusterName()): false,
			},
			breakLoop: make(chan bool),
		}
	case CEPH_FS_DELETE_EVENT:
		v = &CephVerifier{
			timeout: CEPH_FS_DELETION_TIMEOUT,
			events: EventsToCheck{
				fmt.Sprintf(CEPH_FS_DELETION_SUCCESS_NOTIFICATION, name):        true,
				fmt.Sprintf(CEPH_FS_DELETION_FAILED_NOTIFICATION_1, name):       true,
				fmt.Sprintf(CEPH_FS_DELETION_FAILED_NOTIFICATION_2, name):       true,
				fmt.Sprintf(CEPH_FS_DELETION_INTERMEDIATE_NOTIFICATION_1, name): false,
			},
			breakLoop: make(chan bool),
		}
	}
	return
}
