#!/bin/bash
# Copyright 2017 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
set -x

delayed_cleanup() {
  set -x
  echo "GCEExport: preparing for cleaning up..."
  local URL="http://metadata.google.internal/computeMetadata/v1/instance"

  # Sleep 10 more min after timeout before trying to do cleanup, because regular cleanup is not
  # triggered from here. This is just the plan B to avoid left artifacts when workflow failed to
  # trigger its auto cleanup.
  local TIMEOUT="$(curl -f -H Metadata-Flavor:Google ${URL}/attributes/timeout)"
  if [[ $? -ne 0 ]]; then
    echo "GCEExport: Failed to get timeout attribute, stopping cleanup preparation."
    return
  fi
  sleep $TIMEOUT
  sleep 600

  echo "GCEExport: You shouldn't see this output since it's executed after timeout: delayed cleaning up..."

  local NAME="$(curl -f -H Metadata-Flavor:Google ${URL}/name)"
  local ZONE="$(curl -f -H Metadata-Flavor:Google ${URL}/zone)"
  local DEVICES="$(curl -H Metadata-Flavor:Google ${URL}/disks/?recursive=true'&'alt=text | grep '/device-name ' | sed -e 's/\(.*\/device-name \)*//g')"
  local DEVICES=$(echo $DEVICES)

  echo "GCEExport: set auto-delete for disks '$DEVICES' with instance '$NAME'"
  IFS=' '
  DEVICE_ARR=($DEVICES)
  for DEVICE in "${DEVICE_ARR[@]}"
  do
  :
    gcloud --quiet compute instances set-disk-auto-delete $NAME --device-name=$DEVICE --zone=$ZONE
  done

  echo "GCEExport: delete instance"
  gcloud --quiet compute instances delete $NAME --zone=$ZONE
}

delayed_cleanup &

BYTES_1GB=1073741824
URL="http://metadata/computeMetadata/v1/instance/attributes"
GS_PATH=$(curl -f -H Metadata-Flavor:Google ${URL}/gcs-path)
FORMAT=$(curl -f -H Metadata-Flavor:Google ${URL}/format)
DISK_RESIZING_MON=$(curl -f -H Metadata-Flavor:Google ${URL}/resizing-script-name)

# Strip gs://
IMAGE_OUTPUT_PATH=${GS_PATH##*//}
# Create dir for output
OUTS_PATH=${IMAGE_OUTPUT_PATH%/*}
mkdir -p "/gs/${OUTS_PATH}"

# Prepare disk size info.
# 1. Disk image size info.
SIZE_BYTES=$(lsblk /dev/sdb --output=size -b | sed -n 2p)
# 2. Round up to the next GB.
SIZE_OUTPUT_GB=$(awk "BEGIN {print int(((${SIZE_BYTES}-1)/${BYTES_1GB}) + 1)}")
# 3. Add 5GB of additional space to max size to prevent the corner case that output
# file is slightly larger than source disk.
MAX_BUFFER_DISK_SIZE_GB=$(awk "BEGIN {print int(${SIZE_OUTPUT_GB} + 5)}")

# Prepare buffer disk.
echo "GCEExport: Initializing buffer disk for qemu-img output..."
mkfs.ext4 /dev/sdc
mount /dev/sdc "/gs/${OUTS_PATH}"
if [[ $? -ne 0 ]]; then
  echo "ExportFailed: Failed to prepare buffer disk by mkfs + mount."
fi

# Fetch disk size monitor script from GCS1
DISK_RESIZING_MON_GCS_PATH=gs://${OUTS_PATH%/*}/sources/${DISK_RESIZING_MON}
DISK_RESIZING_MON_LOCAL_PATH=/gs/${DISK_RESIZING_MON}
echo "GCEExport: Copying disk size monitor script..."
if ! out=$(gsutil cp "${DISK_RESIZING_MON_GCS_PATH}" "${DISK_RESIZING_MON_LOCAL_PATH}" 2>&1); then
  echo "ExportFailed: Failed to copy disk size monitor script. Error: ${out}"
  exit
fi
echo ${out}

echo "GCEExport: Launching disk size monitor in background..."
chmod +x ${DISK_RESIZING_MON_LOCAL_PATH}
${DISK_RESIZING_MON_LOCAL_PATH} ${MAX_BUFFER_DISK_SIZE_GB} &

echo "GCEExport: Exporting disk of size ${SIZE_OUTPUT_GB}GB and format ${FORMAT}."
if ! out=$(qemu-img convert /dev/sdb "/gs/${IMAGE_OUTPUT_PATH}" -p -O $FORMAT 2>&1); then
  echo "ExportFailed: Failed to export disk source to ${GS_PATH} due to qemu-img error: ${out}"
  exit
fi
echo ${out}

echo "GCEExport: Copying output image to target GCS path..."
if ! out=$(gsutil -o GSUtil:parallel_composite_upload_threshold=150M cp "/gs/${IMAGE_OUTPUT_PATH}" "${GS_PATH}" 2>&1); then
  echo "ExportFailed: Failed to copy output image to ${GS_PATH}, error: ${out}"
  exit
fi
echo ${out}

echo "export success"
sync
