//  Copyright 2019 Google Inc. All Rights Reserved.
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
	"testing"
)

func TestGetResourceRealName(t *testing.T) {
	initTest()

	type testCase struct {
		testName         string
		resourceName     string
		expectedRealName string
	}

	tcs := []testCase{
		{"simple resource name", "resname", "resname"},
		{"URI", "path/resname", "resname"},
		{"longer URI", "https://resource/path/resname", "resname"},
	}

	for _, tc := range tcs {
		realName := getResourceRealName(tc.resourceName)
		if realName != tc.expectedRealName {
			t.Errorf("[%v]: Expected real name '%v' != actrual real name '%v'", tc.testName, tc.expectedRealName, realName)
		}
	}
}

func TestIsNewOSDiskAttached(t *testing.T) {
	initTest()

	type testCase struct {
		testName         string
		project          string
		zone             string
		instanceName     string
		newOSDiskName    string
		expectedAttached bool
	}

	tcs := []testCase{
		{"attached case", testProject, testZone, testInstance, testDisk, true},
		{"detached case", testProject, testZone, testInstance, "new-disk", false},
		{"failed to get instance", DNE, testZone, testInstance, testDisk, false},
		{"no disk", testProject, testZone, testInstanceNoDisk, testDisk, false},
		{"no boot disk", testProject, testZone, testInstanceNoBootDisk, testDisk, false},
	}

	for _, tc := range tcs {
		attached := isNewOSDiskAttached(tc.project, tc.zone, tc.instanceName, tc.newOSDiskName)
		if attached != tc.expectedAttached {
			t.Errorf("[%v]: Expected attached status '%v' != actrual attached status '%v'", tc.testName, tc.expectedAttached, attached)
		}
	}
}
