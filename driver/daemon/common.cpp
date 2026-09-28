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

#include "absl/log/log.h"
#include <sys/stat.h>

#include "x/cpp/env/env.h"

#include "driver/daemon/common.h"
#include "driver/daemon/daemon.h"

namespace fs = std::filesystem;

namespace driver::daemon {
const std::string BINARY_PATH = "/usr/local/bin/synnax-driver";
const std::string ENV_FILE = "/etc/synnax/driver.env";

x::errors::Error load_env() {
    return x::env::load_file(ENV_FILE);
}

x::errors::Error create_system_user() {
    LOG(INFO) << "creating system user";
    if (system("id -u synnax >/dev/null 2>&1 || useradd -r -s /sbin/nologin synnax") !=
        0)
        return x::errors::Error("failed to create system user");
    return x::errors::NIL;
}

x::errors::Error install_binary() {
    LOG(INFO) << "moving binary to " << BINARY_PATH;
    std::error_code ec;
    const fs::path curr_bin_path = fs::read_symlink("/proc/self/exe", ec);
    if (ec)
        return x::errors::Error(
            "failed to get current executable path: " + ec.message()
        );
    const fs::path target_path = BINARY_PATH;
    fs::create_directories(target_path.parent_path(), ec);
    if (ec)
        return x::errors::Error("failed to create binary directory: " + ec.message());
    // An install run from the installed binary would copy the file onto itself, which
    // copy_file rejects. Skip the copy: the binary is already in place.
    const bool already_installed = fs::exists(target_path, ec) &&
                                   fs::equivalent(curr_bin_path, target_path, ec);
    if (!already_installed) {
        // Unlinking first avoids ETXTBSY when a process still runs the old binary.
        fs::remove(target_path, ec);
        if (ec)
            return x::errors::Error(
                "failed to remove existing binary: " + ec.message()
            );
        fs::copy_file(curr_bin_path, target_path, ec);
        if (ec) return x::errors::Error("failed to copy binary: " + ec.message());
    }
    if (chmod(target_path.c_str(), S_IRWXU | S_IRGRP | S_IXGRP | S_IROTH | S_IXOTH) !=
        0)
        return x::errors::Error("failed to set binary permissions");
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
    // The file can hold the Core password: root writes it, the service reads it.
    if (chmod(ENV_FILE.c_str(), S_IRUSR | S_IWUSR | S_IRGRP) != 0)
        return x::errors::Error("failed to set environment file permissions");
    if (system(("chown root:synnax " + ENV_FILE).c_str()) != 0)
        return x::errors::Error("failed to set environment file ownership");
    return x::errors::NIL;
}
}
