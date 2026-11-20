// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cerrno>
#include <cstring>
#include <filesystem>
#include <fstream>
#include <string>

#include "absl/log/log.h"
#include <fcntl.h>
#include <sys/stat.h>
#include <unistd.h>

#include "driver/daemon/common.h"
#include "driver/daemon/daemon.h"

namespace fs = std::filesystem;

namespace driver::daemon {
x::errors::Error create_system_user() {
    LOG(INFO) << "creating system user";
    // NOLINTNEXTLINE(bugprone-command-processor): a fixed string.
    if (system("id -u synnax >/dev/null 2>&1 || useradd -r -s /sbin/nologin synnax") !=
        0)
        return x::errors::Error("failed to create system user");
    return x::errors::NIL;
}

namespace {
/// @brief a descriptor that closes itself.
struct FD {
    int fd = -1;
    ~FD() {
        if (this->fd >= 0) close(this->fd);
    }
};

x::errors::Error errno_error(const std::string &action) {
    return x::errors::Error(action + ": " + std::strerror(errno));
}

/// @brief copies all of in to out, retrying short writes.
bool copy_all(const int in, const int out) {
    char buf[65536];
    ssize_t n = 0;
    while ((n = read(in, buf, sizeof(buf))) > 0)
        for (ssize_t off = 0; off < n;) {
            const auto w = write(out, buf + off, static_cast<size_t>(n - off));
            if (w < 0) return false;
            off += w;
        }
    return n == 0;
}
}

x::errors::Error install_binary(
    const fs::path &src,
    const fs::path &dst,
    const uid_t uid,
    const gid_t gid
) {
    LOG(INFO) << "copying binary to " << dst;
    std::error_code ec;
    fs::create_directories(dst.parent_path(), ec);
    if (ec)
        return x::errors::Error("failed to create binary directory: " + ec.message());
    FD dir{
        open(dst.parent_path().c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC)
    };
    if (dir.fd < 0) return errno_error("failed to open binary directory");
    FD in{open(src.c_str(), O_RDONLY | O_CLOEXEC)};
    if (in.fd < 0) return errno_error("failed to open binary");
    // The service user may own the directory, so root never follows a link in it.
    // Renaming a complete copy over the old binary keeps it in place if the copy fails,
    // and avoids ETXTBSY when a process still runs the old binary.
    const auto name = dst.filename().string();
    const auto tmp = name + "." + std::to_string(getpid()) + ".tmp";
    unlinkat(dir.fd, tmp.c_str(), 0);
    FD out{openat(
        dir.fd,
        tmp.c_str(),
        O_WRONLY | O_CREAT | O_EXCL | O_NOFOLLOW | O_CLOEXEC,
        S_IRUSR | S_IWUSR
    )};
    if (out.fd < 0) return errno_error("failed to create binary copy");
    constexpr auto mode = S_IRWXU | S_IRGRP | S_IXGRP | S_IROTH | S_IXOTH;
    if (!copy_all(in.fd, out.fd) || fchown(out.fd, uid, gid) != 0 ||
        fchmod(out.fd, mode) != 0 || fsync(out.fd) != 0 ||
        renameat(dir.fd, tmp.c_str(), dir.fd, name.c_str()) != 0) {
        auto err = errno_error("failed to install binary");
        unlinkat(dir.fd, tmp.c_str(), 0);
        return err;
    }
    return x::errors::NIL;
}

x::errors::Error create_env_file() {
    LOG(INFO) << "creating environment file at " << ENV_FILE;
    std::error_code ec;
    fs::create_directories(fs::path(ENV_FILE).parent_path(), ec);
    if (ec)
        return x::errors::Error(
            "failed to create environment file directory: " + ec.message()
        );
    // Opened in append mode so an existing file keeps its contents.
    std::ofstream env_file(ENV_FILE, std::ios::app);
    if (!env_file) return x::errors::Error("failed to create environment file");
    env_file.close();
    if (chown(ENV_FILE.c_str(), 0, 0) != 0)
        return x::errors::Error("failed to set environment file ownership");
    if (chmod(ENV_FILE.c_str(), S_IRUSR | S_IWUSR) != 0)
        return x::errors::Error("failed to set environment file permissions");
    return x::errors::NIL;
}
}
