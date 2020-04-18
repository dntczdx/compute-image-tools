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
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/validation"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"google.golang.org/api/compute/v1"
)

func validateParams(params *UpgradeParams) error {
	if err := validation.ValidateStringFlagNotEmpty(params.ClientID, ClientIDFlagKey); err != nil {
		return err
	}

	if params.SourceOS == "" {
		return daisy.Errf("Flag -source-os must be provided. Please choose a supported version from {%v}.", strings.Join(SupportedSourceOSVersions(), ", "))
	}
	if _, ok := supportedSourceOSVersions[params.SourceOS]; !ok {
		return daisy.Errf("Flag -source-os value '%v' unsupported. Please choose a supported version from {%v}.", params.SourceOS, strings.Join(SupportedSourceOSVersions(), ", "))
	}
	if params.TargetOS == "" {
		return daisy.Errf("Flag -target-os must be provided. Please choose a supported version from {%v}.", strings.Join(SupportedTargetOSVersions(), ", "))
	}
	if _, ok := supportedTargetOSVersions[params.TargetOS]; !ok {
		return daisy.Errf("Flag -target-os value '%v' unsupported. Please choose a supported version from {%v}.", params.TargetOS, strings.Join(SupportedTargetOSVersions(), ", "))
	}

	// We may chain several upgrades together in the future (for example, 2008r2->2012r2->2016).
	// For now, we only support 1-step upgrade.
	if expectedTo, _ := supportedSourceOSVersions[params.SourceOS]; expectedTo != params.TargetOS {
		return daisy.Errf("Can't upgrade from %v to %v. Can only upgrade to %v.", params.SourceOS, params.TargetOS, expectedTo)
	}

	if params.InstanceURI == "" {
		return daisy.Errf("Flag -instance must be provided")
	}
	m := daisy.NamedSubexp(instanceURLRgx, params.InstanceURI)
	if m == nil {
		return daisy.Errf("Please provide the instance flag in the form of 'projects/<project>/zones/<zone>/instances/<instance>', not %s", params.InstanceURI)
	}

	params.validatedParams = &validatedParams{
		project:      m["project"],
		zone:         m["zone"],
		instanceName: m["instance"],
	}

	if err := validateInstance(params); err != nil {
		return err
	}

	// Update 'project' value for logging purpose
	*params.ProjectPtr = params.project

	return nil
}

func validateInstance(params *UpgradeParams) error {
	inst, err := computeClient.GetInstance(params.project, params.zone, params.instanceName)
	if err != nil {
		return daisy.Errf("Failed to get instance: %v", err)
	}
	if err := validateLicense(inst, params.SourceOS); err != nil {
		return err
	}

	if err := validateOSDisk(inst.Disks[0], params.validatedParams); err != nil {
		return err
	}

	for _, metadataItem := range inst.Metadata.Items {
		if metadataItem.Key == metadataKeyWindowsStartupScriptURL {
			windowsStartupScriptURLBackup = metadataItem.Value
		} else if metadataItem.Key == metadataKeyWindowsStartupScriptURLBackup {
			windowsStartupScriptURLBackupExists = true
		}
	}
	// If script url backup exists, don't backup again to overwrite it
	if windowsStartupScriptURLBackupExists {
		windowsStartupScriptURLBackup = nil
		fmt.Printf("\n'%v' was backed up to '%v' before.\n\n",
			metadataKeyWindowsStartupScriptURL, metadataKeyWindowsStartupScriptURLBackup)
	}
	return nil
}

func validateOSDisk(osDisk *compute.AttachedDisk, validatedParams *validatedParams) error {
	validatedParams.osDisk = param.GetZonalResourcePath(validatedParams.zone, "disks", osDisk.Source)
	osDiskName := getResourceRealName(osDisk.Source)
	d, err := computeClient.GetDisk(validatedParams.project, validatedParams.zone, osDiskName)
	if err != nil {
		return daisy.Errf("Failed to get OS disk info: %v", err)
	}
	validatedParams.osDiskDeviceName = osDisk.DeviceName
	validatedParams.osDiskAutoDelete = osDisk.AutoDelete
	validatedParams.osDiskType = getResourceRealName(d.Type)
	return nil
}

func validateLicense(inst *compute.Instance, sourceOS string) error {
	matchSourceOSVersion := false
	upgraded := false
	if len(inst.Disks) == 0 {
		return daisy.Errf("No disks attached to the instance.")
	}
	for _, lic := range inst.Disks[0].Licenses {
		if strings.HasSuffix(lic, expectedLicense[sourceOS]) {
			matchSourceOSVersion = true
		} else if strings.HasSuffix(lic, appendLicense[sourceOS]) {
			upgraded = true
		}
	}
	if !matchSourceOSVersion {
		return daisy.Errf(fmt.Sprintf("Can only upgrade GCE instance with %v license attached", expectedLicense[sourceOS]))
	}
	if upgraded {
		return daisy.Errf(fmt.Sprintf("The GCE instance is with %v license attached, which measn it either has been upgraded or has started a upgrade in the past.", appendLicense[sourceOS]))
	}
	return nil
}
