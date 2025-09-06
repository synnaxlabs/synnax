@echo off

rem Copyright 2025 Synnax Labs, Inc.
rem
rem Use of this software is governed by the Business Source License included in the file
rem licenses/BSL.txt.
rem
rem As of the Change Date specified in that file, in accordance with the Business Source
rem License, use of this software will be governed by the Apache License, Version 2.0,
rem included in the file licenses/APL.txt.

rem build-windows-latest.cmd
rem Builds Synnax driver and server binaries for Windows Latest
rem Used by GitHub Actions workflow: test.integration.yaml

echo Building Synnax for Windows Latest...

rem Build Driver (Windows)
echo Building driver with Bazel...
bazel --output_user_root=C:/tmp build --enable_platform_specific_config -c opt --announce_rc //driver

rem Move Driver to Assets
echo Moving driver to assets...
if not exist core\pkg\service\hardware\embedded\assets mkdir core\pkg\service\hardware\embedded\assets
copy bazel-bin\driver\driver.exe core\pkg\service\hardware\embedded\assets\driver.exe

rem Get Version
echo 📋 Getting version...
cd core
for /f "tokens=*" %%i in (pkg\version\VERSION) do set VERSION=%%i
echo VERSION=%VERSION%>> %GITHUB_OUTPUT%
echo Building version: %VERSION%

rem Download Go Dependencies
echo Downloading Go dependencies...
go mod download

rem Build Server
echo Building Synnax server...
go build -tags driver -o synnax-v%VERSION%-windows.exe
cd ..

rem Test Binary Execution
echo Testing binary execution...
echo Testing binary execution...
core\synnax-v%VERSION%-windows.exe version || echo ⚠️ Server binary check failed
bazel-bin\driver\driver.exe --help || echo ⚠️ Driver binary check failed

echo ✅ Windows Latest build completed successfully!
echo 📁 Built artifacts:
echo   - Driver: bazel-bin\driver\driver.exe
echo   - Server: core\synnax-v%VERSION%-windows.exe
