// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <fstream>
#include <sstream>

#include "x/cpp/xerrors/errors.h"

namespace fs {
const auto FS_ERROR = xerrors::Error("fs", "");
const xerrors::Error NOT_FOUND = FS_ERROR.sub("not_found");
const xerrors::Error INVALID_PATH = FS_ERROR.sub("invalid_path");
const xerrors::Error PERMISSION_DENIED = FS_ERROR.sub("permission_denied");
const xerrors::Error READ_ERROR = FS_ERROR.sub("read_error");

/// @brief reads the entire contents of a file into a string.
inline std::pair<std::string, xerrors::Error> read_file(const std::string &path) {
    std::ifstream file(path);
    if (!file.is_open())
        return {"", xerrors::Error(NOT_FOUND, "failed to open " + path)};
    std::stringstream buffer;
    buffer << file.rdbuf();
    if (file.bad()) return {"", xerrors::Error(READ_ERROR, "failed to read " + path)};
    return {buffer.str(), xerrors::NIL};
}
}
