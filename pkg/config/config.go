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

package config

import (
	"errors"
	"io/ioutil"
	"log"
	"os"

	"sigs.k8s.io/yaml"

	"github.com/openshift-kni/resource-topology-exporter/pkg/sysinfo"
)

type Config struct {
	ExcludeList           map[string][]string
	Resources             sysinfo.Config
	TopologyManagerPolicy string
	TopologyManagerScope  string
}

func ReadConfig(configPath string) (Config, error) {
	conf := Config{}
	// TODO modernize using os.ReadFile
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		// config is optional
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("Info: couldn't find configuration in %q", configPath)
			return conf, nil
		}
		return conf, err
	}
	err = yaml.Unmarshal(data, &conf)
	return conf, err
}
