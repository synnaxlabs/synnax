// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "x/cpp/xerrors/errors.h"

namespace fs {
const auto FS_ERROR = xerrors::Error("fs", "");
const xerrors::Error NOT_FOUND = FS_ERROR.sub("not_found");
const xerrors::Error INVALID_PATH = FS_ERROR.sub("invalid_path");
const xerrors::Error PERMISSION_DENIED = FS_ERROR.sub("permission_denied");
const xerrors::Error READ_ERROR = FS_ERROR.sub("read_error");

/// @brief an internal method for reading the entire contents of certificate files
/// into a string.
inline std::pair<std::string, xerrors::Error> read_file(const std::string &path) {
    std::string data;
    FILE *f = fopen(path.c_str(), "r");
    if (f == nullptr)
        return {data, xerrors::Error(NOT_FOUND, "failed to open " + path)};
    char buf[1024];
    for (;;) {
        const size_t n = fread(buf, 1, sizeof(buf), f);
        if (n <= 0) break;
        data.append(buf, n);
    }
    if (ferror(f)) return {"", xerrors::Error(READ_ERROR, "failed to read " + path)};
    fclose(f);
    return {data, xerrors::NIL};
}
}
