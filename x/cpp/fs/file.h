// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <cstdint>
#include <filesystem>
#include <memory>
#include <string_view>
#include <utility>

#include "x/cpp/errors/errors.h"
#include "x/cpp/fs/fs.h"

namespace x::fs {
/// @brief another holder has the lock of a file.
const x::errors::Error LOCKED = FS_ERROR.sub("locked");

/// @brief the state of an open file.
struct Info {
    /// @brief the byte size of the file.
    std::uint64_t size = 0;
    /// @brief the file has no name, because a removal or a replace took it.
    bool removed = false;
};

/// @brief a file opened for appending or for reading.
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

    /// @brief opens the file at path for reading. Other handles can rename, replace,
    /// or remove the file while it is open.
    static std::pair<std::unique_ptr<File>, x::errors::Error>
    open_read(const std::filesystem::path &path);

    /// @brief reads up to size bytes at pos into dst.
    /// @returns the number of bytes read, which is less than size only at the end of
    /// the file.
    [[nodiscard]] std::pair<std::size_t, x::errors::Error>
    read(std::uint64_t pos, char *dst, std::size_t size) const;

    /// @returns the size of the file, and whether a removal or a replace took its
    /// name.
    [[nodiscard]] std::pair<Info, x::errors::Error> stat() const;

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

/// @brief renames from to to, and replaces a file at to, also one that other handles
/// hold open. On Windows, returns after the rename is on the disk, and on a volume with
/// no POSIX rename, such as FAT, fails while another handle holds to open.
x::errors::Error
replace(const std::filesystem::path &from, const std::filesystem::path &to);

/// @brief writes a file at path with data, and replaces an existing file. A crash
/// leaves the old file or the new one, and can leave a temp file named
/// `<path>.tmp.<n>`. Returns after the new file is on the disk.
x::errors::Error create(const std::filesystem::path &path, std::string_view data);

/// @brief removes the temp files that a crash during create(path) left. Call it only
/// while no create(path) runs.
x::errors::Error remove_temps(const std::filesystem::path &path);
}
