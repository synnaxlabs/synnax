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

#include "driver/daemon/daemon.h"

namespace fs = std::filesystem;

namespace driver::daemon {
const std::string ENV_FILE = "/etc/synnax/driver.env";

x::errors::Error load_env() {
    return x::env::load_file(ENV_FILE);
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
