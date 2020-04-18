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

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/option"
)

// Parameter key shared with external packages
const (
	ClientIDFlagKey = "client_id"
)

const (
	logPrefix                      = "[windows-upgrade]"
	rebootWorkflowPath             = "daisy_workflows/windows_upgrade/reboot.wf.json"
	upgradePreparationWorkflowPath = "daisy_workflows/windows_upgrade/windows_upgrade_preparation.wf.json"
	cleanupWorkflowPath            = "daisy_workflows/windows_upgrade/cleanup.wf.json"

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

// UpgradeParams contains user input params and validated (transformed) params
type UpgradeParams struct {
	ClientID               string
	InstanceURI            string
	SkipMachineImageBackup bool
	AutoRollback           bool
	SourceOS               string
	TargetOS               string
	ProjectPtr             *string
	Timeout                string
	ScratchBucketGcsPath   string
	Oauth                  string
	Ce                     string
	GcsLogsDisabled        bool
	CloudLogsDisabled      bool
	StdoutLogsDisabled     bool
	CurrentExecutablePath  string

	*validatedParams
}

// Run runs upgrade workflow.
func Run(upgradeParams *UpgradeParams) (*daisy.Workflow, error) {
	log.SetPrefix(logPrefix + " ")

	var err error
	ctx := context.Background()
	computeClient, err = daisyCompute.NewClient(ctx, option.WithCredentialsFile(upgradeParams.Oauth))
	if err != nil {
		return nil, daisy.Errf("Failed to create GCE client: %v", err)
	}

	err = validateParams(upgradeParams)
	if err != nil {
		return nil, err
	}

	return runUpgradeWorkflow(ctx, upgradeParams)
}

func runUpgradeWorkflow(ctx context.Context, params *UpgradeParams) (*daisy.Workflow, error) {
	var err error
	workflowPath := path.ToWorkingDir(upgradeWorkflowPath[params.SourceOS], params.CurrentExecutablePath)
	retryWorkflowPath := path.ToWorkingDir(retryUpgradeWorkflowPath[params.SourceOS], params.CurrentExecutablePath)
	suffix := path.RandString(8)
	machineImageBackupName := fmt.Sprintf("backup-machine-image-%v", suffix)
	osDiskSnapshotName := fmt.Sprintf("win-upgrade-os-disk-snapshot-%v", suffix)
	newOSDiskName := fmt.Sprintf("windows-upgraded-os-disk-%v", suffix)
	installMediaDiskName := fmt.Sprintf("windows-install-media-%v", suffix)

	preparationVarMap := buildDaisyVarsForPreparation(params.project, params.zone, params.InstanceURI, machineImageBackupName,
		osDiskSnapshotName, newOSDiskName, installMediaDiskName, upgradeScriptName[params.SourceOS],
		params.SourceOS, params.osDisk, params.osDiskType, params.osDiskDeviceName, params.osDiskAutoDelete)
	upgradeVarMap := buildDaisyVarsForUpgrade(params.project, params.zone, params.InstanceURI, installMediaDiskName)
	rebootVarMap := buildDaisyVarsForReboot(params.InstanceURI)

	// If upgrade failed, run cleanup/rollback before exiting.
	defer func() {
		if err == nil {
			fmt.Printf("\nSuccessfully upgraded instance '%v' to %v!\n", params.InstanceURI, params.TargetOS)
			fmt.Printf("\nPlease verify the functionality of the instance. If " +
				"it has a problem and can't be fixed, please manually rollback following the guide.\n\n")
			return
		}

		isNewOSDiskAttached := isNewOSDiskAttached(params.project, params.zone, params.instanceName, newOSDiskName)
		if params.AutoRollback {
			if isNewOSDiskAttached {
				fmt.Printf("\nFailed to finish upgrading. Rollback to original state from original OS disk '%v'...\n\n", params.osDisk)
				_, err := rollback(ctx, params, installMediaDiskName, newOSDiskName)
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
				"and whether the instance has been started. If necessary, please manually rollback following the guide.\n\n", params.osDisk)
		} else {
			if isNewOSDiskAttached {
				fmt.Printf("\nFailed to finish upgrading. Please manually rollback following the guide.\n\n")
			}
		}
		fmt.Print("\nCleaning up temporary resources...\n\n")
		if _, err := cleanup(ctx, upgradeVarMap, params); err != nil {
			fmt.Printf("\nFailed to cleanup temporary resources: %v\n"+
				"Please follow the guide to manually cleanup.\n\n", err)
		}
	}()

	fmt.Printf("%v\n\n", getUpgradeIntroduction(params.project, params.zone, getResourceRealName(params.InstanceURI),
		installMediaDiskName, osDiskSnapshotName, getResourceRealName(params.osDisk), newOSDiskName,
		machineImageBackupName, windowsStartupScriptURLBackup, params.osDiskDeviceName, params.osDiskAutoDelete))

	// step 1: preparation - take snapshot, attach install media, backup/set startup script
	fmt.Print("\nPreparing for upgrade...\n\n")
	prepareWf, err := prepare(ctx, preparationVarMap, params)
	if err != nil {
		return prepareWf, err
	}

	// step 2: run upgrade.
	fmt.Print("\nRunning upgrade...\n\n")
	upgradeWf, err := upgrade(ctx, workflowPath, upgradeVarMap, params)
	if err == nil {
		return upgradeWf, nil
	}

	// step 3: reboot if necessary.
	if !needReboot(err) {
		return upgradeWf, err
	}
	fmt.Print("\nRebooting...\n\n")
	rebootWf, err := reboot(ctx, rebootVarMap, params)
	if err != nil {
		return rebootWf, err
	}

	// step 4: retry upgrade.
	fmt.Print("\nRunning upgrade...\n\n")
	upgradeWf, err = upgrade(ctx, retryWorkflowPath, upgradeVarMap, params)
	return upgradeWf, err
}
