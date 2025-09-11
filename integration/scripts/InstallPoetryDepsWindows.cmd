@echo off

rem Copyright 2025 Synnax Labs, Inc.
rem
rem Use of this software is governed by the Business Source License included in the file
rem licenses/BSL.txt.
rem
rem As of the Change Date specified in that file, in accordance with the Business Source
rem License, use of this software will be governed by the Apache License, Version 2.0,
rem included in the file licenses/APL.txt.

rem SY-2922

echo 📦 Installing Poetry and dependencies on Windows...

rem Change to the integration directory
cd integration

rem Install Poetry on Windows using PowerShell within CMD
powershell -Command "(Invoke-WebRequest -Uri https://install.python-poetry.org -UseBasicParsing).Content | python -"
if %errorlevel% neq 0 exit /b %errorlevel%

rem Add Poetry to PATH for current session
set PATH=%APPDATA%\Python\Scripts;%PATH%

rem Verify Poetry is available
echo Verifying Poetry installation...
poetry --version
if %errorlevel% neq 0 (
    echo ❌ Poetry not found in PATH, trying alternative location...
    set PATH=%APPDATA%\pypoetry\venv\Scripts;%PATH%
    poetry --version
    if %errorlevel% neq 0 exit /b %errorlevel%
)

rem Install dependencies (GitHub Actions handles caching via setup-python)
poetry env remove --all 2>nul
poetry install
if %errorlevel% neq 0 (
    echo ❌ Poetry install failed, trying without lock file...
    if exist "poetry.lock" del "poetry.lock"
    poetry install
    if %errorlevel% neq 0 exit /b %errorlevel%
)

rem Verify installation
echo Verifying key dependencies...
poetry run python -c "import synnax; import playwright; print('Dependencies verified successfully')"
if %errorlevel% neq 0 (
    echo ❌ Dependency verification failed
    exit /b %errorlevel%
)

echo ✅ Poetry and dependencies installed successfully