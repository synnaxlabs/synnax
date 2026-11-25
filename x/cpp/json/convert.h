// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <map>
#include <string>
#include <utility>

#include <nlohmann/json.hpp>

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

namespace x::json {

/// @brief base error type for JSON errors.
const errors::Error BASE_ERROR = errors::Error("x.json");
/// @brief error when conversion from JSON to a Synnax DataType fails.
const errors::Error CONVERSION_ERROR = BASE_ERROR.sub("conversion");

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

/// @brief maps enum labels to the samples they stand for.
using EnumMap = std::map<std::string, telem::SampleValue>;

/// @brief converts a JSON value to a SampleValue of the given target DataType.
/// Inspects the JSON value's type at runtime to determine the conversion path.
/// @param value the JSON value to convert.
/// @param target the Synnax DataType to convert to.
/// @param time_format the expected time format for TIMESTAMP_T conversions. Ignored
/// when the target type is not TIMESTAMP_T.
/// @param enum_values optional mapping of labels to samples. When non-null and the
/// JSON value is a string targeting a numeric or string type, the map is checked
/// first. A found label converts its sample to target: exactly to an integer, and to
/// the nearest value of a float. Past the range of target, it fails. Otherwise normal
/// conversion applies.
/// A JSON_T sample is the JSON text of the value. A UUID_T sample is the 16 bytes of
/// the UUID that a string value holds, and a BYTES_T sample is the bytes that a
/// string of padded, canonical base64 holds. Build a series that keeps the target
/// type with telem::Series(SampleValue, DataType).
/// @returns the converted SampleValue and errors::NIL, or a zero SampleValue and one of
/// CONVERSION_ERROR if an issue occurred while trying to convert the value.
std::pair<telem::SampleValue, errors::Error> to_sample_value(
    const nlohmann::json &value,
    const telem::DataType &target,
    TimeFormat time_format = TimeFormat::ISO8601,
    const EnumMap *enum_values = nullptr
);

/// @brief JSON value type.
enum class Type { Number, String, Boolean };

}
