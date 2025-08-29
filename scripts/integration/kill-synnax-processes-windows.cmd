@echo off

rem Copyright 2025 Synnax Labs, Inc.
rem
rem Use of this software is governed by the Business Source License included in the file
rem licenses/BSL.txt.
rem
rem As of the Change Date specified in that file, in accordance with the Business Source
rem License, use of this software will be governed by the Apache License, Version 2.0,
rem included in the file licenses/APL.txt.

rem kill-synnax-processes-windows.cmd
rem Forcibly terminates existing Synnax processes on Windows
rem Used by GitHub Actions workflow: test.integration.yaml

echo Checking for existing synnax processes on Windows...

taskkill /F /IM "synnax.exe" 2>nul && echo Killed synnax.exe processes || echo No synnax.exe processes found
taskkill /F /IM "synnax-driver.exe" 2>nul && echo Killed synnax-driver.exe processes || echo No synnax-driver.exe processes found

echo ✅ Synnax process cleanup completed
exit /b 0