// Copyright (c) 2018 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cluster

import (
	"fmt"

	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/config"
)

type (
	// Metadata provides information about clusters
	Metadata struct {
		// failoverVersionIncrement is the increment of each cluster's version when failover happen
		failoverVersionIncrement int64
		// primaryClusterName is the name of the primary cluster, only the primary cluster can register / update domain
		// all clusters can do domain failover
		primaryClusterName string
		// currentClusterName is the name of the current cluster
		currentClusterName string
		// allClusters contains all cluster info
		allClusters map[string]config.ClusterInformation
		// enabledClusters contains enabled info
		enabledClusters map[string]config.ClusterInformation
		// remoteClusters contains enabled and remote info
		remoteClusters map[string]config.ClusterInformation
		// versionToClusterName contains all initial version -> corresponding cluster name
		versionToClusterName map[int64]string
	}
)

// NewMetadata create a new instance of Metadata
func NewMetadata(
	failoverVersionIncrement int64,
	primaryClusterName string,
	currentClusterName string,
	clusterGroup map[string]config.ClusterInformation,
) Metadata {
	versionToClusterName := make(map[int64]string)
	for clusterName, info := range clusterGroup {
		versionToClusterName[info.InitialFailoverVersion] = clusterName
	}

	// We never use disable clusters, filter them out on start
	enabledClusters := map[string]config.ClusterInformation{}
	for cluster, info := range clusterGroup {
		if info.Enabled {
			enabledClusters[cluster] = info
		}
	}

	// Precompute remote clusters, they are used in multiple places
	remoteClusters := map[string]config.ClusterInformation{}
	for cluster, info := range enabledClusters {
		if cluster != currentClusterName {
			remoteClusters[cluster] = info
		}
	}

	return Metadata{
		failoverVersionIncrement: failoverVersionIncrement,
		primaryClusterName:       primaryClusterName,
		currentClusterName:       currentClusterName,
		allClusters:              clusterGroup,
		enabledClusters:          enabledClusters,
		remoteClusters:           remoteClusters,
		versionToClusterName:     versionToClusterName,
	}
}

// GetNextFailoverVersion return the next failover version based on input
func (m Metadata) GetNextFailoverVersion(cluster string, currentFailoverVersion int64) int64 {
	info, ok := m.allClusters[cluster]
	if !ok {
		panic(fmt.Sprintf(
			"Unknown cluster name: %v with given cluster initial failover version map: %v.",
			cluster,
			m.allClusters,
		))
	}
	failoverVersion := currentFailoverVersion/m.failoverVersionIncrement*m.failoverVersionIncrement + info.InitialFailoverVersion
	if failoverVersion < currentFailoverVersion {
		return failoverVersion + m.failoverVersionIncrement
	}
	return failoverVersion
}

// IsVersionFromSameCluster return true if 2 version are used for the same cluster
func (m Metadata) IsVersionFromSameCluster(version1 int64, version2 int64) bool {
	return (version1-version2)%m.failoverVersionIncrement == 0
}

func (m Metadata) IsPrimaryCluster() bool {
	return m.primaryClusterName == m.currentClusterName
}

// GetCurrentClusterName return the current cluster name
func (m Metadata) GetCurrentClusterName() string {
	return m.currentClusterName
}

// GetAllClusterInfo return all cluster info
func (m Metadata) GetAllClusterInfo() map[string]config.ClusterInformation {
	return m.allClusters
}

// GetEnabledClusterInfo return enabled cluster info
func (m Metadata) GetEnabledClusterInfo() map[string]config.ClusterInformation {
	return m.enabledClusters
}

// GetRemoteClusterInfo return enabled AND remote cluster info
func (m Metadata) GetRemoteClusterInfo() map[string]config.ClusterInformation {
	return m.remoteClusters
}

// ClusterNameForFailoverVersion return the corresponding cluster name for a given failover version
func (m Metadata) ClusterNameForFailoverVersion(failoverVersion int64) string {
	if failoverVersion == common.EmptyVersion {
		return m.currentClusterName
	}

	initialFailoverVersion := failoverVersion % m.failoverVersionIncrement
	clusterName, ok := m.versionToClusterName[initialFailoverVersion]
	if !ok {
		panic(fmt.Sprintf(
			"Unknown initial failover version %v with given cluster initial failover version map: %v and failover version increment %v.",
			initialFailoverVersion,
			m.allClusters,
			m.failoverVersionIncrement,
		))
	}
	return clusterName
}
