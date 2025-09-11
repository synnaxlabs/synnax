@echo off

rem Copyright 2025 Synnax Labs, Inc.
rem
rem Use of this software is governed by the Business Source License included in the file
rem licenses/BSL.txt.
rem
rem As of the Change Date specified in that file, in accordance with the Business Source
rem License, use of this software will be governed by the Apache License, Version 2.0,
rem included in the file licenses/APL.txt.

echo Installing Playwright browsers on Windows...

rem Change to the integration directory
cd integration

rem Ensure Poetry is in PATH
set PATH=%APPDATA%\Python\Scripts;%APPDATA%\pypoetry\venv\Scripts;%PATH%

rem Install Playwright browsers
poetry run playwright install --with-deps
if %errorlevel% neq 0 exit /b %errorlevel%

echo Playwright browsers installed successfully