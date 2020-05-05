//  Copyright 2020 Google Inc. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

// Package testsuite contains e2e tests for gce_windows_upgrade
package testsuite

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	computeUtils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_e2e_test/common/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_e2e_test/common/utils"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	clitoolstestutils "github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/cli_tools"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/junitxml"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/test_config"
)

const (
	testSuiteName = "WindowsUpgradeTests"
	standardImage = "projects/windows-cloud/global/images/windows-server-2008-r2-dc-v20200114"
)

var (
	cmds = map[clitoolstestutils.CLITestType]string{
		clitoolstestutils.Wrapper:                   "./gce_windows_upgrade",
		clitoolstestutils.GcloudProdWrapperLatest:   "gcloud",
		clitoolstestutils.GcloudLatestWrapperLatest: "gcloud",
	}
)

// TestSuite contains implementations of the e2e tests.
func TestSuite(
	ctx context.Context, tswg *sync.WaitGroup, testSuites chan *junitxml.TestSuite,
	logger *log.Logger, testSuiteRegex, testCaseRegex *regexp.Regexp,
	testProjectConfig *testconfig.Project) {

	testTypes := []clitoolstestutils.CLITestType{
		clitoolstestutils.Wrapper,
	}

	testsMap := map[clitoolstestutils.CLITestType]map[*junitxml.TestCase]func(
		context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, clitoolstestutils.CLITestType){}

	for _, testType := range testTypes {
		normalCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Normal case"))
		richParamsAndLatestInstallMedia := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Rich params and latest install media"))

		testsMap[testType] = map[*junitxml.TestCase]func(
			context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, clitoolstestutils.CLITestType){}
		testsMap[testType][normalCase] = runWindowsUpgradeNormalCase
		testsMap[testType][richParamsAndLatestInstallMedia] = runWindowsUpgradeWithRichParamsAndLatestInstallMedia
	}
	clitoolstestutils.CLITestSuite(ctx, tswg, testSuites, logger, testSuiteRegex, testCaseRegex,
		testProjectConfig, testSuiteName, testsMap)
}

func runWindowsUpgradeNormalCase(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType clitoolstestutils.CLITestType) {

	suffix := path.RandString(5)
	instanceName := fmt.Sprintf("projects/%v/zones/%v/instances/test-upgrade-%v",
		testProjectConfig.TestProjectID, testProjectConfig.TestZone, suffix)

	argsMap := map[clitoolstestutils.CLITestType][]string{
		clitoolstestutils.Wrapper: {
			"-client_id=e2e",
			fmt.Sprintf("-source-os=%v", "windows-2008r2"),
			fmt.Sprintf("-target-os=%v", "windows-2012r2"),
			fmt.Sprintf("-instance=%v", instanceName),
		},
	}
	runTest(ctx, standardImage, argsMap[testType], testType, testProjectConfig, instanceName, true, logger, testCase)
}

func runWindowsUpgradeWithRichParamsAndLatestInstallMedia(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
		testProjectConfig *testconfig.Project, testType clitoolstestutils.CLITestType) {

	suffix := path.RandString(5)
	instanceName := fmt.Sprintf("projects/%v/zones/%v/instances/test-upgrade-%v",
		testProjectConfig.TestProjectID, testProjectConfig.TestZone, suffix)

	// TODO: switch to the latest install media when it's ready
	argsMap := map[clitoolstestutils.CLITestType][]string{
		clitoolstestutils.Wrapper: {
			"-client_id=e2e",
			fmt.Sprintf("-source-os=%v", "windows-2008r2"),
			fmt.Sprintf("-target-os=%v", "windows-2012r2"),
			fmt.Sprintf("-instance=%v", instanceName),
			fmt.Sprintf("-skip-machine-image-backup"),
			fmt.Sprintf("-auto-rollback"),
			fmt.Sprintf("-timeout=2h"),
			fmt.Sprintf("-project=%v", "compute-image-test-pool-002"),
			fmt.Sprintf("-zone=%v", "fake-zone"),
		},
	}
	runTest(ctx, standardImage, argsMap[testType], testType, testProjectConfig, instanceName, true, logger, testCase)
}

func runTest(ctx context.Context, image string, args []string, testType clitoolstestutils.CLITestType,
	testProjectConfig *testconfig.Project, instanceName string, expectSuccess bool, logger *log.Logger, testCase *junitxml.TestCase) {

	// create the test instance
	if !clitoolstestutils.RunTestCommand("gcloud", []string{
		"compute", "instances", "create", fmt.Sprintf("--image=%v", image),
		"--boot-disk-type=pd-ssd", "--machine-type=n1-standard-4", fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
		fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID), instanceName,
	}, logger, testCase) {
		return
	}

	if clitoolstestutils.RunTestForTestType(cmds[testType], args, testType, logger, testCase) {
		verifyUpgradedInstance(ctx, testCase, testProjectConfig, instanceName, expectSuccess, logger)
	}
}

func verifyUpgradedInstance(ctx context.Context, testCase *junitxml.TestCase,
	testProjectConfig *testconfig.Project, instanceName string, expectSuccess bool, logger *log.Logger) {

	// TODO: implement the actual verification

	_, err := daisyCompute.NewClient(ctx)
	if err != nil {
		utils.Failure(testCase, logger, fmt.Sprintf("Error creating client: %v", err))
		return
	}
	logger.Printf("Verifying upgraded instance...")
	instance, err := computeUtils.CreateInstanceObject(ctx, testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName, true)
	if err != nil {
		utils.Failure(testCase, logger, fmt.Sprintf("Failed to fetch instance object for %v: $v", instanceName, err))
		return
	}

	defer func() {
		logger.Printf("Deleting instance `%v`", instanceName)
		if err := instance.Cleanup(); err != nil {
			logger.Printf("Instance '%v' failed to clean up: %v", instanceName, err)
		} else {
			logger.Printf("Instance '%v' cleaned up.", instanceName)
		}
		// cleanup machine image
		// cleanup snapshot
	}()


	// verify license
	hasBootDisk := false
	for _, disk := range instance.Disks {
		if !disk.Boot {
			continue
		}
		if expectSuccess {
			if !containsString(disk.Licenses, "projects/windows-cloud/global/licenses/windows-server-2012-r2-dc-in-place-upgrade") {
				utils.Failure(testCase, logger, "Additional license not found.")
			}
		} else {
			// TODO: expect what for failed case?
		}
		hasBootDisk = true
	}
	if !hasBootDisk {
		utils.Failure(testCase, logger, "Boot disk not found.")
		return
	}
	// verify startup script & backup
	// verify success / failure reason

	// set startup script
	// restart, verify OS version

}

func containsString(strs []string, s string) bool {
	for _, str := range strs {
		if str == s {
			return true
		}
	}
	return false
}
