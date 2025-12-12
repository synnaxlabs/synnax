@echo off

rem Copyright 2025 Synnax Labs, Inc.
rem
rem Use of this software is governed by the Business Source License included in the file
rem licenses/BSL.txt.
rem
rem As of the Change Date specified in that file, in accordance with the Business Source
rem License, use of this software will be governed by the Apache License, Version 2.0,
rem included in the file licenses/APL.txt.

rem SY-2814: Simplify this script

echo Setting up Windows artifacts download...

rem Setup GitHub CLI (Windows)
echo Setting up GitHub CLI...

rem Initialize gh_cmd variable
set "gh_cmd=gh"

rem Check multiple ways if GitHub CLI is already available
where gh >nul 2>nul
if %errorlevel% equ 0 (
    echo GitHub CLI already available in PATH
    gh --version
    goto :skip_install
)

rem Check if GitHub CLI exists in common installation paths
if exist "%ProgramFiles%\GitHub CLI\gh.exe" (
    set "gh_cmd=%ProgramFiles%\GitHub CLI\gh.exe"
    echo ✅ GitHub CLI found at Program Files
    "%gh_cmd%" --version
    goto :skip_install
)

if exist "C:\ProgramData\chocolatey\lib\gh\tools\gh.exe" (
    set "gh_cmd=C:\ProgramData\chocolatey\lib\gh\tools\gh.exe"
    echo ✅ GitHub CLI found in Chocolatey installation
    "%gh_cmd%" --version
    goto :skip_install
)

echo 📦 GitHub CLI not found, proceeding with installation...
goto :install_gh

:install_gh
echo 📦 Installing GitHub CLI via Chocolatey...
choco install gh -y --force
rem Note: Chocolatey may return non-zero exit codes even for successful installs
echo ✅ Chocolatey installation command completed

echo Refreshing environment to find GitHub CLI...
call refreshenv.exe 2>nul || echo "refreshenv not available, continuing..."

rem Try to find GitHub CLI in common paths
echo Searching for GitHub CLI in common locations...

if exist "%ProgramFiles%\GitHub CLI\gh.exe" (
    set "gh_cmd=%ProgramFiles%\GitHub CLI\gh.exe"
    echo ✅ Found GitHub CLI at Program Files
    goto :test_gh
)

if exist "C:\ProgramData\chocolatey\lib\gh\tools\gh.exe" (
    set "gh_cmd=C:\ProgramData\chocolatey\lib\gh\tools\gh.exe"
    echo ✅ Found GitHub CLI in Chocolatey lib
    goto :test_gh
)

rem Check if gh is now in PATH after installation
where gh >nul 2>nul
if %errorlevel% equ 0 (
    set "gh_cmd=gh"
    echo ✅ Found GitHub CLI in PATH
    goto :test_gh
)

rem Try refreshing PATH manually and check again
echo Manually refreshing PATH environment...
for /f "tokens=2*" %%A in ('reg query "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment" /v PATH') do set "SYS_PATH=%%B"
for /f "tokens=2*" %%A in ('reg query "HKCU\Environment" /v PATH') do set "USER_PATH=%%B"
set "PATH=%SYS_PATH%;%USER_PATH%"

where gh >nul 2>nul
if %errorlevel% equ 0 (
    set "gh_cmd=gh"
    echo ✅ Found GitHub CLI in refreshed PATH
    goto :test_gh
)

echo ❌ Error: GitHub CLI not found after installation
exit /b 1

:test_gh
"%gh_cmd%" --version || (
    echo ❌ Error: GitHub CLI test failed
    exit /b 1
)
echo ✅ GitHub CLI installation successful

:skip_install

echo ✅ GitHub CLI setup completed

rem Verify GitHub CLI authentication
echo Verifying GitHub CLI authentication...
"%gh_cmd%" auth status
if %errorlevel% neq 0 (
    echo ❌ GitHub CLI authentication failed
    echo Attempting to authenticate using GITHUB_TOKEN...
    set /p="%GH_TOKEN%" <nul | "%gh_cmd%" auth login --with-token
    if %errorlevel% neq 0 (
        echo ❌ Failed to authenticate with GitHub
        exit /b 1
    )
)
echo ✅ GitHub CLI authentication verified

rem Download artifacts from run: %REF_RUN_ID%
echo Downloading artifacts from run: %REF_RUN_ID%

rem Verify the run exists and has artifacts
echo Verifying run %REF_RUN_ID% exists...
"%gh_cmd%" run view %REF_RUN_ID% --repo synnaxlabs/synnax
if %errorlevel% neq 0 (
    echo ❌ Error: Cannot access run %REF_RUN_ID%
    exit /b 1
)

rem Create binaries directory
if not exist ".\binaries" mkdir ".\binaries"

rem Download artifacts using GitHub CLI
echo "Downloading windows-core artifact..."
"%gh_cmd%" run download %REF_RUN_ID% --name windows-core --dir .\binaries --repo synnaxlabs/synnax

rem Check both exit code and if files were actually downloaded
if %errorlevel% neq 0 (
    echo ❌ Error: GitHub CLI returned error code %errorlevel%
    echo ❌ Debug: gh_cmd=%gh_cmd%, REF_RUN_ID=%REF_RUN_ID%
    echo ❌ This is a critical failure - cannot proceed without artifacts
    exit /b 1
)

rem Verify the binaries directory was created and contains files
if not exist ".\binaries" (
    echo ❌ Error: Binaries directory was not created - download likely failed
    echo ❌ This is a critical failure - cannot proceed without artifacts
    exit /b 1
)

rem Check if any synnax executable was downloaded (binary is named synnax-v{VERSION}.exe)
dir /b .\binaries\synnax-v*.exe >nul 2>nul
if %errorlevel% neq 0 (
    echo ❌ Error: No synnax executable found in binaries directory
    echo Available files in binaries directory:
    dir .\binaries
    echo ❌ This is a critical failure - cannot proceed without artifacts
    exit /b 1
)


rem Setup Binaries (Windows) - Easy access for RDP Session
echo 📦 Setting up binaries...
if not exist "%USERPROFILE%\Desktop" mkdir "%USERPROFILE%\Desktop"

rem Copy the Synnax binary to desktop (binary is named synnax-v{VERSION}.exe)
echo Copying Synnax binary to desktop...
for %%f in (.\binaries\synnax-v*.exe) do (
    echo Found binary: %%f
    copy /Y "%%f" "%USERPROFILE%\Desktop\synnax.exe"
    if %errorlevel% neq 0 (
        echo ❌ Error: Failed to copy %%f to desktop
        exit /b 1
    )
)

rem Verify the binary was copied successfully
if not exist "%USERPROFILE%\Desktop\synnax.exe" (
    echo ❌ Error: synnax.exe not found on desktop after copy
    echo Available files in binaries directory:
    dir .\binaries
    echo Available files on desktop:
    dir "%USERPROFILE%\Desktop\synnax*"
    exit /b 1
)

echo ✅ Synnax binary copied to desktop successfully
dir "%USERPROFILE%\Desktop\synnax*"

echo ✅ Windows artifacts setup completed successfully