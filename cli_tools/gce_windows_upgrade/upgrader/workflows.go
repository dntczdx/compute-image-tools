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

	daisyutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	computeBeta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

var (
	upgradeSteps      = map[string]func(*Upgrader, *daisy.Workflow) error{versionWindows2008r2: populateUpgradeStepsFrom2008r2To2012r2}
	retryUpgradeSteps = map[string]func(*Upgrader, *daisy.Workflow) error{versionWindows2008r2: populateRetryUpgradeStepsFrom2008r2To2012r2}
)

func (u *Upgrader) prepare() (*daisy.Workflow, error) {
	return u.runWorkflowWithSteps("windows-upgrade-preparation", u.Timeout, populatePrepareSteps)
}

func populatePrepareSteps(u *Upgrader, w *daisy.Workflow) error {
	w.Sources = map[string]string{"upgrade_script.ps1": fmt.Sprintf("./%v", upgradeScriptName[u.SourceOS])}

	stepStopInstance, err := daisyutils.NewStep(w, "stop-instance")
	if err != nil {
		return err
	}
	stepStopInstance.StopInstances = &daisy.StopInstances{
		Instances: []string{u.InstanceURI},
	}
	prevStep := stepStopInstance

	if !u.SkipMachineImageBackup {
		stepBackupMachineImage, err := daisyutils.NewStep(w, "backup-machine-image", stepStopInstance)
		if err != nil {
			return err
		}
		stepBackupMachineImage.CreateMachineImages = &daisy.CreateMachineImages{
			&daisy.MachineImage{
				MachineImage: computeBeta.MachineImage{
					Name:           u.machineImageBackupName,
					SourceInstance: u.InstanceURI,
				},
				Resource: daisy.Resource{
					ExactName: true,
					NoCleanup: true,
				},
			},
		}
		prevStep = stepBackupMachineImage
	}

	stepBackupOSDiskSnapshot, err := daisyutils.NewStep(w, "backup-os-disk-snapshot", prevStep)
	if err != nil {
		return err
	}
	stepBackupOSDiskSnapshot.CreateSnapshots = &daisy.CreateSnapshots{
		&daisy.Snapshot{
			Snapshot: compute.Snapshot{
				Name:       u.osDiskSnapshotName,
				SourceDisk: u.osDiskURI,
			},
			Resource: daisy.Resource{
				ExactName: true,
				NoCleanup: true,
			},
		},
	}

	stepCreateNewOSDisk, err := daisyutils.NewStep(w, "create-new-os-disk", stepBackupOSDiskSnapshot)
	if err != nil {
		return err
	}
	stepCreateNewOSDisk.CreateDisks = &daisy.CreateDisks{
		&daisy.Disk{
			Disk: compute.Disk{
				Name:           u.newOSDiskName,
				Zone:           u.zone,
				Type:           u.osDiskType,
				SourceSnapshot: u.osDiskSnapshotName,
				Licenses:       []string{licenseToAdd[u.SourceOS]},
			},
			Resource: daisy.Resource{
				ExactName: true,
				NoCleanup: true,
			},
		},
	}

	stepDetachOldOSDisk, err := daisyutils.NewStep(w, "detach-old-os-disk", stepCreateNewOSDisk)
	if err != nil {
		return err
	}
	stepDetachOldOSDisk.DetachDisks = &daisy.DetachDisks{
		&daisy.DetachDisk{
			Instance:   u.InstanceURI,
			DeviceName: daisyutils.GetDeviceURI(u.project, u.zone, u.osDiskDeviceName),
		},
	}

	stepAttachNewOSDisk, err := daisyutils.NewStep(w, "attach-new-os-disk", stepDetachOldOSDisk)
	if err != nil {
		return err
	}
	stepAttachNewOSDisk.AttachDisks = &daisy.AttachDisks{
		&daisy.AttachDisk{
			Instance: u.InstanceURI,
			AttachedDisk: compute.AttachedDisk{
				Source:     u.newOSDiskName,
				DeviceName: u.osDiskDeviceName,
				AutoDelete: u.osDiskAutoDelete,
				Boot:       true,
			},
		},
	}

	stepCreateInstallDisk, err := daisyutils.NewStep(w, "create-install-disk", stepAttachNewOSDisk)
	if err != nil {
		return err
	}
	stepCreateInstallDisk.CreateDisks = &daisy.CreateDisks{
		&daisy.Disk{
			Disk: compute.Disk{
				Name:        u.installMediaDiskName,
				Zone:        u.zone,
				Type:        "pd-ssd",
				SourceImage: "projects/compute-image-tools/global/images/family/windows-install-media",
			},
			Resource: daisy.Resource{
				ExactName: true,
				NoCleanup: true,
			},
		},
	}

	stepAttachInstallDisk, err := daisyutils.NewStep(w, "attach-install-disk", stepCreateInstallDisk)
	if err != nil {
		return err
	}
	stepAttachInstallDisk.AttachDisks = &daisy.AttachDisks{
		&daisy.AttachDisk{
			Instance: u.InstanceURI,
			AttachedDisk: compute.AttachedDisk{
				Source:     u.installMediaDiskName,
				AutoDelete: true,
			},
		},
	}
	prevStep = stepAttachInstallDisk

	// 'windows-startup-script-url' exists, backup it
	if u.windowsStartupScriptURLBackup != nil {
		fmt.Printf("\nDetected current '%v', value='%v'. Will backup to '%v'.\n\n", metadataKeyWindowsStartupScriptURL,
			*u.windowsStartupScriptURLBackup, metadataKeyWindowsStartupScriptURLBackup)

		stepBackupScript, err := daisyutils.NewStep(w, "backup-script", stepAttachInstallDisk)
		if err != nil {
			return err
		}
		stepBackupScript.UpdateInstancesMetadata = &daisy.UpdateInstancesMetadata{
			&daisy.UpdateInstanceMetadata{
				Instance: u.InstanceURI,
				Metadata: map[string]string{metadataKeyWindowsStartupScriptURLBackup: *u.windowsStartupScriptURLBackup},
			},
		}
		prevStep = stepBackupScript
	} else {
		if !u.windowsStartupScriptURLBackupExists {
			fmt.Printf("\nNo existing startup script (metadata '%v') detected. Won't backup it.\n\n", metadataKeyWindowsStartupScriptURL)
		}
	}

	stepSetScript, err := daisyutils.NewStep(w, "set-script", prevStep)
	if err != nil {
		return err
	}
	stepSetScript.UpdateInstancesMetadata = &daisy.UpdateInstancesMetadata{
		&daisy.UpdateInstanceMetadata{
			Instance: u.InstanceURI,
			Metadata: map[string]string{metadataKeyWindowsStartupScriptURL: "${SOURCESPATH}/upgrade_script.ps1"},
		},
	}
	return nil
}

func (u *Upgrader) upgrade() (*daisy.Workflow, error) {
	return u.runWorkflowWithSteps("upgrade", u.Timeout, upgradeSteps[u.SourceOS])
}

func populateUpgradeStepsFrom2008r2To2012r2(u *Upgrader, w *daisy.Workflow) error {
	cleanupWorkflow, err := u.generateWorkflowWithSteps("cleanup", "10m", populateCleanupSteps)
	if err != nil {
		return nil
	}

	w.Steps = map[string]*daisy.Step{
		"start-instance": {
			StartInstances: &daisy.StartInstances{
				Instances: []string{u.InstanceURI},
			},
		},
		"wait-for-boot": {
			Timeout: "15m",
			WaitForInstancesSignal: &daisy.WaitForInstancesSignal{
				{
					Name: u.InstanceURI,
					SerialOutput: &daisy.SerialOutput{
						Port:         1,
						SuccessMatch: "GCEMetadataScripts: Beginning upgrade startup script.",
					},
				},
			},
		},
		"wait-for-upgrade": {
			WaitForAnyInstancesSignal: &daisy.WaitForAnyInstancesSignal{
				{
					Name: u.InstanceURI,
					SerialOutput: &daisy.SerialOutput{
						Port:         1,
						SuccessMatch: "windows_upgrade_current_version=6.3",
						FailureMatch: []string{"UpgradeFailed:"},
						StatusMatch:  "GCEMetadataScripts:",
					},
				},
				{
					Name: u.InstanceURI,
					SerialOutput: &daisy.SerialOutput{
						Port:         3,
						FailureMatch: []string{"Windows needs to be restarted"},
					},
				},
			},
		},
		"cleanup-temp-resources": {
			IncludeWorkflow: &daisy.IncludeWorkflow{
				Workflow: cleanupWorkflow,
			},
		},
	}
	w.Dependencies = map[string][]string{
		"wait-for-boot":          {"start-instance"},
		"wait-for-upgrade":       {"start-instance"},
		"cleanup-temp-resources": {"wait-for-upgrade"},
	}
	return nil
}

func (u *Upgrader) retryUpgrade() (*daisy.Workflow, error) {
	return u.runWorkflowWithSteps("retry-upgrade", u.Timeout, retryUpgradeSteps[u.SourceOS])
}

func populateRetryUpgradeStepsFrom2008r2To2012r2(u *Upgrader, w *daisy.Workflow) error {
	cleanupWorkflow, err := u.generateWorkflowWithSteps("cleanup", "10m", populateCleanupSteps)
	if err != nil {
		return nil
	}

	w.Steps = map[string]*daisy.Step{
		"wait-for-boot": {
			Timeout: "15m",
			WaitForInstancesSignal: &daisy.WaitForInstancesSignal{
				{
					Name: u.InstanceURI,
					SerialOutput: &daisy.SerialOutput{
						Port:         1,
						SuccessMatch: "GCEMetadataScripts: Beginning upgrade startup script.",
					},
				},
			},
		},
		"wait-for-upgrade": {
			WaitForAnyInstancesSignal: &daisy.WaitForAnyInstancesSignal{
				{
					Name: u.InstanceURI,
					SerialOutput: &daisy.SerialOutput{
						Port:         1,
						SuccessMatch: "windows_upgrade_current_version=6.3",
						FailureMatch: []string{"UpgradeFailed:"},
						StatusMatch:  "GCEMetadataScripts:",
					},
				},
			},
		},
		"cleanup-temp-resources": {
			IncludeWorkflow: &daisy.IncludeWorkflow{
				Workflow: cleanupWorkflow,
			},
		},
	}
	w.Dependencies = map[string][]string{
		"cleanup-temp-resources": {"wait-for-upgrade"},
	}
	return nil
}

func (u *Upgrader) reboot() (*daisy.Workflow, error) {

	return u.runWorkflowWithSteps("reboot", "15m", populateRebootSteps)
}

func populateRebootSteps(u *Upgrader, w *daisy.Workflow) error {
	w.Steps = map[string]*daisy.Step{
		"stop-instance": {
			StopInstances: &daisy.StopInstances{
				Instances: []string{u.InstanceURI},
			},
		},
		"start-instance": {
			StartInstances: &daisy.StartInstances{
				Instances: []string{u.InstanceURI},
			},
		},
	}
	w.Dependencies = map[string][]string{
		"start-instance": {"stop-instance"},
	}
	return nil
}

func (u *Upgrader) cleanup() (*daisy.Workflow, error) {
	return u.runWorkflowWithSteps("cleanup", "10m", populateCleanupSteps)
}

func populateCleanupSteps(u *Upgrader, w *daisy.Workflow) error {
	w.Steps = map[string]*daisy.Step{
		"restore-script": {
			UpdateInstancesMetadata: &daisy.UpdateInstancesMetadata{
				{
					Instance: u.InstanceURI,
					Metadata: map[string]string{
						"windows-startup-script-url": u.getOriginalStartupScriptURL(),
					},
				},
			},
		},
		"detach-install-media-disk": {
			DetachDisks: &daisy.DetachDisks{
				{
					Instance:   u.InstanceURI,
					DeviceName: daisyutils.GetDeviceURI(u.project, u.zone, u.installMediaDiskName),
				},
			},
		},
		"delete-install-media-disk": {
			DeleteResources: &daisy.DeleteResources{
				Disks: []string{
					daisyutils.GetDiskURI(u.project, u.zone, u.installMediaDiskName),
				},
			},
		},
	}
	w.Dependencies = map[string][]string{
		"delete-install-media-disk": {"detach-install-media-disk"},
	}
	return nil
}

func (u *Upgrader) rollback() (*daisy.Workflow, error) {
	return u.runWorkflowWithSteps("rollback", u.Timeout, populateRollbackSteps)
}

func populateRollbackSteps(u *Upgrader, w *daisy.Workflow) error {
	stepStopInstance, err := daisyutils.NewStep(w, "stop-instance")
	if err != nil {
		return err
	}
	stepStopInstance.StopInstances = &daisy.StopInstances{
		Instances: []string{u.InstanceURI},
	}

	stepDetachNewOSDisk, err := daisyutils.NewStep(w, "detach-new-os-disk", stepStopInstance)
	if err != nil {
		return err
	}
	stepDetachNewOSDisk.DetachDisks = &daisy.DetachDisks{
		{
			Instance:   u.InstanceURI,
			DeviceName: daisyutils.GetDeviceURI(u.project, u.zone, u.osDiskDeviceName),
		},
	}

	stepAttachOldOSDisk, err := daisyutils.NewStep(w, "attach-old-os-disk", stepDetachNewOSDisk)
	if err != nil {
		return err
	}
	stepAttachOldOSDisk.AttachDisks = &daisy.AttachDisks{
		{
			Instance: u.InstanceURI,
			AttachedDisk: compute.AttachedDisk{
				Source:     u.osDiskURI,
				DeviceName: u.osDiskDeviceName,
				AutoDelete: u.osDiskAutoDelete,
				Boot:       true,
			},
		},
	}

	stepDetachInstallMediaDisk, err := daisyutils.NewStep(w, "detach-install-media-disk", stepAttachOldOSDisk)
	if err != nil {
		return err
	}
	stepDetachInstallMediaDisk.DetachDisks = &daisy.DetachDisks{
		{
			Instance:   u.InstanceURI,
			DeviceName: daisyutils.GetDeviceURI(u.project, u.zone, u.installMediaDiskName),
		},
	}

	stepRestoreScript, err := daisyutils.NewStep(w, "restore-script", stepDetachInstallMediaDisk)
	if err != nil {
		return err
	}
	stepRestoreScript.UpdateInstancesMetadata = &daisy.UpdateInstancesMetadata{
		{
			Instance: u.InstanceURI,
			Metadata: map[string]string{
				"windows-startup-script-url": u.getOriginalStartupScriptURL(),
			},
		},
	}

	stepStartInstance, err := daisyutils.NewStep(w, "start-instance", stepRestoreScript)
	if err != nil {
		return err
	}
	stepStartInstance.StartInstances = &daisy.StartInstances{
		Instances: []string{u.InstanceURI},
	}

	stepDeleteNewOSDisk, err := daisyutils.NewStep(w, "delete-new-os-disk", stepStartInstance)
	if err != nil {
		return err
	}
	stepDeleteNewOSDisk.DeleteResources = &daisy.DeleteResources{
		Disks: []string{
			daisyutils.GetDiskURI(u.project, u.zone, u.newOSDiskName),
		},
	}

	stepDeleteUpgradeDisks, err := daisyutils.NewStep(w, "delete-install-media-disk", stepStartInstance)
	if err != nil {
		return err
	}
	stepDeleteUpgradeDisks.DeleteResources = &daisy.DeleteResources{
		Disks: []string{
			daisyutils.GetDiskURI(u.project, u.zone, u.newOSDiskName),
			daisyutils.GetDiskURI(u.project, u.zone, u.installMediaDiskName),
		},
	}
	return nil
}

func (u *Upgrader) getOriginalStartupScriptURL() string {
	originalStartupScriptURL := ""
	if u.windowsStartupScriptURLBackup != nil {
		originalStartupScriptURL = *u.windowsStartupScriptURLBackup
	}
	return originalStartupScriptURL
}

func (u *Upgrader) runWorkflowWithSteps(name string, timeout string, populateStepsFunc func(*Upgrader, *daisy.Workflow) error) (*daisy.Workflow, error) {
	w, err := u.generateWorkflowWithSteps(name, timeout, populateStepsFunc)
	if err != nil {
		return w, err
	}

	setWorkflowAttributes(w, u)
	err = daisyutils.RunWorkflowWithCancelSignal(u.ctx, w)
	return w, err
}

func (u *Upgrader) generateWorkflowWithSteps(name string, timeout string, populateStepsFunc func(*Upgrader, *daisy.Workflow) error) (*daisy.Workflow, error) {
	w := daisy.New()
	w.Name = name
	w.DefaultTimeout = timeout
	err := populateStepsFunc(u, w)
	return w, err
}
