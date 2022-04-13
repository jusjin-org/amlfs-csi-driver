/*
Copyright 2019 The Kubernetes Authors.

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
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"
	"github.com/pborman/uuid"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/e2e/framework/config"
	"sigs.k8s.io/azurelustre-csi-driver/pkg/azurelustre"
	"sigs.k8s.io/azurelustre-csi-driver/test/utils/azure"
	"sigs.k8s.io/azurelustre-csi-driver/test/utils/credentials"
	"sigs.k8s.io/azurelustre-csi-driver/test/utils/testutil"
)

const (
	kubeconfigEnvVar = "KUBECONFIG"
	reportDirEnv     = "ARTIFACTS"
	defaultReportDir = "/workspace/_artifacts"
)

var isAzureStackCloud = strings.EqualFold(os.Getenv("AZURE_CLOUD_NAME"), "AZURESTACKCLOUD")
var azureLustreDriver *azureLustre.Driver

var bringKeyStorageClassParameters = map[string]string{
	"csi.storage.k8s.io/provisioner-secret-namespace": "default",
	"csi.storage.k8s.io/node-stage-secret-namespace":  "default",
}

type testCmd struct {
	command  string
	args     []string
	startLog string
	endLog   string
}

var _ = ginkgo.BeforeSuite(func() {
	// k8s.io/kubernetes/test/e2e/framework requires env KUBECONFIG to be set
	// it does not fall back to defaults
	if os.Getenv(kubeconfigEnvVar) == "" {
		kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
		os.Setenv(kubeconfigEnvVar, kubeconfig)
	}
	handleFlags()
	framework.AfterReadingAllFlags(&framework.TestContext)

	creds, err := credentials.CreateAzureCredentialFile()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	azureClient, err := azure.GetClient(creds.Cloud, creds.SubscriptionID, creds.AADClientID, creds.TenantID, creds.AADClientSecret)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	_, err = azureClient.EnsureResourceGroup(context.Background(), creds.ResourceGroup, creds.Location, nil)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	if testutil.IsRunningInProw() {
		// Need to login to ACR using SP credential if we are running in Prow so we can push test images.
		// If running locally, user should run 'docker login' before running E2E tests
		registry := os.Getenv("REGISTRY")
		gomega.Expect(registry).NotTo(gomega.Equal(""))

		log.Println("Attempting docker login with Azure service principal")
		cmd := exec.Command("docker", "login", fmt.Sprintf("--username=%s", creds.AADClientID), fmt.Sprintf("--password=%s", creds.AADClientSecret), registry)
		err = cmd.Run()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		log.Println("docker login is successful")
	}

	// Install Azure Lustre CSI driver on cluster from project root
	e2eBootstrap := testCmd{
		command:  "make",
		args:     []string{"e2e-bootstrap"},
		startLog: "Installing Azure Lustre CSI driver ...",
		endLog:   "Azure Lustre CSI driver installed",
	}

	createMetricsSVC := testCmd{
		command:  "make",
		args:     []string{"create-metrics-svc"},
		startLog: "create metrics service ...",
		endLog:   "metrics service created",
	}
	execTestCmd([]testCmd{e2eBootstrap, createMetricsSVC})

	kubeconfig := os.Getenv(kubeconfigEnvVar)
	driverOptions := azureLustre.DriverOptions{
		NodeID:               os.Getenv("nodeid"),
		DriverName:           azureLustre.DefaultDriverName,
		EnableAzureLustreMockMount: false,
	}
	azureLustreDriver = azureLustre.NewDriver(&driverOptions)
	go func() {
		os.Setenv("AZURE_CREDENTIAL_FILE", credentials.TempAzureCredentialFilePath)
		azureLustreDriver.Run(fmt.Sprintf("unix:///tmp/csi-%s.sock", uuid.NewUUID().String()), kubeconfig, false)
	}()
})

var _ = ginkgo.AfterSuite(func() {
	createExampleDeployment := testCmd{
		command:  "bash",
		args:     []string{"hack/verify-examples.sh"},
		startLog: "create example deployments",
		endLog:   "example deployments created",
	}
	execTestCmd([]testCmd{createExampleDeployment})

	azureLustreLog := testCmd{
		command:  "bash",
		args:     []string{"test/utils/azurelustre_log.sh"},
		startLog: "===================azurelustre log===================",
		endLog:   "==================================================",
	}
	deleteMetricsSVC := testCmd{
		command:  "make",
		args:     []string{"delete-metrics-svc"},
		startLog: "delete metrics service...",
		endLog:   "metrics service deleted",
	}
	e2eTeardown := testCmd{
		command:  "make",
		args:     []string{"e2e-teardown"},
		startLog: "Uninstalling Azure Lustre CSI driver...",
		endLog:   "Azure Lustre CSI driver uninstalled",
	}
	execTestCmd([]testCmd{azureLustreLog, deleteMetricsSVC, e2eTeardown})

	// install/uninstall CSI Driver deployment scripts test
	installDriver := testCmd{
		command:  "bash",
		args:     []string{"deploy/install-driver.sh", "master", "local,enable-azurelustre"},
		startLog: "===================install CSI Driver deployment scripts test===================",
		endLog:   "===================================================",
	}
	uninstallDriver := testCmd{
		command:  "bash",
		args:     []string{"deploy/uninstall-driver.sh", "master", "local,enable-azurelustre"},
		startLog: "===================uninstall CSI Driver deployment scripts test===================",
		endLog:   "===================================================",
	}
	execTestCmd([]testCmd{installDriver, uninstallDriver})

	err := credentials.DeleteAzureCredentialFile()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
})

func TestE2E(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	reportDir := os.Getenv(reportDirEnv)
	if reportDir == "" {
		reportDir = defaultReportDir
	}
	r := []ginkgo.Reporter{reporters.NewJUnitReporter(path.Join(reportDir, "junit_01.xml"))}
	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "Azure Lustre CSI driver End-to-End Tests", r)
}

func execTestCmd(cmds []testCmd) {
	err := os.Chdir("../..")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	defer func() {
		err := os.Chdir("test/e2e")
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	}()

	projectRoot, err := os.Getwd()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(strings.HasSuffix(projectRoot, "azurelustre-csi-driver")).To(gomega.Equal(true))

	for _, cmd := range cmds {
		log.Println(cmd.startLog)
		cmdSh := exec.Command(cmd.command, cmd.args...)
		cmdSh.Dir = projectRoot
		cmdSh.Stdout = os.Stdout
		cmdSh.Stderr = os.Stderr
		err = cmdSh.Run()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		log.Println(cmd.endLog)
	}
}

// handleFlags sets up all flags and parses the command line.
func handleFlags() {
	config.CopyFlags(config.Flags, flag.CommandLine)
	framework.RegisterCommonFlags(flag.CommandLine)
	framework.RegisterClusterFlags(flag.CommandLine)
	flag.Parse()
}
