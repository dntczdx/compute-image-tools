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
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	daisyutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

const (
	logPrefix                      = "[windows-upgrade]"
	rebootWorkflowPath             = "daisy_workflows/windows_upgrade/reboot.wf.json"
	upgradePreparationWorkflowPath = "daisy_workflows/windows_upgrade/windows_upgrade_preparation.wf.json"
	cleanupWorkflowPath            = "daisy_workflows/windows_upgrade/cleanup.wf.json"
	rollbackWorkflowPath           = "daisy_workflows/windows_upgrade/rollback.wf.json"

	rfc1035       = "[a-z]([-a-z0-9]*[a-z0-9])?"
	projectRgxStr = "[a-z]([-.:a-z0-9]*[a-z0-9])?"

	metadataKeyWindowsStartupScriptURL       = "windows-startup-script-url"
	metadataKeyWindowsStartupScriptURLBackup = "windows-startup-script-url-backup"

	versionWindows2008r2 = "windows-2008r2"
	versionWindows2012r2 = "windows-2012r2"

	upgradeIntroductionTemplate = "The following resources will be created/touched during the upgrade. " +
		"Please record their name in order for cleanup or manual rollback.\n" +
		"All resources are in project '%v', zone '%v'.\n" +
		"1. Instance: %v\n" +
		"2. Disk for install media: %v\n" +
		"3. Snapshot for original OS disk: %v\n" +
		"4. Original OS disk: %v\n" +
		"   - Device name of the attachment: %v\n" +
		"   - AutoDelete of the attachment: %v\n" +
		"5. New OS disk: %v\n" +
		"6. Machine image: %v\n" +
		"7. Original startup script url '%v': %v\n" +
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

var (
	supportedSourceOSVersions = map[string]string{versionWindows2008r2: versionWindows2012r2}
	supportedTargetOSVersions = reverseMap(supportedSourceOSVersions)

	upgradeScriptName        = map[string]string{versionWindows2008r2: "upgrade_script_2008r2_to_2012r2.ps1"}
	upgradeWorkflowPath      = map[string]string{versionWindows2008r2: "daisy_workflows/windows_upgrade/windows_upgrade_2008r2_to_2012r2.wf.json"}
	retryUpgradeWorkflowPath = map[string]string{versionWindows2008r2: "daisy_workflows/windows_upgrade/windows_upgrade_2008r2_to_2012r2_retry.wf.json"}

	expectedLicense = map[string]string{versionWindows2008r2: "projects/windows-cloud/global/licenses/windows-server-2008-r2-dc"}
	appendLicense   = map[string]string{versionWindows2008r2: "projects/windows-cloud/global/licenses/windows-server-2012-r2-dc-in-place-upgrade"}

	instanceURLRgx = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?zones/(?P<zone>%[2]s)/instances/(?P<instance>%[2]s)$`, projectRgxStr, rfc1035))

	windowsStartupScriptURLBackup       *string
	windowsStartupScriptURLBackupExists bool

	computeClient daisyCompute.Client
)

type validatedParams struct {
	project          string
	zone             string
	instanceName     string
	osDisk           string
	osDiskType       string
	osDiskDeviceName string
	osDiskAutoDelete bool
}

// Run runs upgrade workflow.
func Run(clientID string, instance string, skipMachineImageBackup bool, autoRollback bool, sourceOS string,
	targetOS string, project *string, timeout string, scratchBucketGcsPath string, oauth string,
	ce string, gcsLogsDisabled bool, cloudLogsDisabled bool, stdoutLogsDisabled bool,
	currentExecutablePath string) (*daisy.Workflow, error) {

	log.SetPrefix(logPrefix + " ")

	var err error
	ctx := context.Background()
	computeClient, err = daisyCompute.NewClient(ctx, option.WithCredentialsFile(oauth))
	if err != nil {
		return nil, daisy.Errf("Failed to create GCE client: %v", err)
	}

	instance = strings.TrimSpace(instance)
	sourceOS = strings.TrimSpace(sourceOS)
	targetOS = strings.TrimSpace(targetOS)
	validatedParams, err := validateParams(instance, sourceOS, targetOS)
	if err != nil {
		return nil, err
	}
	*project = validatedParams.project

	return runUpgradeWorkflow(ctx, currentExecutablePath, skipMachineImageBackup, autoRollback,
		sourceOS, targetOS, instance, validatedParams.project, validatedParams.zone, validatedParams.instanceName,
		validatedParams.osDisk, validatedParams.osDiskType, validatedParams.osDiskDeviceName,
		validatedParams.osDiskAutoDelete, timeout, scratchBucketGcsPath, oauth, ce, gcsLogsDisabled, cloudLogsDisabled,
		stdoutLogsDisabled)
}

func validateParams(instancePath string, sourceOS string, targetOS string) (validatedParams, error) {
	validatedParams := validatedParams{}

	if sourceOS == "" {
		return validatedParams, daisy.Errf("Flag -source-os must be provided. Please choose a supported version from {%v}.", strings.Join(SupportedSourceOSVersions(), ", "))
	}
	if _, ok := supportedSourceOSVersions[sourceOS]; !ok {
		return validatedParams, daisy.Errf("Flag -source-os value '%v' unsupported. Please choose a supported version from {%v}.", sourceOS, strings.Join(SupportedSourceOSVersions(), ", "))
	}
	if targetOS == "" {
		return validatedParams, daisy.Errf("Flag -target-os must be provided. Please choose a supported version from {%v}.", strings.Join(SupportedTargetOSVersions(), ", "))
	}
	if _, ok := supportedTargetOSVersions[targetOS]; !ok {
		return validatedParams, daisy.Errf("Flag -target-os value '%v' unsupported. Please choose a supported version from {%v}.", targetOS, strings.Join(SupportedTargetOSVersions(), ", "))
	}

	// We may chain several upgrades together in the future (for example, 2008r2->2012r2->2016).
	// For now, we only support 1-step upgrade.
	if expectedTo, _ := supportedSourceOSVersions[sourceOS]; expectedTo != targetOS {
		return validatedParams, daisy.Errf("Can't upgrade from %v to %v. Can only upgrade to %v.", sourceOS, targetOS, expectedTo)
	}

	if instancePath == "" {
		return validatedParams, daisy.Errf("Flag -instance must be provided")
	}
	m := daisy.NamedSubexp(instanceURLRgx, instancePath)
	if m == nil {
		return validatedParams, daisy.Errf("Please provide the instance flag in the form of 'projects/<project>/zones/<zone>/instances/<instance>', not %s", instancePath)
	}
	validatedParams.project = m["project"]
	validatedParams.zone = m["zone"]
	validatedParams.instanceName = m["instance"]

	if err := validateInstance(&validatedParams, sourceOS); err != nil {
		return validatedParams, err
	}

	return validatedParams, nil
}

func validateInstance(validatedParams *validatedParams, sourceOS string) error {
	inst, err := computeClient.GetInstance(validatedParams.project, validatedParams.zone, validatedParams.instanceName)
	if err != nil {
		return daisy.Errf("Failed to get instance: %v", err)
	}
	if err := validateLicense(inst, sourceOS); err != nil {
		return err
	}

	if err := validateOSDisk(inst.Disks[0], validatedParams, err); err != nil {
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

func validateOSDisk(osDisk *compute.AttachedDisk, validatedParams *validatedParams, err error) error {
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

func runUpgradeWorkflow(ctx context.Context, currentExecutablePath string,
	skipMachineImageBackup bool, autoRollback bool, sourceOS string, targetOS string, instance string,
	project string, zone string, instanceName string, oldOSDisk string, osDiskType string,
	osDiskDeviceName string, osDiskAutoDelete bool, timeout string, scratchBucketGcsPath string, oauth string,
	ce string, gcsLogsDisabled bool, cloudLogsDisabled bool, stdoutLogsDisabled bool) (*daisy.Workflow, error) {

	var err error
	workflowPath := path.ToWorkingDir(upgradeWorkflowPath[sourceOS], currentExecutablePath)
	retryWorkflowPath := path.ToWorkingDir(retryUpgradeWorkflowPath[sourceOS], currentExecutablePath)
	suffix := path.RandString(8)
	machineImageBackupName := fmt.Sprintf("backup-machine-image-%v", suffix)
	osDiskSnapshotName := fmt.Sprintf("win-upgrade-os-disk-snapshot-%v", suffix)
	newOSDiskName := fmt.Sprintf("windows-upgraded-os-disk-%v", suffix)
	installMediaDiskName := fmt.Sprintf("windows-install-media-%v", suffix)

	preparationVarMap := buildDaisyVarsForPreparation(project, zone, instance, machineImageBackupName,
		osDiskSnapshotName, newOSDiskName, installMediaDiskName, upgradeScriptName[sourceOS],
		sourceOS, oldOSDisk, osDiskType, osDiskDeviceName, osDiskAutoDelete)
	upgradeVarMap := buildDaisyVarsForUpgrade(project, zone, instance, installMediaDiskName)
	rebootVarMap := buildDaisyVarsForReboot(instance)
	rollbackVarMap := buildDaisyVarsForRollback(project, zone, instance, installMediaDiskName, osDiskDeviceName, oldOSDisk, newOSDiskName, osDiskAutoDelete)

	// If upgrade failed, run cleanup/rollback before exiting.
	defer func() {
		if err == nil {
			fmt.Printf("\nSuccessfully upgraded instance '%v' to %v!\n", instance, targetOS)
			fmt.Printf("\nPlease verify the functionality of the instance. If " +
				"it has a problem and can't be fixed, please manually rollback following the guide.\n\n")
			return
		}

		isNewOSDiskAttached := isNewOSDiskAttached(project, zone, instanceName, newOSDiskName)
		if autoRollback {
			if isNewOSDiskAttached {
				fmt.Printf("\nFailed to finish upgrading. Rollback to original state from original OS disk '%v'...\n\n", oldOSDisk)
				_, err := rollback(ctx, rollbackVarMap, project, zone, scratchBucketGcsPath, oauth, timeout, ce,
					gcsLogsDisabled, cloudLogsDisabled, stdoutLogsDisabled)
				if err != nil {
					fmt.Printf("\nFailed to rollback. Error: %v\nPlease manually rollback following the guide.\n\n", err)
				} else {
					fmt.Printf("\nRollback to original state is done. Please verify whether it works as expected. " +
						"If not, you may consider restoring the whole instance from the machine image.\n\n")
				}
				return
			}
			fmt.Printf("\nNew OS disk hadn't been attached when failure happened. No need to rollback. "+
				"If the instance can't work as expected, please verify whether original OS disk %v is attached "+
				"and whether the instance has been started. If necessary, please manually rollback following the guide.\n\n", oldOSDisk)
		} else {
			if isNewOSDiskAttached {
				fmt.Printf("\nFailed to finish upgrading. Please manually rollback following the guide.\n\n")
			}
		}
		fmt.Print("\nCleaning up temporary resources...\n\n")
		if _, err := cleanup(ctx, upgradeVarMap, project, zone, scratchBucketGcsPath, oauth, timeout, ce,
			gcsLogsDisabled, cloudLogsDisabled, stdoutLogsDisabled); err != nil {

			fmt.Printf("\nFailed to cleanup temporary resources: %v\n"+
				"Please follow the guide to manually cleanup.\n\n", err)
		}
	}()

	fmt.Printf("%v\n\n", getUpgradeIntroduction(project, zone, getResourceRealName(instance),
		installMediaDiskName, osDiskSnapshotName, getResourceRealName(oldOSDisk), newOSDiskName,
		machineImageBackupName, windowsStartupScriptURLBackup, osDiskDeviceName, osDiskAutoDelete))

	// step 1: preparation - take snapshot, attach install media, backup/set startup script
	fmt.Print("\nPreparing for upgrade...\n\n")
	prepareWf, err := prepare(ctx, preparationVarMap, project, zone, scratchBucketGcsPath, oauth,
		timeout, ce, gcsLogsDisabled, cloudLogsDisabled, stdoutLogsDisabled, skipMachineImageBackup, instanceName)
	if err != nil {
		return prepareWf, err
	}

	// step 2: run upgrade.
	fmt.Print("\nRunning upgrade...\n\n")
	upgradeWf, err := upgrade(ctx, workflowPath, upgradeVarMap, project, zone, scratchBucketGcsPath,
		oauth, timeout, ce, gcsLogsDisabled, cloudLogsDisabled, stdoutLogsDisabled)
	if err == nil {
		return upgradeWf, nil
	}

	// step 3: reboot if necessary.
	if !needReboot(err) {
		return upgradeWf, err
	}
	fmt.Print("\nRebooting...\n\n")
	rebootWf, err := reboot(ctx, rebootVarMap, project, zone, scratchBucketGcsPath, oauth,
		timeout, ce, gcsLogsDisabled, cloudLogsDisabled, stdoutLogsDisabled)
	if err != nil {
		return rebootWf, err
	}

	// step 4: retry upgrade.
	fmt.Print("\nRunning upgrade...\n\n")
	upgradeWf, err = upgrade(ctx, retryWorkflowPath, upgradeVarMap, project, zone, scratchBucketGcsPath,
		oauth, timeout, ce, gcsLogsDisabled, cloudLogsDisabled, stdoutLogsDisabled)
	return upgradeWf, err
}

func prepare(ctx context.Context, preparationVarMap map[string]string, project string, zone string, scratchBucketGcsPath string,
	oauth string, timeout string, ce string, gcsLogsDisabled bool, cloudLogsDisabled bool, stdoutLogsDisabled bool,
	skipMachineImageBackup bool, instanceName string) (*daisy.Workflow, error) {

	// 'windows-startup-script-url' exists, backup it
	if windowsStartupScriptURLBackup != nil {
		fmt.Printf("\nDetected current '%v', value='%v'. Will backup to '%v'.\n\n", metadataKeyWindowsStartupScriptURL,
			*windowsStartupScriptURLBackup, metadataKeyWindowsStartupScriptURLBackup)
		preparationVarMap["original_startup_script_url"] = *windowsStartupScriptURLBackup
	}
	prepWf, err := daisycommon.ParseWorkflow(upgradePreparationWorkflowPath, preparationVarMap,
		project, zone, scratchBucketGcsPath, oauth, timeout, ce, gcsLogsDisabled,
		cloudLogsDisabled, stdoutLogsDisabled)
	if err != nil {
		return nil, err
	}
	if windowsStartupScriptURLBackup == nil {
		if !windowsStartupScriptURLBackupExists {
			fmt.Printf("\nNo existing '%v' detected. Won't backup it.\n\n", metadataKeyWindowsStartupScriptURL)
		}
		// remove 'backup-script' step
		delete(prepWf.Steps, "backup-script")
		delete(prepWf.Dependencies, "backup-script")
		prepWf.Dependencies["set-script"] = []string{"attach-install-disk"}
	}

	if skipMachineImageBackup {
		// remove 'backup-machine-image' step
		delete(prepWf.Steps, "backup-machine-image")
		delete(prepWf.Dependencies, "backup-machine-image")
		prepWf.Dependencies["backup-os-disk-snapshot"] = []string{"stop-instance"}
	}

	err = daisyutils.RunWorkflowWithCancelSignal(ctx, prepWf)
	return prepWf, err
}

func upgrade(ctx context.Context, workflowPath string, upgradeVarMap map[string]string, project string, zone string,
	scratchBucketGcsPath string, oauth string, timeout string, ce string, gcsLogsDisabled bool,
	cloudLogsDisabled bool, stdoutLogsDisabled bool) (*daisy.Workflow, error) {

	if windowsStartupScriptURLBackup != nil {
		upgradeVarMap["original_startup_script_url"] = *windowsStartupScriptURLBackup
	}

	upgradeWf, err := daisycommon.ParseWorkflow(workflowPath, upgradeVarMap,
		project, zone, scratchBucketGcsPath, oauth, timeout, ce, gcsLogsDisabled,
		cloudLogsDisabled, stdoutLogsDisabled)
	if err != nil {
		return nil, err
	}

	err = daisyutils.RunWorkflowWithCancelSignal(ctx, upgradeWf)
	return upgradeWf, err
}

func cleanup(ctx context.Context, cleanupVarMap map[string]string, project string, zone string,
	scratchBucketGcsPath string, oauth string, timeout string, ce string, gcsLogsDisabled bool,
	cloudLogsDisabled bool, stdoutLogsDisabled bool) (*daisy.Workflow, error) {

	cleanupWf, err := daisycommon.ParseWorkflow(cleanupWorkflowPath, cleanupVarMap,
		project, zone, scratchBucketGcsPath, oauth, timeout, ce, gcsLogsDisabled,
		cloudLogsDisabled, stdoutLogsDisabled)
	if err != nil {
		return nil, err
	}

	err = daisyutils.RunWorkflowWithCancelSignal(ctx, cleanupWf)
	return cleanupWf, err
}

func rollback(ctx context.Context, rollbackVarMap map[string]string, project string, zone string,
	scratchBucketGcsPath string, oauth string, timeout string, ce string, gcsLogsDisabled bool,
	cloudLogsDisabled bool, stdoutLogsDisabled bool) (*daisy.Workflow, error) {

	if windowsStartupScriptURLBackup != nil {
		rollbackVarMap["original_startup_script_url"] = *windowsStartupScriptURLBackup
	}

	rollbackWf, err := daisycommon.ParseWorkflow(rollbackWorkflowPath, rollbackVarMap,
		project, zone, scratchBucketGcsPath, oauth, timeout, ce, gcsLogsDisabled,
		cloudLogsDisabled, stdoutLogsDisabled)
	if err != nil {
		return nil, err
	}

	err = daisyutils.RunWorkflowWithCancelSignal(ctx, rollbackWf)
	return rollbackWf, err
}

func reboot(ctx context.Context, rebootVarMap map[string]string, project string, zone string,
	scratchBucketGcsPath string, oauth string, timeout string, ce string, gcsLogsDisabled bool,
	cloudLogsDisabled bool, stdoutLogsDisabled bool) (*daisy.Workflow, error) {

	rebootWf, err := daisycommon.ParseWorkflow(rebootWorkflowPath, rebootVarMap,
		project, zone, scratchBucketGcsPath, oauth, timeout, ce, gcsLogsDisabled,
		cloudLogsDisabled, stdoutLogsDisabled)

	if err != nil {
		return nil, err
	}
	err = daisyutils.RunWorkflowWithCancelSignal(ctx, rebootWf)
	if err != nil {
		return rebootWf, err
	}
	return nil, nil
}

func needReboot(err error) bool {
	return strings.Contains(err.Error(), "Windows needs to be restarted")
}
