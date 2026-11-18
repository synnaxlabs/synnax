// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <filesystem>
#include <string>
#include <vector>

// clang-format off
// shlobj.h must be included after windows.h.
#include <windows.h>
#include <shlobj.h>
// clang-format on

#include "driver/file/file.h"

namespace driver::file {
std::vector<std::filesystem::path> roots() {
    std::vector<std::filesystem::path> out;
    const DWORD drives = GetLogicalDrives();
    for (int i = 0; i < 26; i++)
        if ((drives & (1U << i)) != 0)
            out.emplace_back(std::string(1, static_cast<char>('A' + i)) + ":\\");
    return out;
}

std::filesystem::path home() {
    PWSTR dir = nullptr;
    const auto result = SHGetKnownFolderPath(FOLDERID_Profile, 0, nullptr, &dir);
    const auto path = SUCCEEDED(result) ? std::filesystem::path(dir)
                                        : std::filesystem::path();
    CoTaskMemFree(dir);
    return path;
}

x::telem::TimeStamp to_timestamp(const std::filesystem::file_time_type time) {
    const auto since_epoch = std::chrono::clock_cast<std::chrono::system_clock>(time)
                                 .time_since_epoch();
    return x::telem::TimeStamp(
        std::chrono::duration_cast<std::chrono::nanoseconds>(since_epoch).count()
    );
}
}
