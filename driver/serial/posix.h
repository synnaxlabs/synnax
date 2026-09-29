// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>

#include "x/cpp/errors/errors.h"

/// @brief port operations that Linux and macOS share.
namespace driver::serial::native {
/// @returns CONFIG_ERROR for a setting that the port rejected, carrying errno.
x::errors::Error failed(const std::string &setting, const std::string &port);
}
