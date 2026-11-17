// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cerrno>
#include <cstdio>
#include <string>
#include <system_error>

#include <fcntl.h>
#include <sys/file.h>
#include <unistd.h>

#include "driver/journal/file.h"

namespace driver::journal {
namespace {
x::errors::Error errno_error(const std::string &msg) {
    return x::errors::Error(
        BASE_ERROR,
        msg + ": " + std::generic_category().message(errno)
    );
}

/// @brief fsync on macOS does not flush the drive cache, so it needs F_FULLFSYNC. Some
/// file systems do not support F_FULLFSYNC, and only get fsync.
int full_sync(const int fd) {
#ifdef __APPLE__
    if (fcntl(fd, F_FULLFSYNC) == 0) return 0;
    if (errno != ENOTSUP && errno != EINVAL && errno != ENOTTY) return -1;
#endif
    return fsync(fd);
}
}

File::File(const std::intptr_t handle): handle(handle) {}

File::~File() {
    close(static_cast<int>(this->handle));
}

std::pair<std::unique_ptr<File>, x::errors::Error>
File::open(const std::filesystem::path &path) {
    const int fd = ::open(
        path.c_str(),
        O_WRONLY | O_CREAT | O_APPEND | O_CLOEXEC,
        0644
    );
    if (fd < 0) return {nullptr, errno_error("failed to open " + path.string())};
    return {std::unique_ptr<File>(new File(fd)), x::errors::NIL};
}

x::errors::Error File::write(std::string_view data) const {
    while (!data.empty()) {
        const auto n = ::write(
            static_cast<int>(this->handle),
            data.data(),
            data.size()
        );
        if (n < 0 && errno == EINTR) continue;
        if (n < 0) return errno_error("failed to write");
        data.remove_prefix(static_cast<std::size_t>(n));
    }
    return x::errors::NIL;
}

x::errors::Error File::truncate(const std::uint64_t size) const {
    if (ftruncate(static_cast<int>(this->handle), static_cast<off_t>(size)) != 0)
        return errno_error("failed to truncate");
    return x::errors::NIL;
}

x::errors::Error File::sync() const {
    if (full_sync(static_cast<int>(this->handle)) != 0)
        return errno_error("failed to sync");
    return x::errors::NIL;
}

x::errors::Error File::lock() const {
    if (flock(static_cast<int>(this->handle), LOCK_EX | LOCK_NB) == 0)
        return x::errors::NIL;
    if (errno == EWOULDBLOCK) return LOCKED;
    return errno_error("failed to lock");
}

x::errors::Error sync_dir(const std::filesystem::path &dir) {
    const int fd = ::open(dir.c_str(), O_RDONLY | O_DIRECTORY | O_CLOEXEC);
    if (fd < 0) return errno_error("failed to open " + dir.string());
    auto err = full_sync(fd) == 0 ? x::errors::NIL
                                  : errno_error("failed to sync " + dir.string());
    close(fd);
    return err;
}

x::errors::Error
replace(const std::filesystem::path &from, const std::filesystem::path &to) {
    if (std::rename(from.c_str(), to.c_str()) != 0)
        return errno_error("failed to rename " + from.string());
    return x::errors::NIL;
}
}
