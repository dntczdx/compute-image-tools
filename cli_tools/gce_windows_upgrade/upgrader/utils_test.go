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
	"testing"
)

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

func TestGetUpgradeGuide(t *testing.T) {
	type testCase struct {
		name         string
		scriptURLPtr *string
	}

	testURL := "url"
	tcs := []testCase{
		{"has no script url", nil},
		{"has a script url", &testURL},
	}

	for _, tc := range tcs {
		u := Upgrader{
			derivedVars: &derivedVars{
				windowsStartupScriptURLBackup: tc.scriptURLPtr,
			},
		}
		_, err := getUpgradeGuide(&u)
		if err != nil {
			t.Errorf("[%v]: Unexpected error: '%v'", tc.name, err)
		}
	}
}
