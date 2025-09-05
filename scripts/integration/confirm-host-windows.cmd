@echo off

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