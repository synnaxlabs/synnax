// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "x/cpp/errors/errors.h"

namespace x::fs {
const auto FS_ERROR = errors::Error("fs", "");
const errors::Error NOT_FOUND = FS_ERROR.sub("not_found");
const errors::Error INVALID_PATH = FS_ERROR.sub("invalid_path");
const errors::Error PERMISSION_DENIED = FS_ERROR.sub("permission_denied");
const errors::Error READ_ERROR = FS_ERROR.sub("read_error");

/// @brief an internal method for reading the entire contents of certificate files
/// into a string.
inline std::pair<std::string, errors::Error> read_file(const std::string &path) {
    std::string data;
    FILE *f = fopen(path.c_str(), "r");
    if (f == nullptr) return {data, errors::Error(NOT_FOUND, "failed to open " + path)};
    char buf[1024];
    for (;;) {
        const size_t n = fread(buf, 1, sizeof(buf), f);
        if (n <= 0) break;
        data.append(buf, n);
    }
    if (ferror(f)) return {"", errors::Error(READ_ERROR, "failed to read " + path)};
    fclose(f);
    return {data, errors::NIL};
}
}
