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
#include <string_view>

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

namespace driver::codec {
/// @brief reports whether a field of kind k converts to samples of dt. A number or
/// boolean also converts to a float, and an unsigned number or boolean to any integer.
bool converts(std::string_view k, const x::telem::DataType &dt);

/// @brief returns the error err with context before its text.
x::errors::Error wrap(const x::errors::Error &err, const std::string &context);

/// @brief returns err as a CONFIG_ERROR with context before its text.
x::errors::Error as_config(const x::errors::Error &err, const std::string &context);
}
