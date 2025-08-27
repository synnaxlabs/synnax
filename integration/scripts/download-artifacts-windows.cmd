@echo off

rem Copyright 2025 Synnax Labs, Inc.
rem
rem Use of this software is governed by the Business Source License included in the file
rem licenses/BSL.txt.
rem
rem As of the Change Date specified in that file, in accordance with the Business Source
rem License, use of this software will be governed by the Apache License, Version 2.0,
rem included in the file licenses/APL.txt.

rem download-artifacts-windows.cmd
rem Downloads build artifacts for Windows platform and sets up binaries
rem Supports both current-run artifacts and reference-run artifacts
rem Used by GitHub Actions workflow: test.integration.yaml

echo 💻 Setting up Windows artifacts download...

rem Setup GitHub CLI (Windows)
echo 🔧 Setting up GitHub CLI...
where gh >nul 2>nul && (
    echo GitHub CLI already installed
    gh --version
) || (
    echo Installing GitHub CLI via Chocolatey...
    choco install gh -y
    refreshenv
    gh --version
)

rem Check build mode and download appropriate artifacts  
if "%SKIP_BUILD%"=="true" (
    if defined REFERENCE_RUN_ID (
        echo 🔄 Skip build mode: using reference run %REFERENCE_RUN_ID%
        echo 📥 Downloading artifacts from reference run: %REFERENCE_RUN_ID%
        
        rem Create binaries directory
        if not exist ".\binaries" mkdir ".\binaries"
        
        rem Download artifacts using GitHub CLI
        echo Downloading driver-windows artifact...
        gh run download %REFERENCE_RUN_ID% --name driver-windows --dir .\binaries
        
        echo Downloading synnax-server-windows artifact...
        gh run download %REFERENCE_RUN_ID% --name synnax-server-windows --dir .\binaries
        
        echo ✅ Reference artifacts downloaded successfully
    ) else (
        echo ❌ Error: SKIP_BUILD is true but no REFERENCE_RUN_ID provided
        exit /b 1
    )
) else (
    echo 📦 Build mode: downloading current run artifacts
    
    rem Create binaries directory
    if not exist ".\binaries" mkdir ".\binaries"
    
    rem Download current run artifacts
    echo Downloading driver-windows artifact...
    gh run download --name driver-windows --dir .\binaries
    
    echo Downloading synnax-server-windows artifact...
    gh run download --name synnax-server-windows --dir .\binaries
    
    echo ✅ Current run artifacts downloaded successfully
)

rem Setup Binaries (Windows)
echo 📦 Setting up binaries...
if not exist "%USERPROFILE%\Desktop" mkdir "%USERPROFILE%\Desktop"
copy /Y ".\binaries\driver.exe" "%USERPROFILE%\Desktop\synnax-driver.exe"
for %%f in (.\binaries\synnax-*-windows.exe) do copy /Y "%%f" "%USERPROFILE%\Desktop\synnax.exe"

echo Binaries prepared in Desktop:
dir "%USERPROFILE%\Desktop\synnax*"

echo ✅ Windows artifacts setup completed successfully