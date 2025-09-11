@echo off

rem Copyright 2025 Synnax Labs, Inc.
rem
rem Use of this software is governed by the Business Source License included in the file
rem licenses/BSL.txt.
rem
rem As of the Change Date specified in that file, in accordance with the Business Source
rem License, use of this software will be governed by the Apache License, Version 2.0,
rem included in the file licenses/APL.txt.

echo 📦 Installing Playwright browsers on Windows...
echo Current directory: %CD%

rem Change to the integration directory if not already there
if not exist "pyproject.toml" (
    echo Changing to integration directory...
    cd integration
    echo New directory: %CD%
)

rem Try to find Poetry executable
set "POETRY_CMD=poetry"
poetry --version >nul 2>nul
if %errorlevel% neq 0 (
    echo Poetry not found in PATH, searching for executable...
    if exist "%APPDATA%\Python\Scripts\poetry.exe" (
        set "POETRY_CMD=%APPDATA%\Python\Scripts\poetry.exe"
        echo Found Poetry at: %POETRY_CMD%
    ) else if exist "%APPDATA%\pypoetry\venv\Scripts\poetry.exe" (
        set "POETRY_CMD=%APPDATA%\pypoetry\venv\Scripts\poetry.exe"
        echo Found Poetry at: %POETRY_CMD%
    ) else if exist "%USERPROFILE%\.local\bin\poetry.exe" (
        set "POETRY_CMD=%USERPROFILE%\.local\bin\poetry.exe"
        echo Found Poetry at: %POETRY_CMD%
    ) else (
        echo ❌ Poetry executable not found
        exit /b 1
    )
)

rem Ensure Playwright package is installed first
echo Verifying Playwright package installation...
"%POETRY_CMD%" run python -c "import playwright; print('Playwright package found:', playwright.__version__)"
if %errorlevel% neq 0 (
    echo ❌ Playwright package not found, reinstalling dependencies...
    "%POETRY_CMD%" install
    if %errorlevel% neq 0 (
        echo ❌ Failed to install dependencies
        exit /b %errorlevel%
    )
)

rem Verify sync_api module specifically
echo Verifying Playwright sync_api module...
"%POETRY_CMD%" run python -c "from playwright.sync_api import sync_playwright; print('Playwright sync_api module found')"
if %errorlevel% neq 0 (
    echo ❌ Playwright sync_api module not found
    exit /b %errorlevel%
)

rem Install Playwright browsers (uses cache by default)
echo Installing Playwright browsers with cache...
echo Using Poetry command: %POETRY_CMD%
"%POETRY_CMD%" run playwright install --with-deps
if %errorlevel% neq 0 (
    echo ❌ Playwright browser installation failed with error code %errorlevel%
    exit /b %errorlevel%
)

echo ✅ Playwright browsers installed successfully