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

package imagefile

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/files"
)

// BootInfo includes metadata returned by `guestfs` script.
type BootInfo struct {
	HasUEFIPartition bool
}

// BootInfoClient runs `guestfs` script and returns the results.
type BootInfoClient interface {
	GetInfo(ctx context.Context, filename, workflowDir string) (BootInfo, error)
}

// NewBootInfoClient returns a new instance of BootInfoClient.
func NewBootInfoClient() BootInfoClient {
	return defaultBootInfoClient{}
}

type defaultBootInfoClient struct{}

func (client defaultBootInfoClient) GetInfo(ctx context.Context, filename, workflowDir string) (info BootInfo, err error) {
	fmt.Println(">>>>>detect1")
	if !files.Exists(filename) {
		return info, fmt.Errorf("file %q not found", filename)
	}
	cmd := exec.CommandContext(ctx, "sudo python3", path.Join(workflowDir, "image_import/inspection/inspect_uefi.py"), filename)
	//cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	fmt.Println(">>>>>detecterr:", err)
	fmt.Println(">>>>>detectoutput:", string(output))

	info.HasUEFIPartition = strings.Contains(string(output), "C12A7328-F81F-11D2-BA4B-00A0C93EC93B")
	return info, nil
}
