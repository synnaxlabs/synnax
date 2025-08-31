@echo off

rem Copyright 2025 Synnax Labs, Inc.
rem
rem Use of this software is governed by the Business Source License included in the file
rem licenses/BSL.txt.
rem
rem As of the Change Date specified in that file, in accordance with the Business Source
rem License, use of this software will be governed by the Apache License, Version 2.0,
rem included in the file licenses/APL.txt.

rem run-test-conductor-windows.cmd
rem Runs the test conductor on Windows using CMD
rem Used by GitHub Actions workflow: test.integration.yaml

echo 🧪 Running test conductor on Windows...

rem Set Poetry PATH (try both common locations)
set PATH=%APPDATA%\Python\Scripts;%APPDATA%\pypoetry\venv\Scripts;%PATH%

rem Change to test directory
cd integration\test\py

rem Run test conductor
poetry run test-conductor --name test-conductor-windows --sequence testcases\basic_tests.json
if %errorlevel% neq 0 exit /b %errorlevel%

echo ✅ Test conductor completed