// Copyright (c) 2019-2020 Cisco Systems, Inc and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tests

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/cloudtest/pkg/commands"
	"github.com/networkservicemesh/cloudtest/pkg/config"
	"github.com/networkservicemesh/cloudtest/pkg/utils"
)

const (
	JunitReport = "reporting/junit.xml"
)

func TestShellProvider(t *testing.T) {
	g := NewWithT(t)

	testConfig := config.NewCloudTestConfig()

	testConfig.Timeout = 300

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir
	createProvider(testConfig, "a_provider")
	createProvider(testConfig, "b_provider")

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:        "simple",
		Timeout:     15,
		PackageRoot: "./sample",
	})

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:    "simple_tagged",
		Timeout: 15,
		Source: config.ExecutionSource{
			Tags: []string{"basic"},
		},
		PackageRoot: "./sample",
	})

	testConfig.Reporting.JUnitReportFile = JunitReport

	report, err := commands.PerformTesting(testConfig, &TestValidationFactory{}, &commands.Arguments{})
	g.Expect(err.Error()).To(Equal("there is failed tests 4"))

	g.Expect(report).NotTo(BeNil())

	rootSuite := report.Suites[0]

	g.Expect(len(rootSuite.Suites)).To(Equal(2))

	g.Expect(rootSuite.Suites[0].Failures).To(Equal(2))
	g.Expect(rootSuite.Suites[0].Tests).To(Equal(6))
	g.Expect(len(rootSuite.Suites[0].Suites[0].TestCases)).To(Equal(3))
	g.Expect(len(rootSuite.Suites[0].Suites[1].TestCases)).To(Equal(3))

	g.Expect(rootSuite.Suites[0].Failures).To(Equal(2))
	g.Expect(rootSuite.Suites[0].Tests).To(Equal(6))
	g.Expect(len(rootSuite.Suites[1].Suites[0].TestCases)).To(Equal(3))
	g.Expect(len(rootSuite.Suites[1].Suites[1].TestCases)).To(Equal(3))

	// Do assertions
}

func createProvider(testConfig *config.CloudTestConfig, name string) *config.ClusterProviderConfig {
	provider := &config.ClusterProviderConfig{
		Timeout:    100,
		Name:       name,
		NodeCount:  1,
		Kind:       "shell",
		RetryCount: 1,
		Instances:  2,
		Scripts: map[string]string{
			"config":  "echo ./.tests/config",
			"start":   "echo started",
			"prepare": "echo prepared",
			"install": "echo installed",
			"stop":    "echo stopped",
		},
		Enabled: true,
	}
	testConfig.Providers = append(testConfig.Providers, provider)
	return provider
}

func TestInvalidProvider(t *testing.T) {
	g := NewWithT(t)

	testConfig := config.NewCloudTestConfig()

	testConfig.Timeout = 300

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir
	createProvider(testConfig, "a_provider")
	delete(testConfig.Providers[0].Scripts, "start")

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:        "simple",
		Timeout:     2,
		PackageRoot: "./sample",
	})

	report, err := commands.PerformTesting(testConfig, &TestValidationFactory{}, &commands.Arguments{})
	logrus.Error(err.Error())
	g.Expect(err.Error()).To(Equal("Failed to create cluster instance. Error invalid start script"))

	g.Expect(report).To(BeNil())
	// Do assertions
}

func TestRequireEnvVars(t *testing.T) {
	g := NewWithT(t)

	testConfig := config.NewCloudTestConfig()

	testConfig.Timeout = 300

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir

	createProvider(testConfig, "a_provider")

	testConfig.Providers[0].EnvCheck = append(testConfig.Providers[0].EnvCheck, []string{
		"KUBECONFIG", "QWE",
	}...)

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:        "simple",
		Timeout:     2,
		PackageRoot: "./sample",
	})

	report, err := commands.PerformTesting(testConfig, &TestValidationFactory{}, &commands.Arguments{})
	logrus.Error(err.Error())
	g.Expect(err.Error()).To(Equal(
		"Failed to create cluster instance. Error environment variable are not specified  Required variables: [KUBECONFIG QWE]"))

	g.Expect(report).To(BeNil())
	// Do assertions
}

func TestRequireEnvVars_DEPS(t *testing.T) {
	g := NewWithT(t)

	testConfig := config.NewCloudTestConfig()

	testConfig.Timeout = 300

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir

	createProvider(testConfig, "a_provider")

	testConfig.Providers[0].EnvCheck = append(testConfig.Providers[0].EnvCheck, "PACKET_AUTH_TOKEN")
	testConfig.Providers[0].EnvCheck = append(testConfig.Providers[0].EnvCheck, "PACKET_PROJECT_ID")

	_ = os.Setenv("PACKET_AUTH_TOKEN", "token")
	_ = os.Setenv("PACKET_PROJECT_ID", "id")

	testConfig.Providers[0].Env = append(testConfig.Providers[0].Env, []string{
		"CLUSTER_RULES_PREFIX=packet",
		"CLUSTER_NAME=$(cluster-name)-$(uuid)",
		"KUBECONFIG=$(tempdir)/config",
		"TERRAFORM_ROOT=$(tempdir)/terraform",
		"TF_VAR_auth_token=${PACKET_AUTH_TOKEN}",
		"TF_VAR_master_hostname=devci-${CLUSTER_NAME}-master",
		"TF_VAR_worker1_hostname=ci-${CLUSTER_NAME}-worker1",
		"TF_VAR_project_id=${PACKET_PROJECT_ID}",
		"TF_VAR_public_key=${TERRAFORM_ROOT}/sshkey.pub",
		"TF_VAR_public_key_name=key-${CLUSTER_NAME}",
		"TF_LOG=DEBUG",
	}...)

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:        "simple",
		Timeout:     2,
		PackageRoot: "./sample",
	})

	report, err := commands.PerformTesting(testConfig, &TestValidationFactory{}, &commands.Arguments{})
	g.Expect(err.Error()).To(Equal("there is failed tests 2"))

	g.Expect(report).ToNot(BeNil())
	// Do assertions
}

func TestShellProviderShellTest(t *testing.T) {
	g := NewWithT(t)

	testConfig := config.NewCloudTestConfig()

	testConfig.Timeout = 300

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir
	createProvider(testConfig, "a_provider")
	createProvider(testConfig, "b_provider")

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:        "simple",
		Timeout:     15,
		PackageRoot: "./sample",
	})

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:    "simple_shell",
		Timeout: 150000,
		Kind:    "shell",
		Run: strings.Join([]string{
			"pwd",
			"ls -la",
			"echo $KUBECONFIG",
		}, "\n"),
	})

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:    "simple_shell_fail",
		Timeout: 15,
		Kind:    "shell",
		Run: strings.Join([]string{
			"pwd",
			"ls -la",
			"exit 1",
		}, "\n"),
	})

	testConfig.Reporting.JUnitReportFile = JunitReport

	report, err := commands.PerformTesting(testConfig, &TestValidationFactory{}, &commands.Arguments{})
	g.Expect(err.Error()).To(Equal("there is failed tests 4"))

	g.Expect(report).NotTo(BeNil())

	rootSuite := report.Suites[0]

	g.Expect(len(rootSuite.Suites)).To(Equal(3))

	for _, executionSuite := range rootSuite.Suites {
		switch executionSuite.Name {
		case "simple":
			g.Expect(executionSuite.Failures).To(Equal(2))
			g.Expect(executionSuite.Tests).To(Equal(6))
			g.Expect(len(executionSuite.Suites[0].TestCases)).To(Equal(3))
			g.Expect(len(executionSuite.Suites[1].TestCases)).To(Equal(3))
		case "simple_shell":
			g.Expect(executionSuite.Failures).To(Equal(0))
			g.Expect(executionSuite.Tests).To(Equal(2))
			g.Expect(len(executionSuite.Suites[0].TestCases)).To(Equal(1))
			g.Expect(len(executionSuite.Suites[1].TestCases)).To(Equal(1))
		case "simple_shell_fail":
			g.Expect(executionSuite.Failures).To(Equal(2))
			g.Expect(executionSuite.Tests).To(Equal(2))
			g.Expect(len(executionSuite.Suites[0].TestCases)).To(Equal(1))
			g.Expect(len(executionSuite.Suites[1].TestCases)).To(Equal(1))
		}
	}

	// Do assertions
}

func TestUnusedClusterShutdownByMonitor(t *testing.T) {
	g := NewWithT(t)
	logKeeper := utils.NewLogKeeper()
	defer logKeeper.Stop()
	testConfig := config.NewCloudTestConfig()

	testConfig.Timeout = 300

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir
	createProvider(testConfig, "a_provider")
	p2 := createProvider(testConfig, "b_provider")
	p2.TestDelay = 7

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:            "simple",
		Timeout:         15,
		PackageRoot:     "./sample",
		ClusterSelector: []string{"a_provider"},
	})

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:    "simple2",
		Timeout: 15,
		Source: config.ExecutionSource{
			Tags: []string{"basic"},
		},
		PackageRoot:     "./sample",
		ClusterSelector: []string{"b_provider"},
	})

	testConfig.Reporting.JUnitReportFile = JunitReport

	report, err := commands.PerformTesting(testConfig, &TestValidationFactory{}, &commands.Arguments{})
	g.Expect(err.Error()).To(Equal("there is failed tests 2"))

	g.Expect(report).NotTo(BeNil())

	rootSuite := report.Suites[0]

	g.Expect(len(rootSuite.Suites)).To(Equal(2))

	g.Expect(rootSuite.Suites[0].Failures).To(Equal(1))
	g.Expect(rootSuite.Suites[0].Tests).To(Equal(3))
	g.Expect(len(rootSuite.Suites[0].Suites[0].TestCases)).To(Equal(3))

	g.Expect(rootSuite.Suites[1].Failures).To(Equal(1))
	g.Expect(rootSuite.Suites[1].Tests).To(Equal(3))
	g.Expect(len(rootSuite.Suites[1].Suites[0].TestCases)).To(Equal(3))

	logKeeper.CheckMessagesOrder(t, []string{
		"All tasks for cluster group a_provider are complete. Starting cluster shutdown",
		"Destroying cluster  a_provider-",
		"Finished test execution",
	})
}

func TestMultiClusterTest(t *testing.T) {
	g := NewWithT(t)

	testConfig := config.NewCloudTestConfig()

	testConfig.Timeout = 300

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir
	p1 := createProvider(testConfig, "a_provider")
	p2 := createProvider(testConfig, "b_provider")
	p3 := createProvider(testConfig, "c_provider")
	p4 := createProvider(testConfig, "d_provider")

	p1.Instances = 1
	p2.Instances = 1
	p3.Instances = 1
	p4.Instances = 1

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:            "simple",
		Timeout:         15,
		PackageRoot:     "./sample",
		ClusterSelector: []string{"a_provider"},
	})

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:    "simple2",
		Timeout: 15,
		Source: config.ExecutionSource{
			Tags: []string{"interdomain"},
		},
		PackageRoot:     "./sample",
		ClusterCount:    2,
		ClusterEnv:      []string{"CFG1", "CFG2"},
		ClusterSelector: []string{"a_provider", "b_provider"},
	})
	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:    "simple3",
		Timeout: 15,
		Source: config.ExecutionSource{
			Tags: []string{"interdomain"},
		},
		PackageRoot:     "./sample",
		ClusterCount:    2,
		ClusterEnv:      []string{"CFG1", "CFG2"},
		ClusterSelector: []string{"c_provider", "d_provider"},
	})

	testConfig.Reporting.JUnitReportFile = JunitReport

	report, err := commands.PerformTesting(testConfig, &TestValidationFactory{}, &commands.Arguments{})
	g.Expect(err.Error()).To(Equal("there is failed tests 3"))

	g.Expect(report).NotTo(BeNil())

	rootSuite := report.Suites[0]

	g.Expect(len(rootSuite.Suites)).To(Equal(3))

	g.Expect(rootSuite.Suites[0].Failures).To(Equal(1))
	g.Expect(rootSuite.Suites[0].Tests).To(Equal(3))

	g.Expect(rootSuite.Suites[1].Failures).To(Equal(1))
	g.Expect(rootSuite.Suites[1].Tests).To(Equal(3))

	g.Expect(rootSuite.Suites[2].Failures).To(Equal(1))
	g.Expect(rootSuite.Suites[2].Tests).To(Equal(3))

	// Do assertions
}

func TestGlobalTimeout(t *testing.T) {
	g := NewWithT(t)

	testConfig := config.NewCloudTestConfig()
	testConfig.Timeout = 3

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	g.Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir
	createProvider(testConfig, "a_provider")

	testConfig.Executions = append(testConfig.Executions, &config.Execution{
		Name:        "simple",
		Timeout:     15,
		PackageRoot: "./sample",
	})

	testConfig.Reporting.JUnitReportFile = JunitReport

	report, err := commands.PerformTesting(testConfig, &TestValidationFactory{}, &commands.Arguments{})
	g.Expect(err.Error()).To(Equal("global timeout elapsed: 3 seconds"))

	g.Expect(report).NotTo(BeNil())

	rootSuite := report.Suites[0]

	g.Expect(len(rootSuite.Suites)).To(Equal(1))
	g.Expect(rootSuite.Suites[0].Failures).To(Equal(1))
	g.Expect(rootSuite.Suites[0].Tests).To(Equal(3))
	g.Expect(len(rootSuite.Suites[0].Suites[0].TestCases)).To(Equal(3))

	// Do assertions
}
