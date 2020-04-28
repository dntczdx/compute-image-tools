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

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/option"
)

// Parameter key shared with external packages
const (
	ClientIDFlagKey = "client_id"
	DefaultTimeout  = "90m"
)

const (
	logPrefix = "[windows-upgrade]"

	metadataKeyWindowsStartupScriptURL       = "windows-startup-script-url"
	metadataKeyWindowsStartupScriptURLBackup = "windows-startup-script-url-backup"

	versionWindows2008r2 = "windows-2008r2"
	versionWindows2012r2 = "windows-2012r2"
)

var (
	supportedSourceOSVersions = map[string]string{versionWindows2008r2: versionWindows2012r2}
	supportedTargetOSVersions = reverseMap(supportedSourceOSVersions)

	upgradeScriptName = map[string]string{versionWindows2008r2: "upgrade_script_2008r2_to_2012r2.ps1"}

	expectedCurrentLicense = map[string]string{versionWindows2008r2: "projects/windows-cloud/global/licenses/windows-server-2008-r2-dc"}
	licenseToAdd           = map[string]string{versionWindows2008r2: "projects/windows-cloud/global/licenses/windows-server-2012-r2-dc-in-place-upgrade"}
)

type derivedVars struct {
	project string
	zone    string

	osDiskURI        string
	osDiskType       string
	osDiskDeviceName string
	osDiskAutoDelete bool

	instanceName           string
	machineImageBackupName string
	osDiskSnapshotName     string
	newOSDiskName          string
	installMediaDiskName   string

	windowsStartupScriptURLBackup       *string
	windowsStartupScriptURLBackupExists bool
}

// Upgrader implements upgrading logic.
type Upgrader struct {
	// Input params
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

	*derivedVars

	ctx context.Context
}

// Run runs upgrade workflow.
func (u *Upgrader) Run() (*daisy.Workflow, error) {
	log.SetPrefix(logPrefix + " ")

	var err error
	u.ctx = context.Background()
	computeClient, err = daisyCompute.NewClient(u.ctx, option.WithCredentialsFile(u.Oauth))
	if err != nil {
		return nil, daisy.Errf("Failed to create GCE client: %v", err)
	}

	err = u.validateParams()
	if err != nil {
		return nil, err
	}

	return u.runUpgradeWorkflow()
}

func (u *Upgrader) runUpgradeWorkflow() (*daisy.Workflow, error) {
	var err error

	// If upgrade failed, run cleanup or rollback before exiting.
	defer func() {
		u.handleFailure(err)
	}()

	guide, err := getUpgradeGuide(u)
	if err != nil {
		return nil, err
	}
	fmt.Print(guide, "\n\n")

	// step 1: preparation - take snapshot, attach install media, backup/set startup script
	fmt.Print("\nPreparing for upgrade...\n\n")
	prepareWf, err := u.prepare()
	if err != nil {
		return prepareWf, err
	}

	// step 2: run upgrade.
	fmt.Print("\nRunning upgrade...\n\n")
	upgradeWf, err := u.upgrade()
	if err == nil {
		return upgradeWf, nil
	}

	// step 3: reboot if necessary.
	if !needReboot(err) {
		return upgradeWf, err
	}
	fmt.Print("\nRebooting...\n\n")
	rebootWf, err := u.reboot()
	if err != nil {
		return rebootWf, err
	}

	// step 4: retry upgrade.
	fmt.Print("\nRunning upgrade...\n\n")
	upgradeWf, err = u.upgrade()
	return upgradeWf, err
}

func (u *Upgrader) handleFailure(err error) {
	if err == nil {
		fmt.Printf("\nSuccessfully upgraded instance '%v' to %v!\n", u.InstanceURI, u.TargetOS)
		// TODO: update the help guide link. b/154838004
		fmt.Printf("\nPlease verify your applications' functionality of " +
			"the instance. If it has a problem and can't be fixed, please manually " +
			"rollback following the guide.\n\n")
		return
	}

	isNewOSDiskAttached := isNewOSDiskAttached(u.project, u.zone, u.instanceName, u.newOSDiskName)
	if u.AutoRollback {
		if isNewOSDiskAttached {
			fmt.Printf("\nFailed to finish upgrading. Rollback to the "+
				"original state from the original OS disk '%v'...\n\n", u.osDiskURI)
			_, err := u.rollback()
			if err != nil {
				fmt.Printf("\nFailed to rollback. Error: %v\n"+
					"Please manually rollback following the guide.\n\n", err)
			} else {
				fmt.Printf("\nRollback to original state is done. Please " +
					"verify whether it works as expected. If not, you may consider " +
					"restoring the whole instance from the machine image.\n\n")
			}
			return
		}
		fmt.Printf("\nNew OS disk hadn't been attached when failure "+
			"happened. No need to rollback. If the instance can't work as expected, "+
			"please verify whether original OS disk %v is attached and whether the "+
			"instance has been started. If necessary, please manually rollback "+
			"following the guide.\n\n", u.osDiskURI)
	} else if isNewOSDiskAttached {
		fmt.Printf("\nFailed to finish upgrading. Please manually " +
			"rollback following the guide.\n\n")
	}
	fmt.Print("\nCleaning up temporary resources...\n\n")
	if _, err := u.cleanup(); err != nil {
		fmt.Printf("\nFailed to cleanup temporary resources: %v\n"+
			"Please follow the guide to manually cleanup.\n\n", err)
	}
}
