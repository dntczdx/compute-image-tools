#!/bin/bash
# Copyright 2019 Google Inc. All Rights Reserved.
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

# step 1: get all disks attached to this instance
# step 2: set them as "auto delete"
# step 3: sleep timeout+5min and delete self


sleep 20s
echo "GCEExport: You shouldn't see this output since it's executed after timeout: delayed cleaning up..."
NAME=$(curl -X GET http://metadata.google.internal/computeMetadata/v1/instance/name -H 'Metadata-Flavor: Google')
ZONE=$(curl -X GET http://metadata.google.internal/computeMetadata/v1/instance/zone -H 'Metadata-Flavor: Google')
DISK=$(curl -X GET http://metadata.google.internal/computeMetadata/v1/instance/disks/ -H 'Metadata-Flavor: Google')
DEVICES=$(curl "http://metadata.google.internal/computeMetadata/v1/instance/disks/?recursive=true&alt=text" -H "Metadata-Flavor: Google" | grep '/device-name ' | sed -e 's/\(.*\/device-name \)*//g')
DEVICES=$(echo $DEVICES)
echo "GCEExport: connecting disks '$DEVICES' with instance '$NAME'"
IFS=$' ' read -r -a DEVICE_ARR <<< "$DEVICES"
for DEVICE in "${DEVICE_ARR[@]}"
do
:
  gcloud --quiet compute instances set-disk-auto-delete $NAME --device-name=$DEVICE --zone=$ZONE
done

gcloud --quiet compute instances delete $NAME --zone=$ZONE
