// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <filesystem>
#include <string>

#include <sys/types.h>

#include "x/cpp/errors/errors.h"

namespace driver::daemon {
/// @brief the environment file of the Driver service.
const std::string ENV_FILE = "/etc/synnax/driver.env";

/// @brief creates the synnax system user the service runs as, if it does not exist.
x::errors::Error create_system_user();

/// @brief copies the binary at src to dst with mode 0755, owned by uid and gid. It
/// never follows a link in the folder of dst, and src may be dst.
x::errors::Error install_binary(
    const std::filesystem::path &src,
    const std::filesystem::path &dst,
    uid_t uid,
    gid_t gid
);

/// @brief creates the Driver's environment file, owned by root with mode 0600, so only
/// root can read the password in it. An existing file keeps its contents but gets that
/// owner and mode. Requires root.
x::errors::Error create_env_file();
}
