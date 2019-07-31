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

URL="http://metadata/computeMetadata/v1/instance/attributes"
GCS_PATH=$(curl -f -H Metadata-Flavor:Google ${URL}/gcs-path)
LICENSES=$(curl -f -H Metadata-Flavor:Google ${URL}/licenses)

echo "GCEExport: Running export tool."
if [[ -n $LICENSES ]]; then
  gce_export -gcs_path "$GCS_PATH" -disk /dev/sdb -licenses "$LICENSES" -y
else
  gce_export -gcs_path "$GCS_PATH" -disk /dev/sdb -y
fi
if [ $? -ne 0 ]; then
  echo "ExportFailed: Failed to export disk source to ${GCS_PATH}."
  exit 1
fi

echo "ExportSuccess"
sync
