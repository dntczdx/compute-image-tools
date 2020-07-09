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

// Package importtestsuites contains e2e tests for image import cli tools
package importtestsuites

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/paramhelper"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_e2e_test/common/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_e2e_test/common/utils"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/junitxml"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/test_config"
)

const (
	testSuiteName = "CLI"
)

// CLITestSuite ensures that gcloud and the wrapper have consistent behavior for image imports.
func CLITestSuite(
	ctx context.Context, tswg *sync.WaitGroup, testSuites chan *junitxml.TestSuite,
	logger *log.Logger, testSuiteRegex, testCaseRegex *regexp.Regexp,
	testProjectConfig *testconfig.Project) {

	testTypes := []utils.CLITestType{
		utils.Wrapper,
		utils.GcloudProdWrapperLatest,
		utils.GcloudLatestWrapperLatest,
	}

	testsMap := map[utils.CLITestType]map[*junitxml.TestCase]func(
		context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, utils.CLITestType){}

	for _, testType := range testTypes {
		imageImportDataDiskTestCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][CLI] %v", testType, "Import data disk"))
		imageImportOSTestCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][CLI] %v", testType, "Import OS"))
		imageImportOSFromImageTestCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][CLI] %v", testType, "Import OS from image"))
		imageImportWithRichParamsTestCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][CLI] %v", testType, "Import with rich params"))
		imageImportWithDifferentNetworkParamStylesTestCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][CLI] %v", testType, "Import with different network param styles"))
		imageImportWithSubnetWithoutNetworkSpecifiedTestCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][CLI] %v", testType, "Import with subnet but without network"))
		imageImportLinuxUEFITestCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][ImageImport] %v", testType, "Import UEFI test for linux UEFI"))
		imageImportLinuxNonUEFITestCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][ImageImport] %v", testType, "Import UEFI test for linux non-UEFI"))
		imageImportLinuxHybridTestCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][ImageImport] %v", testType, "Import UEFI test for linux Hybrid"))
		imageImportWindowsUEFITestCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][ImageImport] %v", testType, "Import UEFI test for windows UEFI"))
		imageImportWindowsNonUEFITestCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][ImageImport] %v", testType, "Import UEFI test for windows non-UEFI"))

		testsMap[testType] = map[*junitxml.TestCase]func(
			context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, utils.CLITestType){}
		testsMap[testType][imageImportDataDiskTestCase] = runImageImportDataDiskTest
		testsMap[testType][imageImportOSTestCase] = runImageImportOSTest
		testsMap[testType][imageImportOSFromImageTestCase] = runImageImportOSFromImageTest
		testsMap[testType][imageImportWithRichParamsTestCase] = runImageImportWithRichParamsTest
		testsMap[testType][imageImportWithDifferentNetworkParamStylesTestCase] = runImageImportWithDifferentNetworkParamStyles
		testsMap[testType][imageImportWithSubnetWithoutNetworkSpecifiedTestCase] = runImageImportWithSubnetWithoutNetworkSpecified
		testsMap[testType][imageImportLinuxUEFITestCase] = runImageImportLinuxUEFITest
		testsMap[testType][imageImportLinuxNonUEFITestCase] = runImageImportLinuxNonUEFITest
		testsMap[testType][imageImportLinuxHybridTestCase] = runImageImportLinuxHybridTest
		testsMap[testType][imageImportWindowsUEFITestCase] = runImageImportWindowsUEFITest
		testsMap[testType][imageImportWindowsNonUEFITestCase] = runImageImportWindowsNonUEFITest

	}
	utils.CLITestSuite(ctx, tswg, testSuites, logger, testSuiteRegex, testCaseRegex,
		testProjectConfig, testSuiteName, testsMap)
}

func runImageImportDataDiskTest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	suffix := path.RandString(5)
	imageName := "e2e-test-image-import-data-disk-" + suffix

	argsMap := map[utils.CLITestType][]string{
		utils.Wrapper: {"-client_id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("-image_name=%s", imageName), "-data_disk",
			fmt.Sprintf("-source_file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			fmt.Sprintf("-zone=%v", testProjectConfig.TestZone),
		},
		utils.GcloudProdWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--data-disk", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
		},
		utils.GcloudLatestWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--data-disk", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
		},
	}

	runImportTest(ctx, argsMap[testType], testType, testProjectConfig, imageName, logger, testCase)
}

func runImageImportOSTest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	suffix := path.RandString(5)
	imageName := "e2e-test-image-import-os-" + suffix

	argsMap := map[utils.CLITestType][]string{
		utils.Wrapper: {"-client_id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("-image_name=%v", imageName), "-os=debian-9",
			fmt.Sprintf("-source_file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			fmt.Sprintf("-zone=%v", testProjectConfig.TestZone),
		},
		utils.GcloudProdWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--os=debian-9", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
		},
		utils.GcloudLatestWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--os=debian-9", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
		},
	}

	runImportTest(ctx, argsMap[testType], testType, testProjectConfig, imageName, logger, testCase)
}

func runImageImportOSFromImageTest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	suffix := path.RandString(5)
	imageName := "e2e-test-image-import-os-from-image-" + suffix

	argsMap := map[utils.CLITestType][]string{
		utils.Wrapper: {"-client_id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("-image_name=%v", imageName), "-os=debian-9", "-source_image=e2e-test-image-10g",
			fmt.Sprintf("-zone=%v", testProjectConfig.TestZone),
		},
		utils.GcloudProdWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--os=debian-9", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			"--source-image=e2e-test-image-10g",
			fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
		},
		utils.GcloudLatestWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--os=debian-9", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			"--source-image=e2e-test-image-10g",
			fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
		},
	}

	runImportTest(ctx, argsMap[testType], testType, testProjectConfig, imageName, logger, testCase)
}

// Test most of params except -oauth, -compute_endpoint_override, and -scratch_bucket_gcs_path
func runImageImportWithRichParamsTest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	family := "test-family"
	description := "test-description"
	labels := []string{"key1=value1", "key2=value2"}

	suffix := path.RandString(5)
	imageName := "e2e-test-image-import-rich-param-" + suffix

	argsMap := map[utils.CLITestType][]string{
		utils.Wrapper: {"-client_id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("-image_name=%s", imageName), "-data_disk",
			fmt.Sprintf("-source_file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			"-no_guest_environment", fmt.Sprintf("-family=%v", family), fmt.Sprintf("-description=%v", description),
			fmt.Sprintf("-network=%v-vpc-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("-subnet=%v-subnet-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("-zone=%v", testProjectConfig.TestZone),
			"-timeout=2h", "-disable_gcs_logging", "-disable_cloud_logging", "-disable_stdout_logging",
			"-no_external_ip", fmt.Sprintf("-labels=%v", strings.Join(labels, ",")),
		},
		utils.GcloudProdWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--data-disk", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			"--no-guest-environment",
			fmt.Sprintf("--network=%v-vpc-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("--subnet=%v-subnet-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("--zone=%v", testProjectConfig.TestZone), "--timeout=2h",
		},
		utils.GcloudLatestWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--data-disk", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			"--no-guest-environment",
			fmt.Sprintf("--network=%v-vpc-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("--subnet=%v-subnet-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("--zone=%v", testProjectConfig.TestZone), "--timeout=2h",
		},
	}

	runImportTestWithExtraParams(ctx, argsMap[testType], testType, testProjectConfig, imageName,
		logger, testCase, nil, []string{"UEFI_COMPATIBLE"}, family, description, labels)
}

func runImageImportWithDifferentNetworkParamStyles(ctx context.Context, testCase *junitxml.TestCase,
	logger *log.Logger, testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	suffix := path.RandString(5)
	imageName := "e2e-test-image-import-subnet-" + suffix
	region, _ := paramhelper.GetRegion(testProjectConfig.TestZone)

	argsMap := map[utils.CLITestType][]string{
		utils.Wrapper: {"-client_id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("-image_name=%s", imageName), "-data_disk",
			fmt.Sprintf("-source_file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			fmt.Sprintf("-network=global/networks/%v-vpc-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("-subnet=projects/%v/regions/%v/subnetworks/%v-subnet-1",
				testProjectConfig.TestProjectID, region, testProjectConfig.TestProjectID),
			fmt.Sprintf("-zone=%v", testProjectConfig.TestZone),
		},
		utils.GcloudProdWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--data-disk", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			fmt.Sprintf("--network=global/networks/%v-vpc-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("--subnet=projects/%v/regions/%v/subnetworks/%v-subnet-1",
				testProjectConfig.TestProjectID, region, testProjectConfig.TestProjectID),
			fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
		},
		utils.GcloudLatestWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--data-disk", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			fmt.Sprintf("--network=global/networks/%v-vpc-1", testProjectConfig.TestProjectID),
			fmt.Sprintf("--subnet=projects/%v/regions/%v/subnetworks/%v-subnet-1",
				testProjectConfig.TestProjectID, region, testProjectConfig.TestProjectID),
			fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
		},
	}

	runImportTest(ctx, argsMap[testType], testType, testProjectConfig, imageName, logger, testCase)
}

func runImageImportWithSubnetWithoutNetworkSpecified(ctx context.Context, testCase *junitxml.TestCase,
	logger *log.Logger, testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	suffix := path.RandString(5)
	imageName := "e2e-test-image-import-subnet-" + suffix
	region, _ := paramhelper.GetRegion(testProjectConfig.TestZone)

	argsMap := map[utils.CLITestType][]string{
		utils.Wrapper: {"-client_id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("-image_name=%s", imageName), "-data_disk",
			fmt.Sprintf("-source_file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			fmt.Sprintf("-subnet=https://www.googleapis.com/compute/v1/projects/%v/regions/%v/subnetworks/%v-subnet-1",
				testProjectConfig.TestProjectID, region, testProjectConfig.TestProjectID),
			fmt.Sprintf("-zone=%v", testProjectConfig.TestZone),
		},
		utils.GcloudProdWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--data-disk", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			fmt.Sprintf("--subnet=https://www.googleapis.com/compute/v1/projects/%v/regions/%v/subnetworks/%v-subnet-1",
				testProjectConfig.TestProjectID, region, testProjectConfig.TestProjectID),
			fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
		},
		utils.GcloudLatestWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", "--data-disk", fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-file=gs://%v-test-image/image-file-10g-vmdk", testProjectConfig.TestProjectID),
			fmt.Sprintf("--subnet=https://www.googleapis.com/compute/v1/projects/%v/regions/%v/subnetworks/%v-subnet-1",
				testProjectConfig.TestProjectID, region, testProjectConfig.TestProjectID),
			fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
		},
	}

	runImportTest(ctx, argsMap[testType], testType, testProjectConfig, imageName, logger, testCase)
}

func runImageImportLinuxUEFITest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
		testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	runImageImportUEFITest(ctx, testCase, logger, testProjectConfig, testType,
		"rhel-7", "projects/gce-uefi-images/global/images/family/rhel-7", true)
}

func runImageImportLinuxNonUEFITest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
		testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	runImageImportUEFITest(ctx, testCase, logger, testProjectConfig, testType,
		"debian-9", "projects/debian-cloud/global/images/family/debian-9", false)
}

func runImageImportLinuxHybridTest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
		testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	runImageImportUEFITest(ctx, testCase, logger, testProjectConfig, testType,
		"ubuntu-1804", "projects/ubuntu-os-cloud/global/images/family/ubuntu-1804-lts", true)
}

func runImageImportWindowsUEFITest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
		testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	runImageImportUEFITest(ctx, testCase, logger, testProjectConfig, testType,
		"windows-2019", "projects/gce-uefi-images/global/images/family/windows-2019-core", true)
}

func runImageImportWindowsNonUEFITest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
		testProjectConfig *testconfig.Project, testType utils.CLITestType) {

	runImageImportUEFITest(ctx, testCase, logger, testProjectConfig, testType,
		"windows-2019", "projects/windows-cloud/global/images/family/windows-2019-core", false)
}

func runImageImportUEFITest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
		testProjectConfig *testconfig.Project, testType utils.CLITestType, os string, sourceImage string,
		isUEFICompatible bool) {

	suffix := path.RandString(5)
	imageName := "e2e-test-image-import-linux-uefi-" + suffix

	argsMap := map[utils.CLITestType][]string{
		utils.Wrapper: {"-client_id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("-image_name=%v", imageName), fmt.Sprintf("-os=%v", os), fmt.Sprintf("-source_image=%v", sourceImage),
			fmt.Sprintf("-zone=%v", testProjectConfig.TestZone),
		},
		utils.GcloudProdWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", fmt.Sprintf("--os=%v", os), fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-image=%v", sourceImage),
			fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
		},
		utils.GcloudLatestWrapperLatest: {"beta", "compute", "images", "import", imageName, "--quiet",
			"--docker-image-tag=latest", fmt.Sprintf("--os=%v", os), fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
			fmt.Sprintf("--source-image=%v", sourceImage),
			fmt.Sprintf("--zone=%v", testProjectConfig.TestZone),
		},
	}

	guestOsFeatureForUEFICompatible := []string{"UEFI_COMPATIBLE"}
	var expectedGuestOsFeatures, unexpectedGuestOsFeatures []string
	if isUEFICompatible {
		expectedGuestOsFeatures = guestOsFeatureForUEFICompatible
	} else {
		unexpectedGuestOsFeatures = guestOsFeatureForUEFICompatible
	}

	runImportTestWithExtraParams(ctx, argsMap[testType], testType, testProjectConfig, imageName, logger, testCase, expectedGuestOsFeatures, unexpectedGuestOsFeatures, "", "", nil)
}

func runImportTest(ctx context.Context, args []string, testType utils.CLITestType,
		testProjectConfig *testconfig.Project, imageName string, logger *log.Logger, testCase *junitxml.TestCase) {

	runImportTestWithExtraParams(ctx, args, testType, testProjectConfig, imageName, logger, testCase, nil, nil, "", "", nil)
}

func runImportTestWithExtraParams(ctx context.Context, args []string, testType utils.CLITestType,
		testProjectConfig *testconfig.Project, imageName string, logger *log.Logger, testCase *junitxml.TestCase,
		expectedGuestOsFeatures []string, unexpectedGuestOsFeatures []string,
		expectedFamily string, expectedDescription string, expectedLabels []string) {

	cmds := map[utils.CLITestType]string{
		utils.Wrapper:                   "./gce_vm_image_import",
		utils.GcloudProdWrapperLatest:   "gcloud",
		utils.GcloudLatestWrapperLatest: "gcloud",
	}

	// "family", "description" and "labels" hasn't been supported by gcloud
	if testType != utils.Wrapper {
		expectedFamily = ""
		expectedDescription = ""
		expectedLabels = nil
	}

	if utils.RunTestForTestType(cmds[testType], args, testType, logger, testCase) {
		verifyImportedImage(ctx, testCase, testProjectConfig, imageName, logger, expectedFamily,
			expectedDescription, expectedLabels, expectedGuestOsFeatures, unexpectedGuestOsFeatures)
	}
}

func verifyImportedImage(ctx context.Context, testCase *junitxml.TestCase,
	testProjectConfig *testconfig.Project, imageName string, logger *log.Logger,
		expectedFamily string, expectedDescription string, expectedLabels []string,
		expectedGuestOsFeatures []string, unexpectedGuestOsFeatures []string) {

	logger.Printf("Verifying imported image...")
	image, err := compute.CreateImageObject(ctx, testProjectConfig.TestProjectID, imageName)
	if err != nil {
		testCase.WriteFailure("Image '%v' doesn't exist after import: %v", imageName, err)
		logger.Printf("Image '%v' doesn't exist after import: %v", imageName, err)
		return
	}
	logger.Printf("Image '%v' exists! Import success.", imageName)

	if expectedFamily != "" && image.Family != expectedFamily {
		utils.Failure(testCase, logger, fmt.Sprintf("Image '%v' family expect: %v, actual: %v", imageName, expectedFamily, image.Family))
	}

	if expectedDescription != "" && image.Description != expectedDescription {
		utils.Failure(testCase, logger, fmt.Sprintf("Image '%v' description expect: %v, actual: %v", imageName, expectedDescription, image.Description))
	}

	if expectedLabels != nil {
		imageLabels := make([]string, 0, len(image.Labels))
		for k, v := range image.Labels {
			imageLabels = append(imageLabels, k+"="+v)
		}
		if !containsAll(imageLabels, expectedLabels) {
			utils.Failure(testCase, logger, fmt.Sprintf("Image '%v' labels expect: %v, actual: %v", imageName, strings.Join(expectedLabels, ","), strings.Join(imageLabels, ",")))
		}
	}

	if expectedGuestOsFeatures != nil {
		guestOsFeatures := make([]string, 0, len(image.GuestOsFeatures))
		for _, f := range image.GuestOsFeatures {
			guestOsFeatures = append(guestOsFeatures, f.Type)
		}
		if !containsAll(guestOsFeatures, expectedGuestOsFeatures) {
			testCase.WriteFailure("Image '%v' GuestOsFeatures expect: %v, actual: %v", imageName, strings.Join(expectedGuestOsFeatures, ","), strings.Join(guestOsFeatures, ","))
			logger.Printf("Image '%v' GuestOsFeatures expect: %v, actual: %v", imageName, strings.Join(expectedGuestOsFeatures, ","), strings.Join(guestOsFeatures, ","))
		}
	}

	if unexpectedGuestOsFeatures != nil {
		guestOsFeatures := make([]string, 0, len(image.GuestOsFeatures))
		for _, f := range image.GuestOsFeatures {
			guestOsFeatures = append(guestOsFeatures, f.Type)
		}
		if containsAny(guestOsFeatures, expectedGuestOsFeatures) {
			testCase.WriteFailure("Image '%v' GuestOsFeatures unexpect: %v, actual: %v", imageName, strings.Join(unexpectedGuestOsFeatures, ","), strings.Join(guestOsFeatures, ","))
			logger.Printf("Image '%v' GuestOsFeatures unexpect: %v, actual: %v", imageName, strings.Join(unexpectedGuestOsFeatures, ","), strings.Join(guestOsFeatures, ","))
		}
	}

	if err := image.Cleanup(); err != nil {
		logger.Printf("Image '%v' failed to clean up.", imageName)
	} else {
		logger.Printf("Image '%v' cleaned up.", imageName)
	}
}

func containsAll(arr []string, subarr []string) bool {
	for item := range subarr {
		exists := false
		for i := range arr {
			if item == i {
				exists = true
				break
			}
		}
		if !exists {
			return false
		}
	}
	return true
}

func containsAny(arr []string, subarr []string) bool {
	for item := range subarr {
		for i := range arr {
			if item == i {
				return true
			}
		}
	}
	return false
}
