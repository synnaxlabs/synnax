@echo off

rem Copyright 2025 Synnax Labs, Inc.
rem
rem Use of this software is governed by the Business Source License included in the file
rem licenses/BSL.txt.
rem
rem As of the Change Date specified in that file, in accordance with the Business Source
rem License, use of this software will be governed by the Apache License, Version 2.0,
rem included in the file licenses/APL.txt.

echo Host confirmed
echo OS: %OS%
echo Computer Name: %COMPUTERNAME%
echo User Name: %USERNAME%
echo Processor: %PROCESSOR_IDENTIFIER%
echo Architecture: %PROCESSOR_ARCHITECTURE%
echo Number of Processors: %NUMBER_OF_PROCESSORS%

for /f "tokens=2 delims=:" %%i in ('systeminfo ^| find "Total Physical Memory"') do (
    echo Total Physical Memory:%%i
)

for /f "tokens=3" %%i in ('dir /-c %SystemDrive%\ ^| find "bytes free"') do (
    echo Disk Space Available: %%i bytes
)

echo Date: %DATE% %TIME%