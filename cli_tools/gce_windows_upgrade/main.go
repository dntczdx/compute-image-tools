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
	clientID               = flag.String(upgrader.ClientIDFlagKey, "", "Identifies the upgrade client. Set to `gcloud`, `api` or `pantheon`.")
	project                = flag.String("project", "", "Project containing the instance to upgrade. Can be overridden if the -instance parameter specifies a different project.")
	zone                   = flag.String("zone", "", "Zone containing the instance to upgrade. Can be overridden if the -instance parameter specifies a different zone.")
	instance               = flag.String("instance", "", "Instance to upgrade. Can be either the instance name or the project path to the instance, for example: 'projects/<project>/zones/<zone>/instances/<instance>'.")
	skipMachineImageBackup = flag.Bool("skip-machine-image-backup", false, "Set to false to generate a backup for the instance. Set to true to prevent the instance from being backed up. True is not recommended unless you already have a backup for the instance.")
	autoRollback           = flag.Bool("auto-rollback", false, "Set to false to retain the OS disk after a failed upgrade. Set to true to delete the OS disk that failed to upgrade; setting to true can make debugging more difficult.")
	sourceOS               = flag.String("source-os", "", fmt.Sprintf("OS version of the source instance to upgrade. Supported values: %v", strings.Join(upgrader.SupportedSourceOSVersions(), ", ")))
	targetOS               = flag.String("target-os", "", fmt.Sprintf("Version of the OS after upgrade. Supported values: %v", strings.Join(upgrader.SupportedTargetOSVersions(), ", ")))
	timeout                = flag.String("timeout", "", "Maximum time limit for an upgrade. For example, if you specify 2h, the upgrade fails after two hours. For information about time duration formats, see $ gcloud topic datetimes.")
	scratchBucketGcsPath   = flag.String("scratch-bucket-gcs-path", "", "Scratch GCS bucket. This setting overrides the workflow setting.")
	oauth                  = flag.String("oauth", "", "Path to OAuth .json file. This setting overrides the workflow setting.")
	ce                     = flag.String("compute-endpoint-override", "", "API endpoint. This setting overrides the default API endpoint setting.")
	gcsLogsDisabled        = flag.Bool("disable-gcs-logging", false, "Set to false to prevent logs from streaming to GCS.")
	cloudLogsDisabled      = flag.Bool("disable-cloud-logging", false, "Set to false to prevent logs from streaming to Cloud Logging.")
	stdoutLogsDisabled     = flag.Bool("disable-stdout-logging", false, "Set to false to disable detailed stdout information.")
)

func upgradeEntry() (*daisy.Workflow, error) {
	currentExecutablePath := string(os.Args[0])
	p := &upgrader.InputParams{
		ClientID:               strings.TrimSpace(*clientID),
		Instance:               strings.TrimSpace(*instance),
		SkipMachineImageBackup: *skipMachineImageBackup,
		AutoRollback:           *autoRollback,
		SourceOS:               strings.TrimSpace(*sourceOS),
		TargetOS:               strings.TrimSpace(*targetOS),
		ProjectPtr:             project,
		Zone:                   *zone,
		Timeout:                strings.TrimSpace(*timeout),
		ScratchBucketGcsPath:   strings.TrimSpace(*scratchBucketGcsPath),
		Oauth:                  strings.TrimSpace(*oauth),
		Ce:                     strings.TrimSpace(*ce),
		GcsLogsDisabled:        *gcsLogsDisabled,
		CloudLogsDisabled:      *cloudLogsDisabled,
		StdoutLogsDisabled:     *stdoutLogsDisabled,
		CurrentExecutablePath:  currentExecutablePath,
	}

	return upgrader.Run(p)
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
				Zone:                    *zone,
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
