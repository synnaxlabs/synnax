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

namespace x::base64 {

/// @brief base64-encodes the input string.
std::string encode(const std::string &input);

/// @brief base64-decodes the input string.
std::string decode(const std::string &input);

}
