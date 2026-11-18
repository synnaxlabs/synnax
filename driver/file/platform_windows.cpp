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

#include <windows.h>

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
    std::wstring buf(MAX_PATH, L'\0');
    DWORD n = GetEnvironmentVariableW(L"USERPROFILE", buf.data(), buf.size());
    if (n > buf.size()) {
        buf.resize(n);
        n = GetEnvironmentVariableW(L"USERPROFILE", buf.data(), buf.size());
    }
    if (n == 0 || n > buf.size()) return {};
    buf.resize(n);
    return buf;
}

x::telem::TimeStamp to_timestamp(const std::filesystem::file_time_type time) {
    const auto since_epoch = std::chrono::clock_cast<std::chrono::system_clock>(time)
                                 .time_since_epoch();
    return x::telem::TimeStamp(
        std::chrono::duration_cast<std::chrono::nanoseconds>(since_epoch).count()
    );
}
}
