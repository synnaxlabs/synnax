@echo off

rem Copyright 2025 Synnax Labs, Inc.
rem
rem Use of this software is governed by the Business Source License included in the file
rem licenses/BSL.txt.
rem
rem As of the Change Date specified in that file, in accordance with the Business Source
rem License, use of this software will be governed by the Apache License, Version 2.0,
rem included in the file licenses/APL.txt.

echo Checking for existing synnax processes on Windows...

taskkill /F /IM "synnax.exe" 2>nul && echo Killed synnax.exe processes || echo No synnax.exe processes found
taskkill /F /IM "synnax-driver.exe" 2>nul && echo Killed synnax-driver.exe processes || echo No synnax-driver.exe processes found

echo Cleaning up synnax directories...
if exist "%USERPROFILE%\synnax-binaries" rmdir /s /q "%USERPROFILE%\synnax-binaries" && echo Removed synnax-binaries directory from %USERPROFILE% || echo No synnax-binaries directory found in %USERPROFILE%
if exist "%USERPROFILE%\synnax-data" rmdir /s /q "%USERPROFILE%\synnax-data" && echo Removed synnax-data directory from %USERPROFILE% || echo No synnax-data directory found in %USERPROFILE%

rem Clean up any existing binaries directory (current working directory)
if exist ".\binaries" (
    echo 🧹 Cleaning existing binaries directory...
    rmdir /s /q ".\binaries"
) else (
    echo 🧹 No existing binaries directory to clean
)

rem Also clean up synnax-binaries if it exists (for consistency with other OS patterns)
if exist "%USERPROFILE%\synnax-binaries" (
    echo 🧹 Cleaning synnax-binaries directory from %USERPROFILE%...
    rmdir /s /q "%USERPROFILE%\synnax-binaries"
) else (
    echo 🧹 No synnax-binaries directory found in %USERPROFILE%
)

rem Clean up any existing desktop binaries
echo Cleaning up desktop binaries...
del /q "%USERPROFILE%\Desktop\synnax*.exe" 2>nul
echo Desktop cleanup completed

rem Verify cleanup was successful
echo Verifying cleanup...
set "cleanup_failed=false"

if exist ".\binaries" (
    echo ❌ ERROR: binaries directory still exists after cleanup
    set "cleanup_failed=true"
)

if exist "%USERPROFILE%\synnax-binaries" (
    echo ❌ ERROR: synnax-binaries directory still exists in %USERPROFILE% after cleanup
    set "cleanup_failed=true"
)

if exist "%USERPROFILE%\synnax-data" (
    echo ❌ ERROR: synnax-data directory still exists in %USERPROFILE% after cleanup
    set "cleanup_failed=true"
)

rem Check if any desktop synnax executables still exist
dir "%USERPROFILE%\Desktop\synnax*.exe" >nul 2>nul
if %errorlevel% equ 0 (
    echo ❌ ERROR: synnax executables still exist on desktop after cleanup:
    dir "%USERPROFILE%\Desktop\synnax*.exe"
    set "cleanup_failed=true"
)

if "%cleanup_failed%"=="true" (
    echo ❌ Cleanup verification failed
    exit /b 1
) else (
    echo ✅ Cleanup verification passed - all directories and files removed
)

echo ✅ Synnax process cleanup completed
exit /b 0