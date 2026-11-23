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

namespace x::json {
/// @brief returns text in double quotes for an error message, with quotes, backslashes,
/// control characters, and bytes that are not valid UTF-8 escaped.
std::string quote(std::string_view text);

/// @brief appends JSON text to out: text as a JSON string. Only quotes, backslashes,
/// and control characters are escaped, with \b, \t, \n, \f, and \r in short form.
/// @returns VALIDATION, appending nothing, when text is not valid UTF-8.
[[nodiscard]] errors::Error append_quoted(std::string_view text, std::string &out);

/// @brief decodes raw, one JSON string literal with its quotes, into dst. dst is
/// cleared first.
/// @returns VALIDATION when raw is not a valid JSON string literal.
errors::Error unquote(std::string_view raw, std::string &dst);
}
