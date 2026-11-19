// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <filesystem>
#include <memory>
#include <string_view>
#include <utility>

#include "x/cpp/errors/errors.h"

namespace driver::journal {
const x::errors::Error BASE_ERROR = x::errors::Error("journal", "");
/// @brief another holder has the lock of a file.
const x::errors::Error LOCKED = BASE_ERROR.sub("locked");

/// @brief a file opened for appending, which can sync to the disk.
class File {
    std::intptr_t handle;
    explicit File(std::intptr_t handle);

public:
    File(const File &) = delete;
    File &operator=(const File &) = delete;
    ~File();

    /// @brief opens or creates the file at path for appending.
    static std::pair<std::unique_ptr<File>, x::errors::Error>
    open(const std::filesystem::path &path);

    /// @brief writes all of data at the end of the file.
    [[nodiscard]] x::errors::Error write(std::string_view data) const;

    /// @brief cuts the file to size bytes.
    [[nodiscard]] x::errors::Error truncate(std::uint64_t size) const;

    /// @brief returns after the data and size of the file are on the disk.
    [[nodiscard]] x::errors::Error sync() const;

    /// @brief takes an exclusive lock on the file without blocking. The lock does
    /// not block reads of the file. Closing the file releases the lock.
    /// @returns LOCKED when another holder has a lock on the file.
    [[nodiscard]] x::errors::Error lock() const;
};

/// @brief returns after the entries of dir are on the disk. Windows has no directory
/// sync, so there it does nothing.
x::errors::Error sync_dir(const std::filesystem::path &dir);

/// @brief renames from to to, and replaces a file at to. On Windows, returns after
/// the rename is on the disk.
x::errors::Error
replace(const std::filesystem::path &from, const std::filesystem::path &to);

/// @brief writes a file at path with data, and replaces an existing file. A crash
/// leaves the old file or the new one, and can leave a temp file named
/// `<path>.tmp.<n>`. Returns after the new file is on the disk.
x::errors::Error create(const std::filesystem::path &path, std::string_view data);
}
