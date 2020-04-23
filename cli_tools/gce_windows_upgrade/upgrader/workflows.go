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

	daisyutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	computeBeta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

func (u *Upgrader) prepare(ctx context.Context) (*daisy.Workflow, error) {
	w := daisy.New()
	w.Name = "windows-upgrade-preparation"
	w.DefaultTimeout = u.Timeout
	w.Sources = map[string]string{"upgrade_script.ps1": fmt.Sprintf("./%v", upgradeScriptName[u.SourceOS])}


	stepStopInstance, err := newStep(w, "stop-instance")
	if err != nil {
		return w, err
	}
	stepStopInstance.StopInstances = &daisy.StopInstances{
		Instances: []string{u.InstanceURI},
	}
	prevStep := stepStopInstance

	if !u.SkipMachineImageBackup {
		stepBackupMachineImage, err := newStep(w, "backup-machine-image", stepStopInstance)
		if err != nil {
			return w, err
		}
		stepBackupMachineImage.CreateMachineImages = &daisy.CreateMachineImages{
			&daisy.MachineImage{
				MachineImage: computeBeta.MachineImage{
					Name: u.machineImageBackupName,
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

	stepBackupOSDiskSnapshot, err := newStep(w, "backup-os-disk-snapshot", prevStep)
	if err != nil {
		return w, err
	}
	stepBackupOSDiskSnapshot.CreateSnapshots = &daisy.CreateSnapshots{
		&daisy.Snapshot{
			Snapshot: compute.Snapshot{
				Name: u.osDiskSnapshotName,
				SourceDisk: u.osDiskURI,
			},
			Resource: daisy.Resource{
				ExactName: true,
				NoCleanup: true,
			},
		},
	}

	stepCreateNewOSDisk, err := newStep(w, "create-new-os-disk", stepBackupOSDiskSnapshot)
	if err != nil {
		return w, err
	}
	stepCreateNewOSDisk.CreateDisks = &daisy.CreateDisks{
		&daisy.Disk{
			Disk: compute.Disk{
				Name: u.newOSDiskName,
				Zone: u.zone,
				Type: u.osDiskType,
				SourceSnapshot: u.osDiskSnapshotName,
				Licenses: []string{licenseToAdd[u.SourceOS]},
			},
			Resource: daisy.Resource{
				ExactName: true,
				NoCleanup: true,
			},
		},
	}

	stepDetachOldOSDisk, err := newStep(w, "detach-old-os-disk", stepCreateNewOSDisk)
	if err != nil {
		return w, err
	}
	stepDetachOldOSDisk.DetachDisks = &daisy.DetachDisks{
		&daisy.DetachDisk{
			Instance:   u.InstanceURI,
			DeviceName: fmt.Sprintf("projects/%v/zones/%v/devices/%v", u.project, u.zone, u.osDiskDeviceName),
		},
	}

	stepAttachNewOSDisk, err := newStep(w, "attach-new-os-disk", stepDetachOldOSDisk)
	if err != nil {
		return w, err
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

	stepCreateInstallDisk, err := newStep(w, "create-install-disk", stepAttachNewOSDisk)
	if err != nil {
		return w, err
	}
	stepCreateInstallDisk.CreateDisks = &daisy.CreateDisks{
		&daisy.Disk{
			Disk: compute.Disk{
				Name: u.installMediaDiskName,
				Zone: u.zone,
				Type: "pd-ssd",
				SourceImage: "projects/compute-image-tools/global/images/family/windows-install-media",
			},
			Resource: daisy.Resource{
				ExactName: true,
				NoCleanup: true,
			},
		},
	}

	stepAttachInstallDisk, err := newStep(w, "attach-install-disk", stepCreateInstallDisk)
	if err != nil {
		return w, err
	}
	stepAttachInstallDisk.AttachDisks = &daisy.AttachDisks{
		&daisy.AttachDisk{
			Instance:   u.InstanceURI,
			AttachedDisk: compute.AttachedDisk{
				Source:                       u.installMediaDiskName,
				AutoDelete:                   true,
			},
		},
	}
	prevStep = stepAttachInstallDisk

	// 'windows-startup-script-url' exists, backup it
	if u.windowsStartupScriptURLBackup != nil {
		fmt.Printf("\nDetected current '%v', value='%v'. Will backup to '%v'.\n\n", metadataKeyWindowsStartupScriptURL,
			*u.windowsStartupScriptURLBackup, metadataKeyWindowsStartupScriptURLBackup)

		stepBackupScript, err := newStep(w, "backup-script", stepAttachInstallDisk)
		if err != nil {
			return w, err
		}
		stepBackupScript.UpdateInstancesMetadata = &daisy.UpdateInstancesMetadata{
			&daisy.UpdateInstanceMetadata{
				Instance:   u.InstanceURI,
				Metadata: map[string]string{metadataKeyWindowsStartupScriptURLBackup: *u.windowsStartupScriptURLBackup},
			},
		}
		prevStep = stepBackupScript
	} else {
		if !u.windowsStartupScriptURLBackupExists {
			fmt.Printf("\nNo existing '%v' detected. Won't backup it.\n\n", metadataKeyWindowsStartupScriptURL)
		}
	}

	stepSetScript, err := newStep(w, "set-script", prevStep)
	if err != nil {
		return w, err
	}
	stepSetScript.UpdateInstancesMetadata = &daisy.UpdateInstancesMetadata{
		&daisy.UpdateInstanceMetadata{
			Instance:   u.InstanceURI,
			Metadata: map[string]string{metadataKeyWindowsStartupScriptURL: "${SOURCESPATH}/upgrade_script.ps1"},
		},
	}

	setWorkflowAttributes(w, u)
	err = daisyutils.RunWorkflowWithCancelSignal(ctx, w)
	return w, err
}

func upgrade(ctx context.Context, workflowPath string, upgradeVarMap map[string]string,
	u *Upgrader) (*daisy.Workflow, error) {

	if u.windowsStartupScriptURLBackup != nil {
		upgradeVarMap["original_startup_script_url"] = *u.windowsStartupScriptURLBackup
	}

	upgradeWf, err := daisycommon.ParseWorkflow(workflowPath, upgradeVarMap,
		u.project, u.zone, u.ScratchBucketGcsPath, u.Oauth, u.Timeout,
		u.Ce, u.GcsLogsDisabled, u.CloudLogsDisabled, u.StdoutLogsDisabled)
	if err != nil {
		return nil, err
	}

	err = daisyutils.RunWorkflowWithCancelSignal(ctx, upgradeWf)
	return upgradeWf, err
}

func reboot(ctx context.Context, rebootVarMap map[string]string, u *Upgrader) (*daisy.Workflow, error) {
	rebootWf, err := daisycommon.ParseWorkflow(rebootWorkflowPath, rebootVarMap,
		u.project, u.zone, u.ScratchBucketGcsPath, u.Oauth, u.Timeout,
		u.Ce, u.GcsLogsDisabled, u.CloudLogsDisabled, u.StdoutLogsDisabled)

	if err != nil {
		return nil, err
	}
	err = daisyutils.RunWorkflowWithCancelSignal(ctx, rebootWf)
	return rebootWf, err
}

func cleanup(ctx context.Context, cleanupVarMap map[string]string, u *Upgrader) (*daisy.Workflow, error) {
	cleanupWf, err := daisycommon.ParseWorkflow(cleanupWorkflowPath, cleanupVarMap,
		u.project, u.zone, u.ScratchBucketGcsPath, u.Oauth, u.Timeout,
		u.Ce, u.GcsLogsDisabled, u.CloudLogsDisabled, u.StdoutLogsDisabled)
	if err != nil {
		return nil, err
	}

	err = daisyutils.RunWorkflowWithCancelSignal(ctx, cleanupWf)
	return cleanupWf, err
}

func (u *Upgrader) rollback(ctx context.Context) (*daisy.Workflow, error) {
	originalStartupScriptURL := ""
	if u.windowsStartupScriptURLBackup != nil {
		originalStartupScriptURL = *u.windowsStartupScriptURLBackup
	}

	w := daisy.New()
	w.Name = "rollback"
	w.DefaultTimeout = u.Timeout

	stepStopInstance, err := newStep(w, "stop-instance")
	if err != nil {
		return w, err
	}
	stepStopInstance.StopInstances = &daisy.StopInstances{
		Instances: []string{u.InstanceURI},
	}

	stepDetachNewOSDisk, err := newStep(w, "detach-new-os-disk", stepStopInstance)
	if err != nil {
		return w, err
	}
	stepDetachNewOSDisk.DetachDisks = &daisy.DetachDisks{
		{
			Instance:   u.InstanceURI,
			DeviceName: fmt.Sprintf("projects/%v/zones/%v/devices/%v", u.project, u.zone, u.osDiskDeviceName),
		},
	}

	stepAttachOldOSDisk, err := newStep(w, "attach-old-os-disk", stepDetachNewOSDisk)
	if err != nil {
		return w, err
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

	stepDetachInstallMediaDisk, err := newStep(w, "detach-install-media-disk", stepAttachOldOSDisk)
	if err != nil {
		return w, err
	}
	stepDetachInstallMediaDisk.DetachDisks = &daisy.DetachDisks{
		{
			Instance:   u.InstanceURI,
			DeviceName: fmt.Sprintf("projects/%v/zones/%v/devices/%v", u.project, u.zone, u.installMediaDiskName),
		},
	}

	stepRestoreScript, err := newStep(w, "restore-script", stepDetachInstallMediaDisk)
	if err != nil {
		return w, err
	}
	stepRestoreScript.UpdateInstancesMetadata = &daisy.UpdateInstancesMetadata{
		{
			Instance: u.InstanceURI,
			Metadata: map[string]string{
				"windows-startup-script-url": originalStartupScriptURL,
			},
		},
	}

	stepStartInstance, err := newStep(w, "start-instance", stepRestoreScript)
	if err != nil {
		return w, err
	}
	stepStartInstance.StartInstances = &daisy.StartInstances{
		Instances: []string{u.InstanceURI},
	}

	stepDeleteNewOSDisk, err := newStep(w, "delete-new-os-disk", stepStartInstance)
	if err != nil {
		return w, err
	}
	stepDeleteNewOSDisk.DeleteResources = &daisy.DeleteResources{
		Disks: []string{
			fmt.Sprintf("projects/%v/zones/%v/disks/%v", u.project, u.zone, u.newOSDiskName),
		},
	}

	stepDeleteUpgradeDisks, err := newStep(w, "delete-install-media-disk", stepStartInstance)
	if err != nil {
		return w, err
	}
	stepDeleteUpgradeDisks.DeleteResources = &daisy.DeleteResources{
		Disks: []string{
			fmt.Sprintf("projects/%v/zones/%v/disks/%v", u.project, u.zone, u.newOSDiskName),
			fmt.Sprintf("projects/%v/zones/%v/disks/%v", u.project, u.zone, u.installMediaDiskName),
		},
	}

	setWorkflowAttributes(w, u)
	err = daisyutils.RunWorkflowWithCancelSignal(ctx, w)
	return w, err
}

func newStep(w *daisy.Workflow, name string, dependencies ...*daisy.Step) (*daisy.Step, error) {
	s, err := w.NewStep(name)
	if err != nil {
		return nil, err
	}

	err = w.AddDependency(s, dependencies...)
	return s, err
}
