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

/// @brief secures the state file at path and its directory, and skips either when it
/// does not exist. Only a dedicated directory, which the Driver chose, changes mode,
/// owner, or access. On Windows, it creates the directory of the computer instead, and
/// returns an error when neither LocalSystem nor the administrators own it.
x::errors::Error secure(const std::filesystem::path &path, bool dedicated);
}
