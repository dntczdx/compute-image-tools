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
	"google.golang.org/api/compute/v1"
)

func prepare(ctx context.Context, preparationVarMap map[string]string, params *UpgradeParams) (*daisy.Workflow, error) {
	// 'windows-startup-script-url' exists, backup it
	if windowsStartupScriptURLBackup != nil {
		fmt.Printf("\nDetected current '%v', value='%v'. Will backup to '%v'.\n\n", metadataKeyWindowsStartupScriptURL,
			*windowsStartupScriptURLBackup, metadataKeyWindowsStartupScriptURLBackup)
		preparationVarMap["original_startup_script_url"] = *windowsStartupScriptURLBackup
	}
	prepWf, err := daisycommon.ParseWorkflow(upgradePreparationWorkflowPath, preparationVarMap,
		params.project, params.zone, params.ScratchBucketGcsPath, params.Oauth, params.Timeout,
		params.Ce, params.GcsLogsDisabled, params.CloudLogsDisabled, params.StdoutLogsDisabled)
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

	if params.SkipMachineImageBackup {
		// remove 'backup-machine-image' step
		delete(prepWf.Steps, "backup-machine-image")
		delete(prepWf.Dependencies, "backup-machine-image")
		prepWf.Dependencies["backup-os-disk-snapshot"] = []string{"stop-instance"}
	}

	err = daisyutils.RunWorkflowWithCancelSignal(ctx, prepWf)
	return prepWf, err
}

func upgrade(ctx context.Context, workflowPath string, upgradeVarMap map[string]string,
	params *UpgradeParams) (*daisy.Workflow, error) {

	if windowsStartupScriptURLBackup != nil {
		upgradeVarMap["original_startup_script_url"] = *windowsStartupScriptURLBackup
	}

	upgradeWf, err := daisycommon.ParseWorkflow(workflowPath, upgradeVarMap,
		params.project, params.zone, params.ScratchBucketGcsPath, params.Oauth, params.Timeout,
		params.Ce, params.GcsLogsDisabled, params.CloudLogsDisabled, params.StdoutLogsDisabled)
	if err != nil {
		return nil, err
	}

	err = daisyutils.RunWorkflowWithCancelSignal(ctx, upgradeWf)
	return upgradeWf, err
}

func reboot(ctx context.Context, rebootVarMap map[string]string, params *UpgradeParams) (*daisy.Workflow, error) {
	rebootWf, err := daisycommon.ParseWorkflow(rebootWorkflowPath, rebootVarMap,
		params.project, params.zone, params.ScratchBucketGcsPath, params.Oauth, params.Timeout,
		params.Ce, params.GcsLogsDisabled, params.CloudLogsDisabled, params.StdoutLogsDisabled)

	if err != nil {
		return nil, err
	}
	err = daisyutils.RunWorkflowWithCancelSignal(ctx, rebootWf)
	return rebootWf, err
}

func cleanup(ctx context.Context, cleanupVarMap map[string]string, params *UpgradeParams) (*daisy.Workflow, error) {

	cleanupWf, err := daisycommon.ParseWorkflow(cleanupWorkflowPath, cleanupVarMap,
		params.project, params.zone, params.ScratchBucketGcsPath, params.Oauth, params.Timeout,
		params.Ce, params.GcsLogsDisabled, params.CloudLogsDisabled, params.StdoutLogsDisabled)
	if err != nil {
		return nil, err
	}

	err = daisyutils.RunWorkflowWithCancelSignal(ctx, cleanupWf)
	return cleanupWf, err
}

func rollback(ctx context.Context, params *UpgradeParams, installMediaDiskName, newOSDiskName string) (*daisy.Workflow, error) {
	originalStartupScriptURL := ""
	if windowsStartupScriptURLBackup != nil {
		originalStartupScriptURL = *windowsStartupScriptURLBackup
	}

	w := &daisy.Workflow{
		Name:           "rollback",
		DefaultTimeout: "30m",
	}

	stepStopInstance, err := newStep(w, "stop-instance")
	if err != nil {
		return w, err
	}
	stepStopInstance.StopInstances = &daisy.StopInstances{
		Instances: []string{params.InstanceURI},
	}

	stepDetachNewOSDisk, err := newStep(w, "detach-new-os-disk", stepStopInstance)
	if err != nil {
		return w, err
	}
	stepDetachNewOSDisk.DetachDisks = &daisy.DetachDisks{
		{
			Instance:   params.InstanceURI,
			DeviceName: fmt.Sprintf("projects/%v/zones/%v/devices/%v", params.project, params.zone, params.osDiskDeviceName),
		},
	}

	stepAttachOldOSDisk, err := newStep(w, "attach-old-os-disk", stepDetachNewOSDisk)
	if err != nil {
		return w, err
	}
	stepAttachOldOSDisk.AttachDisks = &daisy.AttachDisks{
		{
			Instance: params.InstanceURI,
			AttachedDisk: compute.AttachedDisk{
				Source:     params.osDisk,
				DeviceName: params.osDiskDeviceName,
				AutoDelete: params.osDiskAutoDelete,
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
			Instance:   params.InstanceURI,
			DeviceName: fmt.Sprintf("projects/%v/zones/%v/devices/%v", params.project, params.zone, installMediaDiskName),
		},
	}

	stepRestoreScript, err := newStep(w, "restore-script", stepDetachInstallMediaDisk)
	if err != nil {
		return w, err
	}
	stepRestoreScript.UpdateInstancesMetadata = &daisy.UpdateInstancesMetadata{
		{
			Instance: params.InstanceURI,
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
		Instances: []string{params.InstanceURI},
	}

	stepDeleteNewOSDisk, err := newStep(w, "delete-new-os-disk", stepStartInstance)
	if err != nil {
		return w, err
	}
	stepDeleteNewOSDisk.DeleteResources = &daisy.DeleteResources{
		Disks: []string{
			fmt.Sprintf("projects/%v/zones/%v/disks/%v", params.project, params.zone, newOSDiskName),
		},
	}

	stepDeleteUpgradeDisks, err := newStep(w, "delete-install-media-disk", stepStartInstance)
	if err != nil {
		return w, err
	}
	stepDeleteUpgradeDisks.DeleteResources = &daisy.DeleteResources{
		Disks: []string{
			fmt.Sprintf("projects/%v/zones/%v/disks/%v", params.project, params.zone, newOSDiskName),
			fmt.Sprintf("projects/%v/zones/%v/disks/%v", params.project, params.zone, installMediaDiskName),
		},
	}

	setWorkflowAttributes(w, params)
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
