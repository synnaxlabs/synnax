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

#include "x/cpp/errors/errors.h"

namespace driver::rack::state {
/// @brief the state directory of the computer, which the Driver service reads.
std::filesystem::path machine_dir();

/// @brief the state directory of the user, or of the user that ran sudo. Empty on
/// Linux, where every process uses the directory of the computer.
std::filesystem::path user_dir();

/// @brief returns whether the process runs as root under sudo for a user other than
/// root. Always false on Windows.
bool sudo();

/// @brief prepares the state file at path for a Driver run, and never changes the mode
/// or owner of a path that exists. Returns an error when the state directory or file is
/// a link, when other users can read the state file, or when another user owns it. Root
/// also accepts a state file that the Linux service user owns. On Windows, it creates a
/// missing state directory of the computer for LocalSystem and the administrators
/// alone.
x::errors::Error prepare(const std::filesystem::path &path);

/// @brief secures the state file at path and its directory, and skips either when it
/// does not exist. Only a dedicated directory, which the Driver chose, changes. Run as
/// root on Linux, it gives the state to the service user. On Windows, it secures the
/// dedicated state directory of the computer, and returns an error when neither
/// LocalSystem nor the administrators own it.
x::errors::Error secure(const std::filesystem::path &path, bool dedicated);
}
