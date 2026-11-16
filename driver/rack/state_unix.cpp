// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cerrno>
#include <cstdlib>
#include <cstring>
#include <filesystem>
#include <string>

#include <fcntl.h>
#include <pwd.h>
#include <sys/stat.h>
#include <unistd.h>

#include "driver/rack/rack.h"
#include "driver/rack/state.h"

namespace fs = std::filesystem;

namespace driver::rack::state {
namespace {
/// @brief a descriptor that closes itself.
struct FD {
    int fd = -1;
    ~FD() {
        if (this->fd >= 0) close(this->fd);
    }
};

x::errors::Error errno_error(const std::string &action) {
    return x::errors::Error(STATE_ERROR, action + ": " + std::strerror(errno));
}

/// @brief writes all of data to fd, retrying short writes.
bool write_all(const int fd, const char *data, size_t size) {
    while (size > 0) {
        const auto n = write(fd, data, size);
        if (n < 0) return false;
        data += n;
        size -= static_cast<size_t>(n);
    }
    return true;
}

/// @brief replaces a state file that another user owns with a copy this process owns,
/// so its mode can be made owner-only.
x::errors::Error take_state_file(const int dir_fd, const std::string &name) {
    FD src{openat(dir_fd, name.c_str(), O_RDONLY | O_NOFOLLOW | O_CLOEXEC)};
    if (src.fd < 0) {
        if (errno == ENOENT) return x::errors::NIL;
        return errno_error("failed to open state file");
    }
    struct stat st{};
    if (fstat(src.fd, &st) != 0) return errno_error("failed to read state file");
    if (st.st_uid == geteuid()) return x::errors::NIL;
    const auto tmp = name + ".tmp";
    unlinkat(dir_fd, tmp.c_str(), 0);
    FD dst{openat(
        dir_fd,
        tmp.c_str(),
        O_WRONLY | O_CREAT | O_EXCL | O_NOFOLLOW | O_CLOEXEC,
        S_IRUSR | S_IWUSR
    )};
    if (dst.fd < 0) return errno_error("failed to take over state file");
    char buf[4096];
    ssize_t n = 0;
    while ((n = read(src.fd, buf, sizeof(buf))) > 0)
        if (!write_all(dst.fd, buf, static_cast<size_t>(n))) break;
    if (n != 0 || fsync(dst.fd) != 0 ||
        renameat(dir_fd, tmp.c_str(), dir_fd, name.c_str()) != 0) {
        auto err = errno_error("failed to take over state file");
        unlinkat(dir_fd, tmp.c_str(), 0);
        return err;
    }
    return x::errors::NIL;
}

/// @brief creates the OPC UA directory in the state directory if it is missing and
/// gives it to the service user. Root never writes in it.
x::errors::Error give_opcua_dir(const int dir_fd, const passwd &user) {
    if (mkdirat(dir_fd, OPCUA_STATE_DIR.c_str(), S_IRWXU) != 0 && errno != EEXIST)
        return errno_error("failed to create OPC UA directory");
    FD sub{openat(
        dir_fd,
        OPCUA_STATE_DIR.c_str(),
        O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC
    )};
    if (sub.fd < 0) return errno_error("failed to open OPC UA directory");
    if (fchown(sub.fd, user.pw_uid, user.pw_gid) != 0 || fchmod(sub.fd, S_IRWXU) != 0)
        return errno_error("failed to give OPC UA directory to the service user");
    return x::errors::NIL;
}
}

fs::path machine_dir() {
#ifdef __APPLE__
    return "/Library/Application Support/synnax-driver";
#else
    return "/var/lib/synnax-driver";
#endif
}

fs::path user_dir() {
#ifdef __APPLE__
    const char *home = std::getenv("HOME");
    if (const char *sudo_user = std::getenv("SUDO_USER"))
        if (const passwd *user = getpwnam(sudo_user)) home = user->pw_dir;
    if (home != nullptr)
        return fs::path(home) / "Library" / "Application Support" / "synnax-driver";
#endif
    return {};
}

/// @brief opens the state directory and file without following links, so a link put
/// in their place is refused. A file that does not exist yet is skipped. Only the
/// default directory is dedicated to the Driver, so only it changes mode or owner,
/// through the open descriptors. When root writes it, as `sudo synnax-driver login`
/// does, root keeps the directory and the Linux service user gets read access to it
/// and owns the file and the OPC UA directory. The service can then read and write its
/// key but cannot put a link in the directory for root to follow. A process that is not
/// root and does not own the file takes it over first.
x::errors::Error secure(const fs::path &path, const bool dedicated) {
    FD dir{open(
        path.parent_path().c_str(),
        O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC
    )};
    if (dir.fd < 0) {
        if (errno == ENOENT) return x::errors::NIL;
        return errno_error("failed to open state directory");
    }
    const passwd *service_user = nullptr;
#ifdef __linux__
    if (geteuid() == 0) service_user = getpwnam("synnax");
#endif
    if (dedicated) {
        struct stat st{};
        if (fstat(dir.fd, &st) != 0)
            return errno_error("failed to read state directory");
        if (service_user != nullptr) {
            // Other users may pass through to run the service binary in bin/.
            if (fchown(dir.fd, 0, service_user->pw_gid) != 0 ||
                fchmod(dir.fd, S_IRWXU | S_IRGRP | S_IXGRP | S_IXOTH) != 0)
                return errno_error("failed to secure state directory");
            if (auto err = give_opcua_dir(dir.fd, *service_user)) return err;
        } else if (st.st_uid == geteuid() && fchmod(dir.fd, S_IRWXU) != 0)
            return errno_error("failed to secure state directory");
        if (service_user == nullptr)
            if (auto err = take_state_file(dir.fd, path.filename().string()))
                return err;
    }
    FD file{openat(dir.fd, path.filename().c_str(), O_RDONLY | O_NOFOLLOW | O_CLOEXEC)};
    if (file.fd < 0) {
        if (errno == ENOENT) return x::errors::NIL;
        return errno_error("failed to open state file");
    }
    if (dedicated && service_user != nullptr &&
        (fchown(file.fd, service_user->pw_uid, service_user->pw_gid) != 0 ||
         fchmod(file.fd, S_IRUSR | S_IWUSR) != 0))
        return errno_error("failed to give state file to the service user");
    return x::errors::NIL;
}
}
