@echo off

rem Copyright 2025 Synnax Labs, Inc.
rem
rem Use of this software is governed by the Business Source License included in the file
rem licenses/BSL.txt.
rem
rem As of the Change Date specified in that file, in accordance with the Business Source
rem License, use of this software will be governed by the Apache License, Version 2.0,
rem included in the file licenses/APL.txt.

echo 🔍 Build failed - debugging info:

echo === Go version ===
go version 2>nul || echo Go not found

echo === Bazel version ===
bazel version 2>nul || (
    bazelisk version 2>nul || echo Bazel/Bazelisk not found
)

echo === Disk space ===
dir /-c 2>nul || echo Cannot check disk space

echo === Build outputs ===
dir bazel-bin\driver\ 2>nul || echo No driver build output
dir synnax\ | findstr synnax-v 2>nul || echo No server build output

echo 🔍 Debug information collection completed