// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <nlohmann/json.hpp>

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

namespace x::json {

/// @brief base error type for JSON errors.
const errors::Error ERROR = errors::Error("x.json");
/// @brief error when conversion from JSON to a Synnax DataType fails.
const errors::Error CONVERSION_ERROR = ERROR.sub("conversion");

/// @brief time format for converting between `telem::TimeStamp`s and JSON values.
enum class TimeFormat {
    ISO8601,
    UnixSecond,
    UnixMillisecond,
    UnixMicrosecond,
    UnixNanosecond,
};

/// @brief parses a TimeFormat from a string.
/// @param str the string to parse ("iso8601", "unix_sec", "unix_ms", "unix_us",
/// "unix_ns").
/// @returns the TimeFormat and nil, or ISO8601 and INVALID_TIME_FORMAT_ERROR if the
/// string is unknown.
std::pair<TimeFormat, errors::Error> parse_time_format(const std::string &str);

/// @brief converts a JSON value to a SampleValue of the given target DataType.
/// Inspects the JSON value's type at runtime to determine the conversion path.
/// @param value the JSON value to convert.
/// @param target the Synnax DataType to convert to.
/// @param time_format the expected time format for TIMESTAMP_T conversions. Ignored
/// when the target type is not TIMESTAMP_T.
/// @returns the converted SampleValue and errors::NIL, or a zero SampleValue and one of
/// CONVERSION_ERROR if an issue occurred while trying to convert the value.
std::pair<telem::SampleValue, errors::Error> to_sample_value(
    const nlohmann::json &value,
    const telem::DataType &target,
    TimeFormat time_format = TimeFormat::ISO8601
);

/// @brief returns true if a JSON value can at least sometimes be converted to the given
/// DataType.
/// @param target the Synnax DataType to check.
/// @returns true if the DataType can be converted to the given JSON Type, false
/// otherwise.
bool check_to_sample_value(const telem::DataType &target);

/// @brief JSON value type.
enum class Type { Number, String, Boolean };

/// @brief converts a SampleValue to a JSON value of the given target type.
/// @param value the SampleValue to convert.
/// @param target the JSON type to convert to.
/// @returns the JSON value and nil, or an empty JSON value and an error if unsupported.
std::pair<nlohmann::json, errors::Error>
from_sample_value(const telem::SampleValue &value, Type target);

/// @brief checks at config time whether a DataType can be converted to the given JSON
/// Type.
/// @param type the Synnax DataType to check.
/// @param target the JSON type to convert to.
/// @returns nil if supported, CONVERSION_ERROR otherwise.
errors::Error check_from_sample_value(const telem::DataType &type, Type target);

/// @brief converts a TimeStamp to a JSON value using the given TimeFormat.
/// @param ts the timestamp to convert.
/// @param format the output format.
/// @returns the JSON representation of the timestamp.
nlohmann::json from_timestamp(telem::TimeStamp ts, TimeFormat format);

/// @brief returns the zero value for a JSON Type.
/// @param format the JSON type (Number → 0, String → "", Boolean → false).
/// @returns the zero JSON value.
nlohmann::json zero_value(Type format);

}
