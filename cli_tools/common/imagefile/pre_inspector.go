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
	"log"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_vm_image_import/importer"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

const (
	workflowFile = "daisy_workflows/image_import/inspection/inspect-disk.wf.json"
)

// Inspector finds partition and boot-related properties for a disk.
type PreInspector interface {
	// Inspect finds partition and boot-related properties for a disk and
	// returns an PreInspectionResult. The reference is implementation specific.
	Inspect(reference string) (PreInspectionResult, error)
}

// PreInspectionResult contains the partition and boot-related properties of a disk.
type PreInspectionResult struct {
	HasUEFIPartition bool
}

// NewPreInspector creates an PreInspector that can inspect GCS image file.
func NewPreInspector(wfAttributes daisycommon.WorkflowAttributes, args importer.ImportArguments) (PreInspector, error) {
	wf, err := daisy.NewFromFile(workflowFile)
	if err != nil {
		return nil, err
	}
	daisycommon.SetWorkflowAttributes(wf, wfAttributes)
	return defaultPreInspector{wf, args}, nil
}

// defaultPreInspector implements disk.Inspector using a Daisy workflow.
type defaultPreInspector struct {
	wf   *daisy.Workflow
	args importer.ImportArguments
}

// Inspect finds partition and boot-related properties for a GCP persistent disk, and
// returns an PreInspectionResult. `reference` is a fully-qualified PD URI, such as
// "projects/project-name/zones/us-central1-a/disks/disk-name".
func (inspector defaultPreInspector) Inspect(reference string) (PreInspectionResult, error) {
	if !inspector.args.Inspect {
		return PreInspectionResult{}, nil
	}

	log.Printf("Running experimental pre inspections on %v.", reference)

	inspector.wf.AddVar("pd_uri", reference)
	err := inspector.wf.Run(context.Background())
	ir := PreInspectionResult{}

	if inspector.wf.GetSerialConsoleOutputValue("has_uefi_partition") == "true" {
		ir.HasUEFIPartition = true
	}

	if err != nil {
		log.Printf("Inspection error=%v", err)
	} else {
		log.Printf("Inspection result=%v", ir)
	}

	return ir, err
}
