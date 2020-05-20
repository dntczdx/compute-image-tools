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
	"strings"
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
	standardImage = "projects/compute-image-tools-test/global/images/test-image-win2008r2-20200414"
	insufficientDiskSpaceImage = "projects/compute-image-tools-test/global/images/test-image-windows-2008r2-no-space"
	byolImage = "projects/compute-image-tools-test/global/images/test-image-windows-2008r2-byol"
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
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "/"))
		failedAndCleanup := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Failed and cleanup"))
		failedAndRollback := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Failed and rollback"))
		insufficientDiskSpace := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Insufficiant disk space"))
		testBYOL := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v] %v", testType, "Test BYOL"))

		testsMap[testType] = map[*junitxml.TestCase]func(
			context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, utils.CLITestType){}
		testsMap[testType][normalCase] = runWindowsUpgradeNormalCase
		testsMap[testType][richParamsAndLatestInstallMedia] = runWindowsUpgradeWithRichParamsAndLatestInstallMedia
		testsMap[testType][failedAndCleanup] = runWindowsUpgradeFailedAndCleanup
		testsMap[testType][failedAndRollback] = runWindowsUpgradeFailedAndRollback
		testsMap[testType][insufficientDiskSpace] = runWindowsUpgradeInsufficientDiskSpace
		testsMap[testType][testBYOL] = runWindowsUpgradeBYOL
	}

	if !utils.GcloudAuth(logger, nil) {
		logger.Printf("Failed to run gcloud auth.")
		testSuite := junitxml.NewTestSuite(testSuiteName)
		testSuite.Failures = 1
		testSuite.Finish(testSuites)
		tswg.Done()
		return
	}

	utils.CLITestSuite(ctx, tswg, testSuites, logger, testSuiteRegex, testCaseRegex,
		testProjectConfig, testSuiteName, testsMap)
}

func runWindowsUpgradeNormalCase(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	suffix := path.RandString(5)
	instanceName := fmt.Sprintf("test-upgrade-1-%v", suffix)
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
		true, false, "", false, 0, false)
}

func runWindowsUpgradeWithRichParamsAndLatestInstallMedia(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
		testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	suffix := path.RandString(5)
	instanceName := fmt.Sprintf("test-upgrade-2-%v", suffix)
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
			fmt.Sprintf("-timeout=2h"),
			fmt.Sprintf("-project=%v", "compute-image-test-pool-002"),
			fmt.Sprintf("-zone=%v", "fake-zone"),
			"-use-staging-install-media",
		},
	}
	runTest(ctx, standardImage, argsMap[testType], testType, testProjectConfig, instanceName, logger, testCase,
		true, false, "original", true, 2, false)
}

// this test is cli only, since gcloud can't accept ctrl+c and cleanup
func runWindowsUpgradeFailedAndCleanup(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
		testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	suffix := path.RandString(5)
	instanceName := fmt.Sprintf("test-upgrade-3-%v", suffix)
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
		false, true, "", false, 0, false)
}

// this test is cli only, since gcloud can't accept ctrl+c and cleanup
func runWindowsUpgradeFailedAndRollback(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
		testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	suffix := path.RandString(5)
	instanceName := fmt.Sprintf("test-upgrade-4-%v", suffix)
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
		false, true, "original-backup", true, 2, false)
}

func runWindowsUpgradeInsufficientDiskSpace(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
		testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	suffix := path.RandString(5)
	instanceName := fmt.Sprintf("test-upgrade-5-%v", suffix)
	instance := fmt.Sprintf("projects/%v/zones/%v/instances/%v",
		testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName)

	argsMap := map[utils.CLITestType][]string{
		utils.Wrapper: {
			"-client-id=e2e",
			fmt.Sprintf("-source-os=%v", "windows-2008r2"),
			fmt.Sprintf("-target-os=%v", "windows-2012r2"),
			fmt.Sprintf("-instance=%v", instance),
			fmt.Sprintf("-auto-rollback"),
		},
	}
	runTest(ctx, insufficientDiskSpaceImage, argsMap[testType], testType, testProjectConfig, instanceName, logger, testCase,
		false, false, "original", true, 0, false)
}

func runWindowsUpgradeBYOL(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
		testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	suffix := path.RandString(5)
	instanceName := fmt.Sprintf("test-upgrade-5-%v", suffix)
	instance := fmt.Sprintf("projects/%v/zones/%v/instances/%v",
		testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName)

	argsMap := map[utils.CLITestType][]string{
		utils.Wrapper: {
			"-client-id=e2e",
			fmt.Sprintf("-source-os=%v", "windows-2008r2"),
			fmt.Sprintf("-target-os=%v", "windows-2012r2"),
			fmt.Sprintf("-instance=%v", instance),
			fmt.Sprintf("-skip-machine-image-backup"),
		},
	}
	runTest(ctx, byolImage, argsMap[testType], testType, testProjectConfig, instanceName, logger, testCase,
		false, false, "", false, 0, true)
}

func runTest(ctx context.Context, image string, args []string, testType utils.CLITestType,
	testProjectConfig *testconfig.Project, instanceName string, logger *log.Logger, testCase *junitxml.TestCase,
	expectSuccess bool, triggerFailure bool, expectedScriptURL string, autoRollback bool, dataDiskCount int, expectValidationFailure bool) {

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

	// attach data disk
	for dataDiskIndex := 1; dataDiskIndex <= dataDiskCount; dataDiskIndex++ {
		diskName := fmt.Sprintf("%v-%v", instanceName, dataDiskIndex)
		if !utils.RunTestCommand("gcloud", []string{
			"compute", "disks", "create", fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
			fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID), "--size=10gb",
			"--image=projects/compute-image-tools-test/global/images/empty-ntfs-10g",
			diskName,
		}, logger, testCase) {
			return
		}

		if !utils.RunTestCommand("gcloud", []string{
			"compute", "instances", "attach-disk", instanceName, fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
			fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--disk=%v", diskName),
		}, logger, testCase) {
			return
		}
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
			if triggerFailure {
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
					testCase.WriteFailure("Error during waiting for preparation finished: %v", err)
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
		} else {
			success = true
		}
	} else {
		success = utils.RunTestForTestType(cmd, args, testType, logger, testCase)
	}

	verifyUpgradedInstance(ctx, logger, testCase, testProjectConfig, instanceName, success,
		expectSuccess, expectedScriptURL, autoRollback, dataDiskCount, expectValidationFailure)
}

func verifyUpgradedInstance(ctx context.Context, logger *log.Logger, testCase *junitxml.TestCase,
	testProjectConfig *testconfig.Project, instanceName string, success bool, expectSuccess bool,
	expectedScriptURL string, autoRollback bool, dataDiskCount int, expectValidationFailure bool) {

	if success != expectSuccess {
		utils.Failure(testCase, logger, fmt.Sprintf("Actual success: %v, expect success: %v", success, expectSuccess))
		return
	}

	instance, err := computeUtils.CreateInstanceObject(ctx, testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName, true)
	if err != nil {
		utils.Failure(testCase, logger, fmt.Sprintf("Failed to fetch instance object for %v: %v", instanceName, err))
		return
	}

	logger.Printf("Verifying upgraded instance...")

	// verify licenses and disks
	hasBootDisk := false
	for _, disk := range instance.Disks {
		if !disk.Boot {
			continue
		}

		if !expectValidationFailure {
			logger.Printf("Existing licenses: %v", disk.Licenses)
			if !containsSubString(disk.Licenses, "projects/windows-cloud/global/licenses/windows-server-2008-r2-dc") {
				utils.Failure(testCase, logger, "Original 2008r2 license not found.")
			}
			containsAdditionalLicense := containsSubString(disk.Licenses, "projects/windows-cloud/global/licenses/windows-server-2012-r2-dc-in-place-upgrade")
			// success case
			if expectSuccess {
				if !containsAdditionalLicense {
					utils.Failure(testCase, logger, "Additional license not found.")
				}
			} else {
				if autoRollback {
					// rollback case
					if containsAdditionalLicense {
						utils.Failure(testCase, logger, "Additional license found after rollback.")
					}
				} else {
					// cleanup case
					if !containsAdditionalLicense {
						utils.Failure(testCase, logger, "Additional license not found.")
					}
				}
			}
		}

		hasBootDisk = true
	}
	if !hasBootDisk {
		utils.Failure(testCase, logger, "Boot disk not found.")
		return
	}
	if len(instance.Disks) != dataDiskCount + 1 {
		utils.Failure(testCase, logger, fmt.Sprintf("Count of disks not match: expect %v, actual %v.", dataDiskCount+1, len(instance.Disks)))
	}

	if expectSuccess {
		// for success case, verify OS version by startup script
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
		// for failed case, verify rollback
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
	}

	// for all cases, verify cleanup: install media, startup script & backup
	for _, d := range instance.Disks {
		if strings.HasSuffix(d.Source, "global/images/family/windows-install-media") {
			utils.Failure(testCase, logger, "Install media is not cleaned up.")
		}
	}
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
}

func containsSubString(strs []string, s string) bool {
	for _, str := range strs {
		if strings.Contains(str, s) {
			return true
		}
	}
	return false
}
