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
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	SysDevicesNode = "/sys/devices/system/node"
)

// TODO review
type Hugepages struct {
	NodeID int
	SizeKB int
	Total  int
}

func GetHugepages() ([]*Hugepages, error) {
	entries, err := ioutil.ReadDir(SysDevicesNode)
	if err != nil {
		return nil, err
	}

	hugepages := []*Hugepages{}
	for _, entry := range entries {
		entryName := entry.Name()
		if entry.IsDir() && strings.HasPrefix(entryName, "node") {
			nodeID, err := strconv.Atoi(entryName[4:])
			if err != nil {
				// TODO log
				continue
			}
			nodeHugepages, err := HugepagesForNode(nodeID)
			if err != nil {
				// TODO log
				continue
			}
			hugepages = append(hugepages, nodeHugepages...)
		}
	}
	return hugepages, nil
}

func HugepagesForNode(nodeID int) ([]*Hugepages, error) {
	path := filepath.Join(
		SysDevicesNode,
		fmt.Sprintf("node%d", nodeID),
		"hugepages",
	)
	hugepages := []*Hugepages{}

	entries, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		entryName := entry.Name()
		entryPath := filepath.Join(path, entryName)
		var hugepageSizeKB int
		if n, err := fmt.Sscanf(entryName, "hugepages-%dkB", &hugepageSizeKB); n != 1 || err != nil {
			// TODO: log
			continue
		}

		totalCount, err := readIntFromFile(filepath.Join(entryPath, "nr_hugepages"))
		if err != nil {
			// TODO: log
			continue
		}

		hugepages = append(hugepages, &Hugepages{
			NodeID: nodeID,
			SizeKB: hugepageSizeKB,
			Total:  totalCount,
		})
	}

	return hugepages, nil
}

func readIntFromFile(path string) (int, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return -1, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}
