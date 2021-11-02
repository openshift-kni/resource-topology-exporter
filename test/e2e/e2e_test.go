/*
Copyright 2021 The Kubernetes Authors.

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

package e2e

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/e2e/framework/config"
	"k8s.io/kubernetes/test/e2e/framework/testfiles"

	_ "github.com/k8stopologyawareschedwg/resource-topology-exporter/test/e2e/rte"
	_ "github.com/k8stopologyawareschedwg/resource-topology-exporter/test/e2e/rte_local"
	_ "github.com/k8stopologyawareschedwg/resource-topology-exporter/test/e2e/topology_updater"
	"github.com/k8stopologyawareschedwg/resource-topology-exporter/test/e2e/utils"
)

// handleFlags sets up all flags and parses the command line.
func handleFlags() {
	config.CopyFlags(config.Flags, flag.CommandLine)
	framework.RegisterCommonFlags(flag.CommandLine)
	framework.RegisterClusterFlags(flag.CommandLine)
	flag.Parse()
}

func TestMain(m *testing.M) {
	// Register test flags, then parse flags.
	handleFlags()
	setBinariesPath()

	framework.AfterReadingAllFlags(&framework.TestContext)

	// TODO: Deprecating repo-root over time... instead just use gobindata_util.go , see #23987.
	// Right now it is still needed, for example by
	// test/e2e/framework/ingress/ingress_utils.go
	// for providing the optional secret.yaml file and by
	// test/e2e/framework/util.go for cluster/log-dump.
	if framework.TestContext.RepoRoot != "" {
		testfiles.AddFileSource(testfiles.RootFileSource{Root: framework.TestContext.RepoRoot})
	}

	rand.Seed(time.Now().UnixNano())
	os.Exit(m.Run())
}

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E RTE")
}

// setBinariesPath overrides the utils.BinariesPath value from
// "github.com/k8stopologyawareschedwg/resource-topology-exporter/test/e2e/utils"
func setBinariesPath() {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Printf("Cannot retrieve tests directory")
	}

	baseDir := filepath.Dir(file)
	utils.BinariesPath = filepath.Clean(filepath.Join(baseDir, "..", "..", "./_out"))
}
