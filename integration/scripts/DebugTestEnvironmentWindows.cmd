@echo off

rem Copyright 2025 Synnax Labs, Inc.
rem
rem Use of this software is governed by the Business Source License included in the file
rem licenses/BSL.txt.
rem
rem As of the Change Date specified in that file, in accordance with the Business Source
rem License, use of this software will be governed by the Apache License, Version 2.0,
rem included in the file licenses/APL.txt.

echo 🔍 Test failed - debugging info:
echo Matrix OS: windows

echo === Python/Poetry environment ===
rem Set Poetry PATH (try both common locations)
set PATH=%APPDATA%\Python\Scripts;%APPDATA%\pypoetry\venv\Scripts;%PATH%

python --version 2>nul || echo Python not found
poetry --version 2>nul || echo Poetry not found

echo === Synnax connectivity ===
rem Test port connectivity using PowerShell
powershell -Command "try { $connection = Test-NetConnection -ComputerName localhost -Port 9090 -WarningAction SilentlyContinue; if ($connection.TcpTestSucceeded) { Write-Host '✅ Port 9090 reachable' } else { Write-Host '❌ Port 9090 unreachable' } } catch { Write-Host 'Cannot test port 9090' }"

echo === Service status ===
rem Check for synnax processes
tasklist | findstr synnax 2>nul || echo No synnax processes found

echo === Test artifacts ===
rem List test artifacts
if exist "integration\test\py\*.png" dir /b integration\test\py\*.png 2>nul
if exist "integration\test\py\*.log" dir /b integration\test\py\*.log 2>nul  
if exist "integration\test\py\*.json" dir /b integration\test\py\*.json 2>nul
if not exist "integration\test\py\*.png" if not exist "integration\test\py\*.log" if not exist "integration\test\py\*.json" echo No test artifacts found

echo 🔍 Debug information collection completed