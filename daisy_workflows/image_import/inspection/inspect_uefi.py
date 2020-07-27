#!/usr/bin/env python3
# Copyright 2020 Google Inc. All Rights Reserved.
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

import sys
import guestfs

# exit(0) means UEFI partition.

try:
  if len(sys.argv) < 2:
    print("inspect_uefi: missing image file to inspect", file=sys.stderr)
    sys.exit(1)
  disk = sys.argv[1]

  print("inspect_uefi: 1", file=sys.stderr)
  print("inspect_uefi: disk ", disk, file=sys.stderr)
  g = guestfs.GuestFS(python_return_dict=True)
  print("inspect_uefi: 2", file=sys.stderr)
  g.add_drive_opts(disk, readonly=1)
  print("inspect_uefi: 3", file=sys.stderr)
  g.launch()
  print("inspect_uefi: 4", file=sys.stderr)


  part_list = g.part_list('/dev/sda')
  print("inspect_uefi: 5", file=sys.stderr)
  for part in part_list:
    print("inspect_uefi: 6", file=sys.stderr)
    print("guid: ", g.part_get_gpt_type('/dev/sda', part['part_num']), file=sys.stderr)
    if g.part_get_gpt_type('/dev/sda', part['part_num']) == "C12A7328-F81F-11D2-BA4B-00A0C93EC93B":
      print("inspect_uefi: 7", file=sys.stderr)
      sys.exit(0)

  print("inspect_uefi: 8", file=sys.stderr)
  sys.exit(2)
except Exception as e:
  print("inspect_uefi: failed to get partition guid", file=sys.stderr)
  print("inspect_uefi err:", e, file=sys.stderr)
  sys.exit(3)
