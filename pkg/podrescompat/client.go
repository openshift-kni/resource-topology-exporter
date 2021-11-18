/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package podrescompat

import (
	"context"
	"log"

	"google.golang.org/grpc"

	podresourcesapi "k8s.io/kubelet/pkg/apis/podresources/v1"

	"github.com/openshift-kni/resource-topology-exporter/pkg/sysinfo"
)

type sysinfoClient struct {
	sysConf sysinfo.Config
	cli     podresourcesapi.PodResourcesListerClient
}

func NewSysinfoClientFromLister(cli podresourcesapi.PodResourcesListerClient, sysConf sysinfo.Config) podresourcesapi.PodResourcesListerClient {
	return &sysinfoClient{
		cli:     cli,
		sysConf: sysConf,
	}
}

func (sc *sysinfoClient) List(ctx context.Context, in *podresourcesapi.ListPodResourcesRequest, opts ...grpc.CallOption) (*podresourcesapi.ListPodResourcesResponse, error) {
	return sc.cli.List(ctx, in, opts...)
}

func (sc *sysinfoClient) GetAllocatableResources(ctx context.Context, in *podresourcesapi.AllocatableResourcesRequest, opts ...grpc.CallOption) (*podresourcesapi.AllocatableResourcesResponse, error) {
	resp, err := sc.cli.GetAllocatableResources(ctx, in, opts...)
	if err != nil {
		log.Printf("podresourcesapi GetAllocatableResources() failed with %v - using sysinfo", err)
		sysResp, sysErr := sc.makeAllocatableResourcesResponse()
		if sysErr != nil {
			log.Printf("sysinfo makeAllocatableResourcesResponse failed with %v", sysErr)
			return resp, err
		}
		return sysResp, nil
	}
	return resp, nil
}

func (sc *sysinfoClient) makeAllocatableResourcesResponse() (*podresourcesapi.AllocatableResourcesResponse, error) {
	sysInfo, err := sysinfo.NewSysinfo(sc.sysConf)
	if err != nil {
		return nil, err
	}
	return MakeAllocatableResourcesResponseFromSysInfo(sysInfo), nil
}

func MakeAllocatableResourcesResponseFromSysInfo(sysInfo sysinfo.SysInfo) *podresourcesapi.AllocatableResourcesResponse {
	resp := podresourcesapi.AllocatableResourcesResponse{
		CpuIds: sysInfo.CPUs.ToSliceInt64(),
	}
	for resourceName, resourceDevices := range sysInfo.Resources {
		for numaCellID, numaDevices := range resourceDevices {
			cntDevs := podresourcesapi.ContainerDevices{
				ResourceName: resourceName,
				DeviceIds:    numaDevices,
				Topology: &podresourcesapi.TopologyInfo{
					Nodes: []*podresourcesapi.NUMANode{
						{ID: int64(numaCellID)},
					},
				},
			}
			resp.Devices = append(resp.Devices, &cntDevs)
		}
	}
	return &resp
}
