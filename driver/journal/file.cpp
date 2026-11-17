// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <random>
#include <string>

#include "driver/journal/file.h"

namespace driver::journal {
x::errors::Error
create(const std::filesystem::path &path, const std::string_view data) {
    auto tmp = path;
    tmp += ".tmp." + std::to_string(std::random_device()());
    {
        auto [file, err] = File::open(tmp);
        if (err) return err;
        if (err = file->truncate(0); err) return err;
        if (err = file->write(data); err) return err;
        if (err = file->sync(); err) return err;
    }
    if (auto err = replace(tmp, path)) return err;
    const auto dir = path.parent_path();
    return sync_dir(dir.empty() ? "." : dir);
}
}
