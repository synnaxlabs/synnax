// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <filesystem>
#include <fstream>
#include <string>

#include "absl/log/log.h"
#include <sys/stat.h>
#include <unistd.h>

#include "x/cpp/env/env.h"

#include "driver/daemon/common.h"
#include "driver/daemon/daemon.h"

namespace fs = std::filesystem;

namespace driver::daemon {
const std::string ENV_FILE = "/etc/synnax/driver.env";

x::errors::Error load_env() {
    return x::env::load_file(ENV_FILE);
}

x::errors::Error create_system_user() {
    LOG(INFO) << "creating system user";
    // NOLINTNEXTLINE(bugprone-command-processor): a fixed string.
    if (system("id -u synnax >/dev/null 2>&1 || useradd -r -s /sbin/nologin synnax") !=
        0)
        return x::errors::Error("failed to create system user");
    return x::errors::NIL;
}

x::errors::Error
install_binary(const fs::path &path, const uid_t uid, const gid_t gid) {
    LOG(INFO) << "moving binary to " << path;
    std::error_code ec;
    const fs::path curr_bin_path = fs::read_symlink("/proc/self/exe", ec);
    if (ec)
        return x::errors::Error(
            "failed to get current executable path: " + ec.message()
        );
    fs::create_directories(path.parent_path(), ec);
    if (ec)
        return x::errors::Error("failed to create binary directory: " + ec.message());
    constexpr auto mode = S_IRWXU | S_IRGRP | S_IXGRP | S_IROTH | S_IXOTH;
    // An install run from the installed binary would copy the file onto itself, which
    // copy_file rejects. Skip the copy: the binary is already in place.
    if (fs::exists(path, ec) && fs::equivalent(curr_bin_path, path, ec)) {
        if (chown(path.c_str(), uid, gid) != 0 || chmod(path.c_str(), mode) != 0)
            return x::errors::Error("failed to set binary permissions");
        return x::errors::NIL;
    }
    // Renaming a complete copy over the old binary keeps it in place if the copy fails,
    // and avoids ETXTBSY when a process still runs the old binary.
    fs::path tmp_path = path;
    tmp_path += "." + std::to_string(getpid()) + ".tmp";
    fs::copy_file(curr_bin_path, tmp_path, fs::copy_options::overwrite_existing, ec);
    if (ec) {
        const auto msg = ec.message();
        fs::remove(tmp_path, ec);
        return x::errors::Error("failed to copy binary: " + msg);
    }
    if (chown(tmp_path.c_str(), uid, gid) != 0 || chmod(tmp_path.c_str(), mode) != 0) {
        fs::remove(tmp_path, ec);
        return x::errors::Error("failed to set binary permissions");
    }
    fs::rename(tmp_path, path, ec);
    if (ec) {
        const auto msg = ec.message();
        fs::remove(tmp_path, ec);
        return x::errors::Error("failed to replace binary: " + msg);
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
