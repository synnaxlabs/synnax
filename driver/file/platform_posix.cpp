// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <cstdlib>
#include <filesystem>
#include <vector>

#include <pwd.h>
#include <unistd.h>

#include "driver/file/file.h"

namespace driver::file {
std::vector<std::filesystem::path> roots() {
    return {"/"};
}

std::filesystem::path home() {
    if (const char *dir = std::getenv("HOME"); dir != nullptr && *dir != '\0')
        return dir;
    if (const passwd *user = getpwuid(getuid()); user != nullptr) return user->pw_dir;
    return {};
}

x::telem::TimeStamp to_timestamp(const std::filesystem::file_time_type time) {
    const auto since_epoch = std::chrono::file_clock::to_sys(time).time_since_epoch();
    return x::telem::TimeStamp(
        std::chrono::duration_cast<std::chrono::nanoseconds>(since_epoch).count()
    );
}
}
