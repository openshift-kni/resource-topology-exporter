package podrescli

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"time"

	"google.golang.org/grpc"

	podresourcesapi "k8s.io/kubelet/pkg/apis/podresources/v1"
	"k8s.io/kubernetes/pkg/kubelet/apis/podresources"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
)

const (
	defaultPodResourcesTimeout = 10 * time.Second
	defaultPodResourcesMaxSize = 1024 * 1024 * 16 // 16 Mb
	// obtained these values from node e2e tests : https://github.com/kubernetes/kubernetes/blob/82baa26905c94398a0d19e1b1ecf54eb8acb6029/test/e2e_node/util.go#L70
)

type ContainerIdent struct {
	Namespace     string
	PodName       string
	ContainerName string
}

func (ci *ContainerIdent) String() string {
	if ci == nil {
		return ""
	}
	return fmt.Sprintf("%s/%s/%s", ci.Namespace, ci.PodName, ci.ContainerName)
}

type PodResourcesFilter interface {
	FilterListResponse(resp *podresourcesapi.ListPodResourcesResponse) *podresourcesapi.ListPodResourcesResponse
	FilterAllocatableResponse(resp *podresourcesapi.AllocatableResourcesResponse) *podresourcesapi.AllocatableResourcesResponse
}

type filteringClient struct {
	debug          bool
	cli            podresourcesapi.PodResourcesListerClient
	refCnt         *ContainerIdent
	sharedPoolCPUs cpuset.CPUSet // used only for logging
}

func (fc *filteringClient) FilterListResponse(resp *podresourcesapi.ListPodResourcesResponse) *podresourcesapi.ListPodResourcesResponse {
	sharedPoolCPUs := findSharedPoolCPUsInListResponse(fc.refCnt, resp.GetPodResources())
	if !fc.sharedPoolCPUs.Equals(sharedPoolCPUs) {
		log.Printf("detected shared pool change: %v -> %v", fc.sharedPoolCPUs, sharedPoolCPUs)
		fc.sharedPoolCPUs = sharedPoolCPUs
	}
	for _, podRes := range resp.GetPodResources() {
		for _, cntRes := range podRes.GetContainers() {
			cpuIds := removeCPUs(cntRes.CpuIds, sharedPoolCPUs)
			if fc.debug && !reflect.DeepEqual(cpuIds, cntRes.CpuIds) {
				curCpus := cpuset.NewCPUSetInt64(cntRes.CpuIds...)
				newCpus := cpuset.NewCPUSetInt64(cpuIds...)
				log.Printf("performed pool change for %s/%s: %s -> %s", podRes.Name, cntRes.Name, curCpus, newCpus)
			}
			cntRes.CpuIds = cpuIds
		}
	}
	return resp
}

func (fc *filteringClient) FilterAllocatableResponse(resp *podresourcesapi.AllocatableResourcesResponse) *podresourcesapi.AllocatableResourcesResponse {
	return resp // nothing to do here
}

func (fc *filteringClient) List(ctx context.Context, in *podresourcesapi.ListPodResourcesRequest, opts ...grpc.CallOption) (*podresourcesapi.ListPodResourcesResponse, error) {
	resp, err := fc.cli.List(ctx, in, opts...)
	if err != nil {
		return resp, err
	}
	return fc.FilterListResponse(resp), nil
}

func (fc *filteringClient) GetAllocatableResources(ctx context.Context, in *podresourcesapi.AllocatableResourcesRequest, opts ...grpc.CallOption) (*podresourcesapi.AllocatableResourcesResponse, error) {
	resp, err := fc.cli.GetAllocatableResources(ctx, in, opts...)
	if err != nil {
		return resp, err
	}
	return fc.FilterAllocatableResponse(resp), nil
}

func NewK8SClient(socketPath string) (podresourcesapi.PodResourcesListerClient, error) {
	cli, _, err := podresources.GetV1Client(socketPath, defaultPodResourcesTimeout, defaultPodResourcesMaxSize)
	return cli, err
}

func NewFilteringClient(socketPath string, debug bool, referenceContainer *ContainerIdent) (podresourcesapi.PodResourcesListerClient, error) {
	cli, err := NewK8SClient(socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create podresource client: %v", err)
	}
	log.Printf("connected to %q", socketPath)
	return NewFilteringClientFromLister(cli, debug, referenceContainer)
}

func NewFilteringClientFromLister(cli podresourcesapi.PodResourcesListerClient, debug bool, referenceContainer *ContainerIdent) (podresourcesapi.PodResourcesListerClient, error) {
	return &filteringClient{
		debug:  debug,
		cli:    cli,
		refCnt: referenceContainer,
	}, nil
}

func findSharedPoolCPUsInListResponse(refCnt *ContainerIdent, podResources []*podresourcesapi.PodResources) cpuset.CPUSet {
	if refCnt == nil || podResources == nil {
		return cpuset.CPUSet{}
	}
	for _, podRes := range podResources {
		if podRes.Namespace != refCnt.Namespace {
			continue
		}
		if podRes.Name != refCnt.PodName {
			continue
		}
		for _, cntRes := range podRes.GetContainers() {
			if cntRes.Name != refCnt.ContainerName {
				continue
			}
			return cpuset.NewCPUSetInt64(cntRes.CpuIds...)
		}
	}
	return cpuset.CPUSet{}
}

func removeCPUs(cpuIDs []int64, toRemove cpuset.CPUSet) []int64 {
	cs := cpuset.NewCPUSetInt64(cpuIDs...)
	res := cs.Difference(toRemove)
	return res.ToSliceInt64()
}
