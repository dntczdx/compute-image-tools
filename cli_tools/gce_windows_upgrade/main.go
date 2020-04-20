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

// gce_windows_upgrade is a tool for upgrading GCE Windows instances.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_windows_upgrade/upgrader"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

var (
	// project/zone/network/subnet/labels are not required when the upgrade has to happen in place.
	clientID               = flag.String(upgrader.ClientIDFlagKey, "", "Identifies the client of the importer, e.g. `gcloud` or `pantheon`.")
	instance               = flag.String("instance", "", "Instance to upgrade, in the form of 'projects/<project>/zones/<zone>/instances/<instance>'.")
	skipMachineImageBackup = flag.Bool("skip-machine-image-backup", false, "Skip backup for the instance. It's not recommended to skip machine image backup if the instance is not already backed up.")
	autoRollback           = flag.Bool("auto-rollback", false, "Rollback automatically if upgrade failed. If enabled, debugging upgrade failure reason will be impossible.")
	sourceOS               = flag.String("source-os", "", fmt.Sprintf("Source OS version of the instance to be upgraded. Supported values: %v", strings.Join(upgrader.SupportedSourceOSVersions(), ", ")))
	targetOS               = flag.String("target-os", "", fmt.Sprintf("Target OS version to upgrade the instance to. Supported values: %v", strings.Join(upgrader.SupportedTargetOSVersions(), ", ")))
	timeout                = flag.String("timeout", "", "Maximum time an upgrade can take before it times out. For example, specifying 2h will fail the process after 2 hours. See $ gcloud topic datetimes for information on duration formats.")
	scratchBucketGcsPath   = flag.String("scratch-bucket-gcs-path", "", "GCS scratch bucket to use, overrides what is set in workflow.")
	oauth                  = flag.String("oauth", "", "Path to oauth json file, overrides what is set in workflow.")
	ce                     = flag.String("compute-endpoint-override", "", "API endpoint to override default.")
	gcsLogsDisabled        = flag.Bool("disable-gcs-logging", false, "Do not stream logs to GCS.")
	cloudLogsDisabled      = flag.Bool("disable-cloud-logging", false, "Do not stream logs to Cloud Logging.")
	stdoutLogsDisabled     = flag.Bool("disable-stdout-logging", false, "Do not display detailed logs on stdout.")

	project = new(string)
)

func upgradeEntry() (*daisy.Workflow, error) {
	currentExecutablePath := string(os.Args[0])
	upgradeParams := &upgrader.UpgradeParams{
		ClientID:               strings.TrimSpace(*clientID),
		InstanceURI:            strings.TrimSpace(*instance),
		SkipMachineImageBackup: *skipMachineImageBackup,
		AutoRollback:           *autoRollback,
		SourceOS:               strings.TrimSpace(*sourceOS),
		TargetOS:               strings.TrimSpace(*targetOS),
		ProjectPtr:             project,
		Timeout:                strings.TrimSpace(*timeout),
		ScratchBucketGcsPath:   strings.TrimSpace(*scratchBucketGcsPath),
		Oauth:                  strings.TrimSpace(*oauth),
		Ce:                     strings.TrimSpace(*ce),
		GcsLogsDisabled:        *gcsLogsDisabled,
		CloudLogsDisabled:      *cloudLogsDisabled,
		StdoutLogsDisabled:     *stdoutLogsDisabled,
		CurrentExecutablePath:  currentExecutablePath,
	}

	return upgrader.Run(upgradeParams)
}

func main() {
	flag.Parse()

	paramLog := service.InputParams{
		WindowsUpgradeParams: &service.WindowsUpgradeParams{
			CommonParams: &service.CommonParams{
				ClientID:                *clientID,
				Timeout:                 *timeout,
				Project:                 *project,
				ObfuscatedProject:       service.Hash(*project),
				ScratchBucketGcsPath:    *scratchBucketGcsPath,
				Oauth:                   *oauth,
				ComputeEndpointOverride: *ce,
				DisableGcsLogging:       *gcsLogsDisabled,
				DisableCloudLogging:     *cloudLogsDisabled,
				DisableStdoutLogging:    *stdoutLogsDisabled,
			},
			Instance:               *instance,
			SkipMachineImageBackup: *skipMachineImageBackup,
			AutoRollback:           *autoRollback,
			SourceOS:               *sourceOS,
			TargetOS:               *targetOS,
		},
	}

	if err := service.RunWithServerLogging(service.WindowsUpgrade, paramLog, project, upgradeEntry); err != nil {
		os.Exit(1)
	}
}
