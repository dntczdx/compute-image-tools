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

// Package windowsupgrade contains e2e tests for gce_windows_upgrade
package windowsupgradetestsuite

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_e2e_test/common/compute"
	clitoolstestutils "github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/cli_tools"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/junitxml"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/test_config"
)

const (
	testSuiteName = "WindowsUpgradeTests"
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

	argsMap := map[clitoolstestutils.CLITestType][]string{
		clitoolstestutils.Wrapper: {
			"-client-id=e2e",
			fmt.Sprintf("-source-os=%v", "windows-2008r2"),
			fmt.Sprintf("-target-os=%v", "windows-2012r2"),
			fmt.Sprintf("-instance=projects/%v/zones/%v/instances/test-upgrade-%v",
				testProjectConfig.TestProjectID, testProjectConfig.TestZone, suffix),
		},
	}
	runTest(ctx, argsMap[testType], testType, testProjectConfig, imageName, logger, testCase)
}

func runWindowsUpgradeWithRichParamsAndLatestInstallMedia(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
		testProjectConfig *testconfig.Project, testType clitoolstestutils.CLITestType) {

	suffix := path.RandString(5)

	// TODO: switch to the latest install media when it's ready
	argsMap := map[clitoolstestutils.CLITestType][]string{
		clitoolstestutils.Wrapper: {
			"-client_id=e2e",
			fmt.Sprintf("-source-os=%v", "windows-2008r2"),
			fmt.Sprintf("-target-os=%v", "windows-2012r2"),
			fmt.Sprintf("-instance=projects/%v/zones/%v/instances/test-upgrade-%v",
				testProjectConfig.TestProjectID, testProjectConfig.TestZone, suffix),
			fmt.Sprintf("-skip-machine-image-backup"),
			fmt.Sprintf("-auto-rollback"),
			fmt.Sprintf("-timeout=2h"),
			fmt.Sprintf("-project=%v", "compute-image-test-pool-002"),
			fmt.Sprintf("-zone=%v", "fake-zone"),
		},
	}
	runTest(ctx, argsMap[testType], testType, testProjectConfig, imageName, logger, testCase)
}

func runTest(ctx context.Context, args []string, testType clitoolstestutils.CLITestType,
	testProjectConfig *testconfig.Project, imageName string, logger *log.Logger, testCase *junitxml.TestCase) {

	// gcloud is not ready yet. However, it's harmless to keep the command name here.
	cmds := map[clitoolstestutils.CLITestType]string{
		clitoolstestutils.Wrapper:                   "./gce_windows_upgrade",
		clitoolstestutils.GcloudProdWrapperLatest:   "gcloud",
		clitoolstestutils.GcloudLatestWrapperLatest: "gcloud",
	}

	if clitoolstestutils.RunTestForTestType(cmds[testType], args, testType, logger, testCase) {
		verifyUpgradedInstance(ctx, testCase, testProjectConfig, imageName, logger)
	}
}

func verifyUpgradedInstance(ctx context.Context, testCase *junitxml.TestCase,
	testProjectConfig *testconfig.Project, imageName string, logger *log.Logger) {

	// get instance object

	// verify startup script & backup
	// verify failure reason

	// set startup script
	// restart, verify OS version

	// delete instance
}
