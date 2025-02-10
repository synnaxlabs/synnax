// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std.
#include <string>
#include <unistd.h>
#include <array>

namespace xos {
/// @brief resolves hostname of the machine. If the hostname could not be resolved,
/// returns an empty string and false.
inline std::pair<std::string, bool> get_hostname() {
    std::array<char, 256> hostname{};
    bool ok;
#ifdef _WIN32
    DWORD size = hostname.size();
    ok = GetComputerNameA(hostname.data(), &size) == 0;
#else
    ok = gethostname(hostname.data(), hostname.size()) == 0;
#endif
    return {hostname.data(), ok};
}

/// @brief returns the name of the operating system, if the operating system could not
/// be determined, returns "unknown".
inline std::string get_os() {
#if defined(_WIN32) || defined(_WIN64)
    return "windows";
#elif defined(__APPLE__)
    return "macos";
#elif defined(__linux__)
    return "linux";
#else
    return "unknown";
#endif
}
}

