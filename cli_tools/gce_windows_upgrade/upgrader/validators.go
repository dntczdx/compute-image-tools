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
	"regexp"
	"strings"

	daisyutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/validation"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
)

const (
	rfc1035       = "[a-z]([-a-z0-9]*[a-z0-9])?"
	projectRgxStr = "[a-z]([-.:a-z0-9]*[a-z0-9])?"
)

var (
	instanceURLRgx = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?zones/(?P<zone>%[2]s)/instances/(?P<instance>%[2]s)$`, projectRgxStr, rfc1035))

	computeClient daisyCompute.Client
)

func (u *Upgrader) validateParams() error {
	if u.derivedVars == nil {
		u.derivedVars = &derivedVars{}
	}

	if err := validation.ValidateStringFlagNotEmpty(u.ClientID, ClientIDFlagKey); err != nil {
		return err
	}
	if err := validateOSVersion(u.SourceOS, u.TargetOS); err != nil {
		return err
	}
	if err := validateInstanceURI(u.InstanceURI, u.derivedVars); err != nil {
		return err
	}
	if err := validateInstance(u.derivedVars, u.SourceOS); err != nil {
		return err
	}

	if u.Timeout == "" {
		u.Timeout = DefaultTimeout
	}

	// Prepare resource names with a random suffix
	suffix := path.RandString(8)
	u.machineImageBackupName = fmt.Sprintf("windows-upgrade-backup-%v", suffix)
	u.osDiskSnapshotName = fmt.Sprintf("windows-upgrade-backup-os-%v", suffix)
	u.newOSDiskName = fmt.Sprintf("windows-upgraded-os-%v", suffix)
	u.installMediaDiskName = fmt.Sprintf("windows-install-media-%v", suffix)

	// Update 'project' value for logging purpose
	*u.ProjectPtr = u.project

	return nil
}

func validateOSVersion(sourceOS, targetOS string) error {
	if sourceOS == "" {
		return daisy.Errf("Flag -source-os must be provided. Please choose a supported version from {%v}.", strings.Join(SupportedSourceOSVersions(), ", "))
	}
	if _, ok := supportedSourceOSVersions[sourceOS]; !ok {
		return daisy.Errf("Flag -source-os value '%v' unsupported. Please choose a supported version from {%v}.", sourceOS, strings.Join(SupportedSourceOSVersions(), ", "))
	}
	if targetOS == "" {
		return daisy.Errf("Flag -target-os must be provided. Please choose a supported version from {%v}.", strings.Join(SupportedTargetOSVersions(), ", "))
	}
	if _, ok := supportedTargetOSVersions[targetOS]; !ok {
		return daisy.Errf("Flag -target-os value '%v' unsupported. Please choose a supported version from {%v}.", targetOS, strings.Join(SupportedTargetOSVersions(), ", "))
	}
	// We may chain several upgrades together in the future (for example, 2008r2->2012r2->2016->2019).
	// For now, we only support one-step upgrade.
	if expectedTargetOS, _ := supportedSourceOSVersions[sourceOS]; expectedTargetOS != targetOS {
		return daisy.Errf("Can't upgrade from %v to %v. Can only upgrade to %v.", sourceOS, targetOS, expectedTargetOS)
	}
	return nil
}

func validateInstanceURI(instanceURI string, derivedVars *derivedVars) error {
	if instanceURI == "" {
		return daisy.Errf("Flag -instance must be provided")
	}
	m := daisy.NamedSubexp(instanceURLRgx, instanceURI)
	if m == nil {
		return daisy.Errf("Please provide the instance flag in the form of 'projects/<project>/zones/<zone>/instances/<instance>', not %s", instanceURI)
	}
	derivedVars.project = m["project"]
	derivedVars.zone = m["zone"]
	derivedVars.instanceName = m["instance"]
	return nil
}

func validateInstance(derivedVars *derivedVars, sourceOS string) error {
	inst, err := computeClient.GetInstance(derivedVars.project, derivedVars.zone, derivedVars.instanceName)
	if err != nil {
		return daisy.Errf("Failed to get instance: %v", err)
	}

	if len(inst.Disks) == 0 {
		return daisy.Errf("No disks attached to the instance.")
	}
	if err := validateOSDisk(inst.Disks[0], derivedVars); err != nil {
		return err
	}
	if err := validateLicense(inst.Disks[0], sourceOS); err != nil {
		return err
	}

	if inst.Metadata != nil && inst.Metadata.Items != nil {
		for _, metadataItem := range inst.Metadata.Items {
			if metadataItem.Key == metadataKeyWindowsStartupScriptURL {
				derivedVars.windowsStartupScriptURLBackup = metadataItem.Value
			} else if metadataItem.Key == metadataKeyWindowsStartupScriptURLBackup {
				derivedVars.windowsStartupScriptURLBackupExists = true
			}
		}
	}
	// If script url backup exists, don't backup again to avoid overwriting
	if derivedVars.windowsStartupScriptURLBackupExists {
		derivedVars.windowsStartupScriptURLBackup = nil
		fmt.Printf("\n'%v' was backed up to '%v' before.\n\n",
			metadataKeyWindowsStartupScriptURL, metadataKeyWindowsStartupScriptURLBackup)
	}
	return nil
}

func validateOSDisk(osDisk *compute.AttachedDisk, derivedVars *derivedVars) error {
	if osDisk.Boot == false {
		return daisy.Errf("The instance has no boot disk.")
	}
	osDiskName := daisyutils.GetResourceRealName(osDisk.Source)
	d, err := computeClient.GetDisk(derivedVars.project, derivedVars.zone, osDiskName)
	if err != nil {
		return daisy.Errf("Failed to get OS disk info: %v", err)
	}

	derivedVars.osDiskURI = param.GetZonalResourcePath(derivedVars.zone, "disks", osDisk.Source)
	derivedVars.osDiskDeviceName = osDisk.DeviceName
	derivedVars.osDiskAutoDelete = osDisk.AutoDelete
	derivedVars.osDiskType = daisyutils.GetResourceRealName(d.Type)
	return nil
}

func validateLicense(osDisk *compute.AttachedDisk, sourceOS string) error {
	matchSourceOSVersion := false
	upgraded := false
	for _, lic := range osDisk.Licenses {
		if strings.HasSuffix(lic, expectedCurrentLicense[sourceOS]) {
			matchSourceOSVersion = true
		} else if strings.HasSuffix(lic, licenseToAdd[sourceOS]) {
			upgraded = true
		}
	}
	if !matchSourceOSVersion {
		return daisy.Errf(fmt.Sprintf("Can only upgrade GCE instance with %v license attached", expectedCurrentLicense[sourceOS]))
	}
	if upgraded {
		return daisy.Errf(fmt.Sprintf("The GCE instance is with %v license attached, which means it either has been upgraded or has started an upgrade in the past.", licenseToAdd[sourceOS]))
	}
	return nil
}
