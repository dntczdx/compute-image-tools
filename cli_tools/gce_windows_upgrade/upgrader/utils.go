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

func getUpgradeIntroduction(project, zone, instanceName, installMediaDiskName,
	osDiskSnapshotName, oldOSDiskName, newOSDiskName, machineImageName string,
	oldStartupScriptURLPtr *string, osDiskDeviceName string, osDiskAutoDelete bool) string {

	oldStartupScriptURL := "None"
	if oldStartupScriptURLPtr != nil {
		oldStartupScriptURL = *oldStartupScriptURLPtr
	}
	return fmt.Sprintf(upgradeIntroductionTemplate, project, zone, instanceName,
		installMediaDiskName, osDiskSnapshotName, oldOSDiskName, osDiskDeviceName,
		osDiskAutoDelete, newOSDiskName, machineImageName, metadataKeyWindowsStartupScriptURL,
		oldStartupScriptURL)
}

func getResourceRealName(resourceURI string) string {
	dm := strings.Split(resourceURI, "/")
	return dm[len(dm)-1]
}

func isNewOSDiskAttached(project, zone, instanceName, newOSDiskName string) bool {
	inst, err := computeClient.GetInstance(project, zone, instanceName)
	if err != nil {
		// failed to fetch info. Can't guarantee new OS disk is attached.
		return false
	}
	currentBootDiskURL := inst.Disks[0].Source

	// ignore project / zone, only compare real name, because it's guaranteed that
	// old OS disk and new OS disk are in the same project and zone.
	currentBootDiskName := getResourceRealName(currentBootDiskURL)
	return currentBootDiskName == newOSDiskName
}

func buildDaisyVarsForPreparation(project, zone, instance, machineImageBackupName,
	osDiskSnapshotName, newOSDiskName, installMedia, upgradeScriptName, sourceOS,
	oldOSDisk, osDiskType, osDiskDeviceName string, osDiskAutoDelete bool) map[string]string {

	varMap := map[string]string{}

	varMap["project"] = project
	varMap["zone"] = zone
	varMap["instance"] = instance
	varMap["install_media"] = installMedia
	varMap["upgrade_script_name"] = upgradeScriptName
	varMap["machine_image_backup_name"] = machineImageBackupName
	varMap["os_disk_snapshot_name"] = osDiskSnapshotName
	varMap["new_os_disk_name"] = newOSDiskName
	varMap["old_os_disk"] = oldOSDisk
	varMap["os_disk_type"] = osDiskType
	varMap["os_disk_device_name"] = osDiskDeviceName
	if osDiskAutoDelete {
		varMap["os_disk_auto_delete"] = "true"
	} else {
		varMap["os_disk_auto_delete"] = "false"
	}
	varMap["upgraded_license"] = appendLicense[sourceOS]

	return varMap
}

func buildDaisyVarsForUpgrade(project string, zone string, instance string, installMedia string) map[string]string {
	varMap := map[string]string{}

	varMap["project"] = project
	varMap["zone"] = zone
	varMap["instance"] = instance
	varMap["install_media"] = installMedia

	return varMap
}

func buildDaisyVarsForReboot(instance string) map[string]string {
	varMap := map[string]string{}

	varMap["instance"] = instance

	return varMap
}
