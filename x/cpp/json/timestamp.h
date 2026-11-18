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
#include <utility>

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/convert.h"
#include "x/cpp/telem/telem.h"

namespace x::json {
/// @brief parses an RFC 3339 timestamp in text, which it may change. It also takes a
/// lowercase "t" or a space between the date and time, and a lowercase "z".
/// @returns VALIDATION when text is not RFC 3339, or is past the range of a timestamp.
std::pair<telem::TimeStamp, errors::Error> parse_iso8601(std::string &text);

/// @brief parses text, a decimal number, as a count of the units of format, which is
/// not ISO8601. It keeps every digit down to the nanosecond.
/// @returns VALIDATION when text is not a number, or is past the range of a timestamp.
std::pair<telem::TimeStamp, errors::Error>
parse_unix(std::string_view text, TimeFormat format);
}
