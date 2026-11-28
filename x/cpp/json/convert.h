// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <map>
#include <string>
#include <utility>

#include <nlohmann/json.hpp>

#include "x/cpp/errors/errors.h"
#include "x/cpp/payload/types.gen.h"
#include "x/cpp/telem/telem.h"

namespace x::json {

/// @brief base error type for JSON errors.
const errors::Error BASE_ERROR = errors::Error("x.json");
/// @brief error when conversion from JSON to a Synnax DataType fails.
const errors::Error CONVERSION_ERROR = BASE_ERROR.sub("conversion");

namespace details {
/// @brief TimeFormat is a payload time format in the form that a conversion reads for
/// each sample.
enum class TimeFormat {
    ISO8601,
    UnixSecond,
    UnixMillisecond,
    UnixMicrosecond,
    UnixNanosecond,
};

/// @brief parses format, a payload time format such as payload::TIME_FORMAT_UNIX_MS.
/// @returns VALIDATION when format is not a payload time format.
std::pair<TimeFormat, errors::Error> parse_time_format(const std::string &format);

/// @brief splits ts into a whole count of the units of format, a Unix format, and the
/// nanoseconds left over, both with the sign of ts. UnixMillisecond and UnixMicrosecond
/// round the count toward negative infinity and leave nothing over.
std::pair<std::int64_t, std::int64_t>
split_unix(telem::TimeStamp ts, TimeFormat format);
}

/// @brief maps enum labels to the samples they stand for.
using EnumMap = std::map<std::string, telem::SampleValue>;

/// @brief converts a JSON value to a SampleValue of the given target DataType.
/// Inspects the JSON value's type at runtime to determine the conversion path.
/// @param value the JSON value to convert.
/// @param target the Synnax DataType to convert to.
/// @param time_format the payload time format of a TIMESTAMP_T conversion. Ignored
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
/// CONVERSION_ERROR if an issue occurred while trying to convert the value, or
/// VALIDATION when time_format is not a payload time format.
std::pair<telem::SampleValue, errors::Error> to_sample_value(
    const nlohmann::json &value,
    const telem::DataType &target,
    const std::string &time_format = payload::TIME_FORMAT_ISO_8601,
    const EnumMap *enum_values = nullptr
);

/// @brief JSON value type.
enum class Type { Number, String, Boolean };

}
