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
	"bytes"
	"strings"
	"text/template"

	daisyutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

const (
	upgradeIntroductionTemplate = "The following resources will be created/accessed during the upgrade. " +
		"Please keep track of their names if manual cleanup and/or rollback are necessary.\n" +
		"All resources are in project '{{.project}}', zone '{{.zone}}'.\n" +
		"1. Instance: {{.instanceName}}\n" +
		"2. Disk for install media: {{.installMediaDiskName}}\n" +
		"3. Snapshot for original OS disk: {{.osDiskSnapshotName}}\n" +
		"4. Original OS disk: {{.osDiskName}}\n" +
		"   - Device name of the attachment: {{.osDiskDeviceName}}\n" +
		"   - AutoDelete of the attachment: {{.osDiskAutoDelete}}\n" +
		"5. New OS disk: {{.newOSDiskName}}\n" +
		"6. Machine image: {{.machineImageName}}\n" +
		"7. Original startup script url: {{.originalStartupScriptURL}}\n" +
		"\n" +
		"When upgrading succeeded but cleanup failed, please manually cleanup by following steps:\n" +
		"1. Delete 'windows-startup-script-url' from the instance's metadata if there isn't an original value. " +
		"If there is an original value, restore it. The original value is backed up as metadata 'windows-startup-script-url-backup'.\n" +
		"2. Detach the install media disk from the instance and delete it.\n" +
		"\n" +
		"When upgrading failed but you didn't enable auto-rollback, or auto-rollback failed, or " +
		"upgrading succeeded but you still need to rollback for any reason, " +
		"please manually rollback by following steps:\n" +
		"1. Detach the new OS disk from the instance and delete it.\n" +
		"2. Attach the old OS disk as boot disk.\n" +
		"3. Detach the install media disk from the instance and delete it.\n" +
		"4. Delete 'windows-startup-script-url' from the instance's metadata if there isn't an original value. " +
		"If there is an original value, restore it. The original value is backed up as metadata 'windows-startup-script-url-backup'.\n" +
		"\n" +
		"Once you verified the upgrading succeeded and decided to never rollback, you can:\n" +
		"1. Delete the original OS disk.\n" +
		"2. Delete the machine image.\n" +
		"3. Delete the snapshot.\n" +
		"\n"
)

func reverseMap(m map[string]string) map[string]string {
	newMap := make(map[string]string, len(m))
	for k, v := range m {
		newMap[v] = k
	}
	return newMap
}

func getKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// SupportedSourceOSVersions returns supported source versions of upgrading
func SupportedSourceOSVersions() []string {
	return getKeys(supportedSourceOSVersions)
}

// SupportedTargetOSVersions returns supported target versions of upgrading
func SupportedTargetOSVersions() []string {
	return getKeys(supportedTargetOSVersions)
}

func getUpgradeGuide(project, zone, instanceName, installMediaDiskName,
	osDiskSnapshotName, oldOSDiskName, newOSDiskName, machineImageName string,
	originalStartupScriptURLPtr *string, osDiskDeviceName string, osDiskAutoDelete bool) (string, error) {

	originalStartupScriptURL := "None."
	if originalStartupScriptURLPtr != nil {
		originalStartupScriptURL = *originalStartupScriptURLPtr
	}
	if machineImageName == "" {
		machineImageName = "Not created. Machine Image backup is disabled."
	}

	t, err := template.New("guide").Option("missingkey=error").Parse(upgradeIntroductionTemplate)
	if err != nil {
		return "", daisy.Errf("Failed to parse upgrade guide.")
	}
	var buf bytes.Buffer
	varMap := map[string]interface{}{
		"project":                  project,
		"zone":                     zone,
		"instanceName":             instanceName,
		"installMediaDiskName":     installMediaDiskName,
		"osDiskSnapshotName":       osDiskSnapshotName,
		"osDiskName":               oldOSDiskName,
		"osDiskDeviceName":         osDiskDeviceName,
		"osDiskAutoDelete":         osDiskAutoDelete,
		"newOSDiskName":            newOSDiskName,
		"machineImageName":         machineImageName,
		"originalStartupScriptURL": originalStartupScriptURL,
	}
	if err := t.Execute(&buf, varMap); err != nil {
		return "", daisy.Errf("Failed to generate upgrade guide.")
	}
	return string(buf.Bytes()), nil
}

func isNewOSDiskAttached(project, zone, instanceName, newOSDiskName string) bool {
	inst, err := computeClient.GetInstance(project, zone, instanceName)
	if err != nil {
		// failed to fetch info. Can't guarantee new OS disk is attached.
		return false
	}
	if inst.Disks == nil || len(inst.Disks) == 0 || inst.Disks[0].Boot == false {
		// if the instance has no boot disk attached
		return false
	}

	currentBootDiskURL := inst.Disks[0].Source

	// ignore project / zone, only compare real name, because it's guaranteed that
	// old OS disk and new OS disk are in the same project and zone.
	currentBootDiskName := daisyutils.GetResourceRealName(currentBootDiskURL)
	return currentBootDiskName == newOSDiskName
}

func needReboot(err error) bool {
	return strings.Contains(err.Error(), "Windows needs to be restarted")
}

func setWorkflowAttributes(w *daisy.Workflow, u *Upgrader) {
	daisycommon.SetWorkflowAttributes(w, u.project, u.zone, u.ScratchBucketGcsPath,
		u.Oauth, u.Timeout, u.Ce, u.GcsLogsDisabled, u.CloudLogsDisabled, u.StdoutLogsDisabled)
}
