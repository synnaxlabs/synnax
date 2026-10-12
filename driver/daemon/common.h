// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "x/cpp/errors/errors.h"

namespace driver::daemon {
/// @brief creates the synnax system user the service runs as, if it does not exist.
x::errors::Error create_system_user();

/// @brief copies the running binary to /usr/local/bin/synnax-driver. Skips the copy
/// when the running binary is already the installed one.
x::errors::Error install_binary();

/// @brief creates the Driver's environment file, owned by root with mode 0600. The
/// service manager reads it as root, so only root can read the password in it. An
/// existing file keeps its contents but gets that owner and mode. Requires root.
x::errors::Error create_env_file();
}
