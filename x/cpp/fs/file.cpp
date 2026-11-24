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
#include <system_error>
#include <vector>

#include "x/cpp/fs/file.h"

namespace x::fs {
namespace {
x::errors::Error write_synced(
    const std::filesystem::path &path,
    std::string_view data,
    const OnCreate &on_create
) {
    auto [file, err] = File::open(path);
    if (err) return err;
    if (err = on_create(path); err) return err;
    if (err = file->truncate(0); err) return err;
    if (err = file->write(data); err) return err;
    return file->sync();
}

const std::string TEMP_INFIX = ".tmp.";
}

x::errors::Error create(
    const std::filesystem::path &path,
    const std::string_view data,
    const OnCreate &on_create
) {
    auto tmp = path;
    tmp += TEMP_INFIX + std::to_string(std::random_device()());
    auto err = write_synced(tmp, data, on_create);
    if (!err) err = replace(tmp, path);
    if (err) {
        std::error_code ec;
        std::filesystem::remove(tmp, ec);
        return err;
    }
    const auto dir = path.parent_path();
    return sync_dir(dir.empty() ? "." : dir);
}

x::errors::Error create_new(
    const std::filesystem::path &path,
    const std::string_view data,
    const OnCreate &on_create
) {
    auto tmp = path;
    tmp += TEMP_INFIX + std::to_string(std::random_device()());
    auto err = write_synced(tmp, data, on_create);
    if (!err) err = rename_new(tmp, path);
    if (err) {
        std::error_code ec;
        // A concurrent create of path removes the temp files of the others.
        if (!err.matches(EXISTS) && !std::filesystem::exists(tmp, ec) &&
            std::filesystem::exists(path, ec))
            return x::errors::Error(EXISTS, path.string());
        std::filesystem::remove(tmp, ec);
        return err;
    }
    const auto dir = path.parent_path();
    return sync_dir(dir.empty() ? "." : dir);
}

x::errors::Error remove_temps(const std::filesystem::path &path) {
    const auto dir = path.parent_path().empty() ? std::filesystem::path(".")
                                                : path.parent_path();
    const auto prefix = path.filename().string() + TEMP_INFIX;
    std::error_code ec;
    std::vector<std::filesystem::path> temps;
    std::filesystem::directory_iterator it(dir, ec);
    for (; !ec && it != std::filesystem::directory_iterator(); it.increment(ec))
        if (it->path().filename().string().starts_with(prefix))
            temps.push_back(it->path());
    if (ec)
        return x::errors::Error(
            BASE_ERROR,
            "failed to list " + dir.string() + ": " + ec.message()
        );
    for (const auto &temp: temps)
        std::filesystem::remove(temp, ec);
    return x::errors::NIL;
}
}
