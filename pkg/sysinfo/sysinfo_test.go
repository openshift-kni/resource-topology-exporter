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

package sysinfo

import (
	"reflect"
	"testing"

	"github.com/jaypipes/ghw/pkg/pci"
	"github.com/jaypipes/ghw/pkg/topology"
	"github.com/jaypipes/pcidb"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
)

func TestGetCPUResources(t *testing.T) {
	var testCases = []struct {
		name     string
		online   string
		reserved string
		expected string
	}{
		{"no reserved", "0-15", "", "0-15"},
		{"reserved", "0-15", "0,8", "1-7,9-15"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := GetCPUResources(testCase.reserved, func() (cpuset.CPUSet, error) { return cpuset.Parse(testCase.online) })
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			expectedCPUs := cpuset.MustParse(testCase.expected)
			if !got.Equals(expectedCPUs) {
				t.Errorf("got %s, want %s", got, expectedCPUs)
			}
		})
	}
}

func TestGetPCIResources(t *testing.T) {
	var testCases = []struct {
		name     string
		devs     []*pci.Device
		resMap   map[string]string
		expected map[string]PerNUMADevices
	}{
		{"no devs", nil, map[string]string{"8086:1520": "intel_nics"}, map[string]PerNUMADevices{}},
		{
			"devs no numa",
			[]*pci.Device{
				fakePCIDevice("8086", "1520", "0000:00:02.0", -1),
				fakePCIDevice("8086", "1520", "0000:00:02.1", -1),
			},
			map[string]string{"8086:1520": "intel_nics"},
			map[string]PerNUMADevices{
				"intel_nics": map[int][]string{
					-1: []string{"0000:00:02.0", "0000:00:02.1"},
				},
			},
		},
		{
			"devs single numa",
			[]*pci.Device{
				fakePCIDevice("8086", "1520", "0000:00:02.0", 0),
				fakePCIDevice("8086", "1520", "0000:00:02.1", 0),
			},
			map[string]string{"8086:1520": "intel_nics"},
			map[string]PerNUMADevices{
				"intel_nics": map[int][]string{
					0: []string{"0000:00:02.0", "0000:00:02.1"},
				},
			},
		},
		{
			"devs multi numa",
			[]*pci.Device{
				fakePCIDevice("8086", "1520", "0000:00:02.0", 0),
				fakePCIDevice("8086", "1520", "0000:00:03.0", 1),
			},
			map[string]string{"8086:1520": "intel_nics"},
			map[string]PerNUMADevices{
				"intel_nics": map[int][]string{
					0: []string{"0000:00:02.0"},
					1: []string{"0000:00:03.0"},
				},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := GetPCIResources(testCase.resMap, func() ([]*pci.Device, error) { return testCase.devs, nil })
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, testCase.expected) {
				t.Errorf("got %v, want %v", got, testCase.expected)
			}
		})
	}
}

func TestResourceNameForDevice(t *testing.T) {
	var testCases = []struct {
		name     string
		dev      *pci.Device
		resMap   map[string]string
		expected string
	}{
		{"anonymous", namedPCIDevice("", ""), map[string]string{}, ""},
		{"full match", namedPCIDevice("8086", "1520"), map[string]string{"8086:1520": "intel_nics"}, "intel_nics"},
		{"vendor match", namedPCIDevice("8086", "1520"), map[string]string{"8086": "intel_nics"}, "intel_nics"},
		{"full over partial match", namedPCIDevice("8086", "1520"), map[string]string{"8086:1520": "my_nics", "8086": "intel_nics"}, "my_nics"},
		{"no product match", namedPCIDevice("8086", "1520"), map[string]string{"1520": "my_nics", "8086": "intel_nics"}, "intel_nics"},
		{"ignore if no resMap", namedPCIDevice("8086", "1520"), map[string]string{}, ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, _ := ResourceNameForDevice(testCase.dev, testCase.resMap)
			if got != testCase.expected {
				t.Errorf("got %q, want %q", got, testCase.expected)
			}
		})
	}
}

func namedPCIDevice(vendorID, productID string) *pci.Device {
	return &pci.Device{
		Vendor: &pcidb.Vendor{
			ID: vendorID,
		},
		Product: &pcidb.Product{
			ID: productID,
		},
	}
}

func fakePCIDevice(vendorID, productID, address string, numaNode int) *pci.Device {
	dev := namedPCIDevice(vendorID, productID)
	dev.Address = address
	if numaNode != -1 {
		dev.Node = &topology.Node{ID: numaNode}
	}
	return dev
}
