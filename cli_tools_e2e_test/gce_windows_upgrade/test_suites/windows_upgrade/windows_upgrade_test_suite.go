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
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	computeUtils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_e2e_test/common/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_e2e_test/common/utils"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/junitxml"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/test_config"
)

const (
	testSuiteName = "WindowsUpgradeTests"
	standardImage = "projects/windows-cloud/global/images/windows-server-2008-r2-dc-v20200114"
)

var (
	cmds = map[utils.CLITestType]string{
		utils.Wrapper:                   "./gce_windows_upgrade",
		utils.GcloudProdWrapperLatest:   "gcloud",
		utils.GcloudLatestWrapperLatest: "gcloud",
	}
)

// TestSuite contains implementations of the e2e tests.
func TestSuite(
	ctx context.Context, tswg *sync.WaitGroup, testSuites chan *junitxml.TestSuite,
	logger *log.Logger, testSuiteRegex, testCaseRegex *regexp.Regexp,
	testProjectConfig *testconfig.Project) {

	testTypes := []utils.CLITestType{
		utils.Wrapper,
	}

	testsMap := map[utils.CLITestType]map[*junitxml.TestCase]func(
		context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, utils.CLITestType){}

	for _, testType := range testTypes {
		normalCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Normal case"))
		richParamsAndLatestInstallMedia := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Rich params and latest install media"))
		failedAndCleanup := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Failed and cleanup"))
		failedAndRollback := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Failed and rollback"))

		testsMap[testType] = map[*junitxml.TestCase]func(
			context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, utils.CLITestType){}
		testsMap[testType][normalCase] = runWindowsUpgradeNormalCase
		testsMap[testType][richParamsAndLatestInstallMedia] = runWindowsUpgradeWithRichParamsAndLatestInstallMedia
		testsMap[testType][failedAndCleanup] = runWindowsUpgradeFailedAndCleanup
		testsMap[testType][failedAndRollback] = runWindowsUpgradeFailedAndRollback
	}
	utils.CLITestSuite(ctx, tswg, testSuites, logger, testSuiteRegex, testCaseRegex,
		testProjectConfig, testSuiteName, testsMap)
}

func runWindowsUpgradeNormalCase(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	suffix := path.RandString(5)
	instanceName := fmt.Sprintf("test-upgrade-%v", suffix)
	instance := fmt.Sprintf("projects/%v/zones/%v/instances/%v",
		testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName)

	argsMap := map[utils.CLITestType][]string{
		utils.Wrapper: {
			"-client-id=e2e",
			fmt.Sprintf("-source-os=%v", "windows-2008r2"),
			fmt.Sprintf("-target-os=%v", "windows-2012r2"),
			fmt.Sprintf("-instance=%v", instance),
		},
	}
	runTest(ctx, standardImage, argsMap[testType], testType, testProjectConfig, instanceName, logger, testCase,
		true, "", false)
}

func runWindowsUpgradeWithRichParamsAndLatestInstallMedia(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
		testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	suffix := path.RandString(5)
	instanceName := fmt.Sprintf("test-upgrade-%v", suffix)
	instance := fmt.Sprintf("projects/%v/zones/%v/instances/%v",
		testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName)

	// TODO: switch to the latest install media when it's ready
	argsMap := map[utils.CLITestType][]string{
		utils.Wrapper: {
			"-client-id=e2e",
			fmt.Sprintf("-source-os=%v", "windows-2008r2"),
			fmt.Sprintf("-target-os=%v", "windows-2012r2"),
			fmt.Sprintf("-instance=%v", instance),
			fmt.Sprintf("-skip-machine-image-backup"),
			fmt.Sprintf("-auto-rollback"),
			fmt.Sprintf("-timeout=2h"),
			fmt.Sprintf("-project=%v", "compute-image-test-pool-002"),
			fmt.Sprintf("-zone=%v", "fake-zone"),
		},
	}
	runTest(ctx, standardImage, argsMap[testType], testType, testProjectConfig, instanceName, logger, testCase,
		true, "original", true)
}

// this test is cli only, since gcloud can't accept ctrl+c and cleanup
func runWindowsUpgradeFailedAndCleanup(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
		testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	suffix := path.RandString(5)
	instanceName := fmt.Sprintf("test-upgrade-%v", suffix)
	instance := fmt.Sprintf("projects/%v/zones/%v/instances/%v",
		testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName)

	argsMap := map[utils.CLITestType][]string{
		utils.Wrapper: {
			"-client-id=e2e",
			fmt.Sprintf("-source-os=%v", "windows-2008r2"),
			fmt.Sprintf("-target-os=%v", "windows-2012r2"),
			fmt.Sprintf("-instance=%v", instance),
		},
	}
	runTest(ctx, standardImage, argsMap[testType], testType, testProjectConfig, instanceName, logger, testCase,
		false, "", false)
}

// this test is cli only, since gcloud can't accept ctrl+c and cleanup
func runWindowsUpgradeFailedAndRollback(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
		testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	suffix := path.RandString(5)
	instanceName := fmt.Sprintf("test-upgrade-%v", suffix)
	instance := fmt.Sprintf("projects/%v/zones/%v/instances/%v",
		testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName)

	argsMap := map[utils.CLITestType][]string{
		utils.Wrapper: {
			"-client-id=e2e",
			fmt.Sprintf("-source-os=%v", "windows-2008r2"),
			fmt.Sprintf("-target-os=%v", "windows-2012r2"),
			fmt.Sprintf("-instance=%v", instance),
			fmt.Sprintf("-skip-machine-image-backup"),
			fmt.Sprintf("-auto-rollback"),
		},
	}
	runTest(ctx, standardImage, argsMap[testType], testType, testProjectConfig, instanceName, logger, testCase,
		false, "original-backup", false)
}

func runTest(ctx context.Context, image string, args []string, testType utils.CLITestType,
	testProjectConfig *testconfig.Project, instanceName string, logger *log.Logger, testCase *junitxml.TestCase,
	expectSuccess bool, expectedScriptURL string, autoRollback bool) {

	cmd, ok := cmds[testType]
	if !ok {
		return
	}

	// create the test instance
	if !utils.RunTestCommand("gcloud", []string{
		"compute", "instances", "create", fmt.Sprintf("--image=%v", image),
		"--boot-disk-type=pd-ssd", "--machine-type=n1-standard-4", fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
		fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID), instanceName,
	}, logger, testCase) {
		return
	}

	// set original startup script
	if expectedScriptURL != "" {
		key := "windows-startup-script-url"
		if expectedScriptURL == "original-backup" {
			key = "windows-startup-script-url-backup"
		}
		_, err := computeUtils.SetMetadata(ctx, testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName,
			key, expectedScriptURL, true)
		if err != nil {
			utils.Failure(testCase, logger, fmt.Sprintf("Failed to set metadata for %v: %v", instanceName, err))
			return
		}
	}

	var success bool
	if testType == utils.Wrapper {
		cmd := utils.RunTestCommandAsync(cmd, args, logger, testCase)

		go func() {
			// send a INT signal to fail the upgrade
			if !expectSuccess {
				// wait for "preparation" to finish
				instance, err := computeUtils.CreateInstanceObject(ctx, testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName, true)
				if err != nil {
					utils.Failure(testCase, logger, fmt.Sprintf("Failed to fetch instance object for %v: %v", instanceName, err))
					return
				}
				expectedOutput := "Beginning upgrade startup script."
				logger.Printf("[%v] Waiting for `%v` in instance serial console.", instanceName,
					expectedOutput)
				if err := instance.WaitForSerialOutput(
					expectedOutput, 1, 15*time.Second, 20*time.Minute); err != nil {
					testCase.WriteFailure("Error during validation: %v", err)
					return
				}

				err = cmd.Process.Signal(os.Interrupt)
				if err != nil {
					utils.Failure(testCase, logger, fmt.Sprintf("Failed to send ctrl+c to upgrade %v: %v", instanceName, err))
					return
				}
			}
		}()

		err := cmd.Wait()
		if err != nil {
			success = false
			utils.Failure(testCase, logger, fmt.Sprintf("Failed to execute upgrade for %v: %v", instanceName, err))
		} else {
			success = true
		}
	} else {
		success = utils.RunTestForTestType(cmd, args, testType, logger, testCase)
	}

	verifyUpgradedInstance(ctx, logger, testCase, testProjectConfig, instanceName, success,
		expectSuccess, expectedScriptURL, autoRollback)
}

func verifyUpgradedInstance(ctx context.Context, logger *log.Logger, testCase *junitxml.TestCase,
	testProjectConfig *testconfig.Project, instanceName string, success bool, expectSuccess bool,
	expectedScriptURL string, autoRollback bool) {

	if success != expectSuccess {
		utils.Failure(testCase, logger, fmt.Sprintf("Actual success: %v, expect success: %v", success, expectSuccess))
		return
	}

	instance, err := computeUtils.CreateInstanceObject(ctx, testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName, true)
	if err != nil {
		utils.Failure(testCase, logger, fmt.Sprintf("Failed to fetch instance object for %v: %v", instanceName, err))
		return
	}

	defer func() {
		// TODO: uncomment
		/*
		logger.Printf("Deleting instance `%v`", instanceName)
		if err := instance.Cleanup(); err != nil {
			logger.Printf("Instance '%v' failed to clean up: %v", instanceName, err)
		} else {
			logger.Printf("Instance '%v' cleaned up.", instanceName)
		}*/
		// machine image and snapshot will be cleaned up from test pool after 24h.
	}()

	logger.Printf("Verifying upgraded instance...")

	// verify license
	hasBootDisk := false
	for _, disk := range instance.Disks {
		if !disk.Boot {
			continue
		}
		containsAdditionalLicense := containsString(disk.Licenses, "projects/windows-cloud/global/licenses/windows-server-2012-r2-dc-in-place-upgrade")
		// success case
		if expectSuccess {
			if !containsAdditionalLicense {
				utils.Failure(testCase, logger, "Additional license not found.")
			}
		} else {
			if autoRollback {
				// rollback case
				if !containsAdditionalLicense {
					utils.Failure(testCase, logger, "Additional license found after rollback.")
				}
			} else {
				// cleanup case
				if containsAdditionalLicense {
					utils.Failure(testCase, logger, "Additional license not found.")
				}
			}
		}
		hasBootDisk = true
	}
	if !hasBootDisk {
		utils.Failure(testCase, logger, "Boot disk not found.")
		return
	}

	// verify startup script & backup
	var windowsStartupScriptURL string
	var windowsStartupScriptURLBackup string
	for _, i := range instance.Metadata.Items {
		if i.Key == "windows-startup-script-url" && i.Value != nil {
			windowsStartupScriptURL = *i.Value
		} else if i.Key == "windows-startup-script-url-backup" && i.Value != nil {
			windowsStartupScriptURLBackup = *i.Value
		}
	}
	if expectedScriptURL != windowsStartupScriptURL {
		utils.Failure(testCase, logger, fmt.Sprintf("Unexpected startup script URL: %v", windowsStartupScriptURL))
	}
	if windowsStartupScriptURLBackup != "" {
		utils.Failure(testCase, logger, fmt.Sprintf("Unexpected startup script URL backup: %v", windowsStartupScriptURLBackup))
	}

	if expectSuccess {
		// verify OS version by startup script
		err = instance.RestartWithScript("$ver=[System.Environment]::OSVersion.Version\n" +
			"Write-Host \"windows_upgrade_verify_version=$($ver.Major).$($ver.Minor)\"")
		if err != nil {
			testCase.WriteFailure("Error starting instance `%v` with script: `%v`", instanceName, err)
			return
		}
		expectedOutput := "windows_upgrade_verify_version=6.3"
		logger.Printf("[%v] Waiting for `%v` in instance serial console.", instanceName,
			expectedOutput)
		if err := instance.WaitForSerialOutput(
			expectedOutput, 1, 15*time.Second, 15*time.Minute); err != nil {
			testCase.WriteFailure("Error during validation: %v", err)
		}
	} else {
		// verify cleanup / rollback
		if autoRollback {
			// original boot disk name == instance name by default
			originalOSDisk, err := instance.Client.GetDisk(testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName)
			if err != nil {
				utils.Failure(testCase, logger, "Failed to get original OS disk.")
			}
			if len(originalOSDisk.Users) == 0 {
				utils.Failure(testCase, logger, "Original OS disk didn't get rollback.")
			}
		}
		for _, d := range instance.Disks {
			if d.Source == "projects/compute-image-tools/global/images/family/windows-install-media" {
				utils.Failure(testCase, logger, "Install media is not cleaned up.")
			}
		}
	}
}

func containsString(strs []string, s string) bool {
	for _, str := range strs {
		if str == s {
			return true
		}
	}
	return false
}
