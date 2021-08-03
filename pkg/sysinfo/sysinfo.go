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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/jaypipes/ghw/pkg/pci"

	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
)

const (
	SysDevicesOnlineCPUs = "/sys/devices/system/cpu/online"
)

type Config struct {
	ReservedCPUs string `json:"reserved_cpus"`
	// vendor:device -> resourcename
	ResourceMapping map[string]string `json:"resource_mapping"`
}

// NUMA Cell -> deviceIDs
type PerNUMADevices map[int][]string

type SysInfo struct {
	CPUs cpuset.CPUSet
	// resource name -> devices
	Resources map[string]PerNUMADevices
}

func (si SysInfo) String() string {
	b := strings.Builder{}
	fmt.Fprintf(&b, "cpus: allocatable %q\n", si.CPUs.String())
	for resourceName, numaDevs := range si.Resources {
		fmt.Fprintf(&b, "resource %q:\n", resourceName)
		for numaNode, devs := range numaDevs {
			fmt.Fprintf(&b, "  numa cell %d -> %v\n", numaNode, devs)
		}
	}
	return b.String()
}

func NewSysinfo(configFile string) (SysInfo, error) {
	sysinfo := SysInfo{}
	conf, err := readConfig(configFile)
	if err != nil {
		return sysinfo, err
	}

	sysinfo.CPUs, err = GetCPUResources(conf.ReservedCPUs, GetOnlineCPUs)
	if sysinfo.CPUs.Size() == 0 {
		return sysinfo, fmt.Errorf("no allocatable cpus")
	}

	sysinfo.Resources, err = GetPCIResources(conf.ResourceMapping, GetPCIDevices)
	if err != nil {
		return sysinfo, err
	}
	return sysinfo, nil
}

func GetCPUResources(resCPUs string, getCPUs func() (cpuset.CPUSet, error)) (cpuset.CPUSet, error) {
	reservedCPUs, err := cpuset.Parse(resCPUs)
	if err != nil {
		return cpuset.CPUSet{}, err
	}
	log.Printf("cpus: reserved %q", reservedCPUs.String())

	cpus, err := getCPUs()
	if err != nil {
		return cpuset.CPUSet{}, err
	}
	log.Printf("cpus: online %q", cpus.String())

	return cpus.Difference(reservedCPUs), nil
}

func GetPCIResources(resourceMap map[string]string, getPCIs func() ([]*pci.Device, error)) (map[string]PerNUMADevices, error) {
	numaResources := make(map[string]PerNUMADevices)
	devices, err := getPCIs()
	if err != nil {
		return numaResources, err
	}

	for _, dev := range devices {
		resourceName, ok := ResourceNameForDevice(dev, resourceMap)
		if !ok {
			continue
		}

		numaDevs, ok := numaResources[resourceName]
		if !ok {
			numaDevs = make(PerNUMADevices)
		}

		nodeID := -1
		if dev.Node != nil {
			nodeID = dev.Node.ID
		}
		numaDevs[nodeID] = append(numaDevs[nodeID], dev.Address)
		numaResources[resourceName] = numaDevs
	}

	return numaResources, nil
}

func ResourceNameForDevice(dev *pci.Device, resourceMap map[string]string) (string, bool) {
	devID := fmt.Sprintf("%s:%s", dev.Vendor.ID, dev.Product.ID)
	if resourceName, ok := resourceMap[devID]; ok {
		log.Printf("devs: resource for %s is %q", devID, resourceName)
		return resourceName, true
	}
	if resourceName, ok := resourceMap[dev.Vendor.ID]; ok {
		log.Printf("devs: resource for %s is %q", dev.Vendor.ID, resourceName)
		return resourceName, true
	}
	return "", false
}

func GetOnlineCPUs() (cpuset.CPUSet, error) {
	data, err := ioutil.ReadFile(SysDevicesOnlineCPUs)
	if err != nil {
		return cpuset.CPUSet{}, err
	}
	cpus := strings.TrimSpace(string(data))
	return cpuset.Parse(cpus)
}

func GetPCIDevices() ([]*pci.Device, error) {
	info, err := pci.New()
	if err != nil {
		return nil, err
	}
	return info.Devices, nil
}

func readConfig(configFile string) (Config, error) {
	var conf Config
	src, err := os.Open(configFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("conf: none found")
			return conf, nil
		}
		return conf, err
	}
	defer src.Close()
	err = json.NewDecoder(src).Decode(&conf)
	return conf, err
}
