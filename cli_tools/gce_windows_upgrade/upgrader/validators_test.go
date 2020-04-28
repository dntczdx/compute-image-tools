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

package upgrader

import (
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"google.golang.org/api/compute/v1"
)

func TestValidateParams(t *testing.T) {
	type testCase struct {
		testName        string
		u               *Upgrader
		expectError     bool
		expectedTimeout string
	}

	var u *Upgrader
	var tcs []testCase

	tcs = append(tcs, testCase{"Normal case", initTest(), false, DefaultTimeout})

	u = initTest()
	u.ClientID = ""
	tcs = append(tcs, testCase{"No client id", u, true, DefaultTimeout})

	u = initTest()
	u.SourceOS = "android"
	tcs = append(tcs, testCase{"validateOSVersion failure", u, true, DefaultTimeout})

	u = initTest()
	u.InstanceURI = "bad/url"
	tcs = append(tcs, testCase{"validateInstanceURI failure", u, true, DefaultTimeout})

	u = initTest()
	u.InstanceURI = daisy.GetInstanceURI(testProject, testZone, testInstanceNoLicense)
	tcs = append(tcs, testCase{"validateInstance failure", u, true, DefaultTimeout})

	u = initTest()
	u.Timeout = "1m"
	tcs = append(tcs, testCase{"override timeout", u, false, "1m"})

	for _, tc := range tcs {
		u = tc.u
		err := u.validateParams()
		if tc.expectError && err == nil {
			t.Errorf("[%v]: Expect error but none.", tc.testName)
		} else if !tc.expectError && err != nil {
			t.Errorf("[%v]: Unexpected error: %v", tc.testName, err)
		}
		if err != nil {
			continue
		}

		if u.Timeout != tc.expectedTimeout {
			t.Errorf("[%v]: Unexpected timeout: %v, expect: %v", tc.testName, u.Timeout, tc.expectedTimeout)
		}
		if u.machineImageBackupName == "" {
			t.Errorf("[%v]: machineImageBackupName shouldn't be empty", tc.testName)
		}
		if u.osDiskSnapshotName == "" {
			t.Errorf("[%v]: osDiskSnapshotName shouldn't be empty", tc.testName)
		}
		if u.newOSDiskName == "" {
			t.Errorf("[%v]: newOSDiskName shouldn't be empty", tc.testName)
		}
		if u.installMediaDiskName == "" {
			t.Errorf("[%v]: installMediaDiskName shouldn't be empty", tc.testName)
		}
		if *u.ProjectPtr != testProject {
			t.Errorf("[%v]: Unexpected project ptr: %v, expect: pointer to %v", tc.testName, u.ProjectPtr, testProject)
		}
	}
}

func TestValidateOSVersion(t *testing.T) {
	type testCase struct {
		testName    string
		sourceOS    string
		targetOS    string
		expectError bool
	}

	tcs := []testCase{
		{"Unsupported source OS", "windows-2008", "windows-2008r2", true},
		{"Unsupported target OS", "windows-2008r2", "windows-2012", true},
		{"Source OS not provided", "", versionWindows2012r2, true},
		{"Target OS not provided", versionWindows2008r2, "", true},
	}
	for supportedSourceOS, supportedTargetOS := range supportedSourceOSVersions {
		tcs = append(tcs, testCase{
			fmt.Sprintf("From %v to %v", supportedSourceOS, supportedTargetOS),
			supportedSourceOS,
			supportedTargetOS,
			false,
		})
	}

	for _, tc := range tcs {
		err := validateOSVersion(tc.sourceOS, tc.targetOS)
		if tc.expectError && err == nil {
			t.Errorf("[%v]: Expect error but none.", tc.testName)
		} else if !tc.expectError && err != nil {
			t.Errorf("[%v]: Unexpected error: %v", tc.testName, err)
		}
	}
}

func TestValidateInstance(t *testing.T) {
	initTest()

	type testCase struct {
		testName    string
		instanceURI string
		expectError bool
	}

	tcs := []testCase{
		{
			"Normal case without original startup script",
			daisy.GetInstanceURI(testProject, testZone, testInstance),
			false,
		},
		{
			"Normal case with original startup script",
			daisy.GetInstanceURI(testProject, testZone, testInstanceWithStartupScript),
			false,
		},
		{
			"Normal case with existing startup script backup",
			daisy.GetInstanceURI(testProject, testZone, testInstanceWithExistingStartupScriptBackup),
			false,
		},
		{
			"No disk error",
			daisy.GetInstanceURI(testProject, testZone, testInstanceNoDisk),
			true,
		},
		{
			"License error",
			daisy.GetInstanceURI(testProject, testZone, testInstanceNoLicense),
			true,
		},
		{
			"OS disk error",
			daisy.GetInstanceURI(testProject, testZone, testInstanceNoBootDisk),
			true,
		},
		{
			"Instance doesn't exist",
			daisy.GetInstanceURI(testProject, testZone, DNE),
			true,
		},
		{
			"Bad instance URI error",
			"bad/url",
			true,
		},
		{
			"No instance URI flag",
			"",
			true,
		},
	}

	for _, tc := range tcs {
		derivedVars := derivedVars{}

		err := validateInstanceURI(tc.instanceURI, &derivedVars)
		if !instanceURLRgx.Match([]byte(tc.instanceURI)) {
			if err == nil {
				t.Errorf("[%v]: Expect validateInstanceURI error but none.", tc.testName)
			}
			continue
		} else if err != nil {
			t.Errorf("[%v]: Unexpected error when validating instance URI: %v", tc.testName, err)
			continue
		}

		if tc.instanceURI != daisy.GetInstanceURI(derivedVars.project, derivedVars.zone, derivedVars.instanceName) {
			t.Errorf("[%v]: Unexpected breakdown of instance URI. Actual project, zone, instanceName are  %v, %v, %v but they are from %v.",
				tc.testName, derivedVars.project, derivedVars.zone, derivedVars.instanceName, tc.instanceURI)
		}

		err = validateInstance(&derivedVars, testSourceOS)
		if !tc.expectError {
			if err != nil {
				t.Errorf("[%v]: Unexpected error: %v", tc.testName, err)
			} else {
				if tc.instanceURI == testInstance {
					if derivedVars.windowsStartupScriptURLBackup != nil {
						t.Errorf("[%v]: Unexpected windowsStartupScriptURLBackup: %v, expect: nil", tc.testName, derivedVars.windowsStartupScriptURLBackup)
					}
					if derivedVars.windowsStartupScriptURLBackupExists {
						t.Errorf("[%v]: Unexpected windowsStartupScriptURLBackupExists: %v, expect: false", tc.testName, derivedVars.windowsStartupScriptURLBackupExists)
					}
				} else if tc.instanceURI == testInstanceWithStartupScript {
					if derivedVars.windowsStartupScriptURLBackup == nil || *derivedVars.windowsStartupScriptURLBackup != testOriginalStartupScript {
						t.Errorf("[%v]: Unexpected windowsStartupScriptURLBackup: %v, expect: %v", tc.testName, derivedVars.windowsStartupScriptURLBackup, testOriginalStartupScript)
					}
					if !derivedVars.windowsStartupScriptURLBackupExists {
						t.Errorf("[%v]: Unexpected windowsStartupScriptURLBackupExists: %v, expect: true", tc.testName, derivedVars.windowsStartupScriptURLBackupExists)
					}
				} else if tc.instanceURI == testInstanceWithExistingStartupScriptBackup {
					if derivedVars.windowsStartupScriptURLBackup != nil {
						t.Errorf("[%v]: Unexpected windowsStartupScriptURLBackup: %v, expect: nil", tc.testName, derivedVars.windowsStartupScriptURLBackup)
					}
					if !derivedVars.windowsStartupScriptURLBackupExists {
						t.Errorf("[%v]: Unexpected windowsStartupScriptURLBackupExists: %v, expect: true", tc.testName, derivedVars.windowsStartupScriptURLBackupExists)
					}
				}
			}
		} else if err == nil && tc.expectError {
			t.Errorf("[%v]: Expect an error but none.", tc.testName)
		}
	}
}

func TestValidateOSDisk(t *testing.T) {
	initTest()

	type testCase struct {
		testName    string
		osDisk      *compute.AttachedDisk
		expectError bool
	}

	tcs := []testCase{
		{
			"Disk exists",
			&compute.AttachedDisk{Source: testDiskURI, DeviceName: testDiskDeviceName,
				AutoDelete: testDiskAutoDelete, Boot: true},
			false,
		},
		{
			"Disk not exist",
			&compute.AttachedDisk{Source: daisy.GetDiskURI(testProject, testZone, DNE),
				DeviceName: testDiskDeviceName, AutoDelete: testDiskAutoDelete, Boot: true},
			true,
		},
		{
			"Disk not boot",
			&compute.AttachedDisk{Source: testDiskURI, DeviceName: testDiskDeviceName,
				AutoDelete: testDiskAutoDelete, Boot: false},
			true,
		},
	}

	for _, tc := range tcs {
		derivedVars := derivedVars{}
		err := validateOSDisk(tc.osDisk, &derivedVars)
		if !tc.expectError {
			if err != nil {
				t.Errorf("[%v]: Unexpected error: %v", tc.testName, err)
			} else {
				if derivedVars.osDiskURI != testDiskURI {
					t.Errorf("[%v]: Unexpected osDiskURI: %v, expect: %v", tc.testName, derivedVars.osDiskURI, testDiskURI)
				}
				if derivedVars.osDiskDeviceName != testDiskDeviceName {
					t.Errorf("[%v]: Unexpected osDiskDeviceName: %v, expect: %v", tc.testName, derivedVars.osDiskDeviceName, testDiskDeviceName)
				}
				if derivedVars.osDiskAutoDelete != testDiskAutoDelete {
					t.Errorf("[%v]: Unexpected osDiskAutoDelete: %v, expect: %v", tc.testName, derivedVars.osDiskAutoDelete, testDiskAutoDelete)
				}
				if derivedVars.osDiskType != testDiskType {
					t.Errorf("[%v]: Unexpected osDiskType: %v, expect: %v", tc.testName, derivedVars.osDiskType, testDiskType)
				}
			}
		} else if err == nil && tc.expectError {
			t.Errorf("[%v]: Expect an error but none.", tc.testName)
		}
	}
}

func TestValidateLicense(t *testing.T) {
	type testCase struct {
		testName    string
		osDisk      *compute.AttachedDisk
		expectError bool
	}

	tcs := []testCase{
		{
			"No license",
			&compute.AttachedDisk{},
			true,
		},
		{
			"No expected license",
			&compute.AttachedDisk{
				Licenses: []string{
					"random-license",
				}},
			true,
		},
		{
			"Expected license",
			&compute.AttachedDisk{
				Licenses: []string{
					expectedCurrentLicense[testSourceOS],
				}},
			false,
		},
		{
			"Expected license with some other license",
			&compute.AttachedDisk{
				Licenses: []string{
					"random-1",
					expectedCurrentLicense[testSourceOS],
					"random-2",
				}},
			false,
		},
		{
			"Upgraded",
			&compute.AttachedDisk{
				Licenses: []string{
					expectedCurrentLicense[testSourceOS],
					licenseToAdd[testSourceOS],
				}},
			true,
		},
	}

	for _, tc := range tcs {
		err := validateLicense(tc.osDisk, testSourceOS)
		if err != nil && !tc.expectError {
			t.Errorf("[%v]: Unexpected error: %v", tc.testName, err)
		} else if err == nil && tc.expectError {
			t.Errorf("[%v]: Expect an error but none.", tc.testName)
		}
	}
}
