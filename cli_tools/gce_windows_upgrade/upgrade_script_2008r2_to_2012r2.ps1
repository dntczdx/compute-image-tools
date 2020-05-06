#  Copyright 2020 Google Inc. All Rights Reserved.
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

$script:install_media_drive = ''

try {
  Write-Host 'GCEMetadataScripts: Beginning upgrade startup script.'

  # cleanup garbages from a failed upgrade to unblock a new upgrade
  Remove-Item 'C:\$WINDOWS.~BT' -Recurse -ErrorAction Ignore

  $ver=[System.Environment]::OSVersion.Version
  Write-Host "windows_upgrade_current_version=$($ver.Major).$($ver.Minor)"
  if ("$($ver.Major).$($ver.Minor)" -ne "6.1") {
    if ("$($ver.Major).$($ver.Minor)" -eq "6.3") {
      Write-Host "GCEMetadataScripts: It's already windows 2012r2!"
      Return
    }
    throw "It's not windows 2008r2!"
  }

  # find the drive which contains install media
  $Drive_Letters = Get-WmiObject Win32_LogicalDisk
  ForEach ($DriveLetter in $Drive_Letters.DeviceID) {
    if (Test-Path "$($DriveLetter)\Windows_Svr_Std_and_DataCtr_2012_R2_64Bit_English") {
      $script:install_media_drive = "$($DriveLetter)"
    }
  }
  if (!$script:install_media_drive) {
    throw "No install media found."
  }
  Write-Host "Detected install media folder drive letter: $script:install_media_drive"

  # run upgrade script
  Set-ExecutionPolicy Unrestricted
  Set-Location "$($script:install_media_drive)/Windows_Svr_Std_and_DataCtr_2012_R2_64Bit_English"
  ./upgrade.ps1
}
catch {
  Write-Host "UpgradeFailed: $($_.Exception.Message)"
  exit 1
}
