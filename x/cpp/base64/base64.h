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
#include <utility>

#include "x/cpp/errors/errors.h"

namespace x::base64 {

/// @brief error for input that is not canonical base64.
const errors::Error INVALID = errors::SY.sub("base64.invalid");

/// @brief base64-encodes the input string.
std::string encode(const std::string &input);

/// @brief base64-decodes the input string. It skips characters outside the alphabet.
std::string decode(const std::string &input);

/// @brief base64-decodes the input string. The input must be padded, and only the
/// canonical text of its bytes is valid.
/// @param input the base64 text.
/// @returns the decoded bytes and nil, or INVALID if the input is not canonical
/// base64.
std::pair<std::string, errors::Error> decode_strict(const std::string &input);

}
