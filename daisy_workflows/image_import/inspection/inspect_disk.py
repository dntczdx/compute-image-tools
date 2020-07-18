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

# Example of communicating a key-value pair back to Daisy:
#

'''
import os

os.system('fdisk -l /dev/sdb -o type')
stream = os.popen('fdisk -l /dev/sdb -o type')
output = stream.read()
output
if "EFI System" in output:
  print("Status: <serial-output key:'has_uefi_partition' value:'true'>")
'''

import os
import sys
import guestfs

# TODO get path from metadata
# TODO get file from metadata
# create mount dir
os.mkdir("temp")
# TODO mount gcsfuse
os.system('gcsfuse path temp')
disk = 'path/file'

# TODO try-catch for guestfs

g = guestfs.GuestFS(python_return_dict=True)
g.add_drive_opts(disk, readonly=1)
g.launch()

part_list = g.part_list('/dev/sda')
for part in part_list:
  guid = g.part_get_gpt_type('/dev/sda', part['part_num'])
  if guid == 'C12A7328-F81F-11D2-BA4B-00A0C93EC93B':
    print("Status: <serial-output key:'has_uefi_partition' value:'true'>")
    break

# unmount gcsfuse
os.system('fusermount -u temp')

print("Success: Done!")
