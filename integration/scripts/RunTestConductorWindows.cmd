@echo off

rem Copyright 2025 Synnax Labs, Inc.
rem
rem Use of this software is governed by the Business Source License included in the file
rem licenses/BSL.txt.
rem
rem As of the Change Date specified in that file, in accordance with the Business Source
rem License, use of this software will be governed by the Apache License, Version 2.0,
rem included in the file licenses/APL.txt.

rem RunTestConductorWindows.cmd
rem Runs the test conductor on Windows using CMD
rem Used by GitHub Actions workflow: test.integration.yaml

rem SY-2922

echo Running test conductor on Windows...

rem Debug: Show initial working directory
echo Initial directory: %CD%

rem Set Poetry PATH
set PATH=%APPDATA%\Python\Scripts;%APPDATA%\pypoetry\venv\Scripts;%PATH%

rem Change to integration directory first
cd integration

rem Debug: Show current directory and check for pyproject.toml
echo Current directory: %CD%
if exist pyproject.toml (
    echo Found pyproject.toml
) else (
    echo WARNING: pyproject.toml not found
)

rem Test Playwright import
echo Testing Playwright import...
poetry run python -c "import sys; print('Python executable:', sys.executable)"
poetry run python -c "from playwright.sync_api import sync_playwright; print('Playwright sync_api: SUCCESS')"

rem Debug: Show Poetry environment info
echo Getting Poetry environment info for debugging...
poetry env info

rem Run test conductor
poetry run test-conductor --name test-conductor-win
if %errorlevel% neq 0 exit /b %errorlevel%

echo Test conductor completed