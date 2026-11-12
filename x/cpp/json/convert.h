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
#include <vector>

#include <nlohmann/json.hpp>

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/series.h"
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

/// @brief maps enum labels to the samples they stand for (read direction: JSON string
/// → sample).
using EnumMap = std::map<std::string, telem::SampleValue>;

/// @brief pairs samples with enum labels (write direction: sample → JSON string). A
/// sample takes the label of the first entry it matches.
using ReverseEnumMap = std::vector<std::pair<telem::SampleValue, std::string>>;

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
/// @param enum_values optional reverse enum mapping. When non-null, the value takes the
/// label of the first entry it matches, whatever the target. Integers, strings, and
/// timestamps match by exact value. Floats match by value in the type of the sample,
/// and a NaN entry matches any NaN. Otherwise normal conversion applies, and a String
/// target writes NaN and the infinities as "NaN", "Infinity", and "-Infinity".
/// @returns the JSON value and nil, or an empty JSON value and CONVERSION_ERROR if
/// unsupported. NaN and infinity have no JSON number, and NaN has no JSON boolean.
std::pair<nlohmann::json, errors::Error> from_sample_value(
    const telem::SampleValue &value,
    Type target,
    const ReverseEnumMap *enum_values = nullptr
);

/// @brief checks at config time whether a DataType can be converted to the given JSON
/// Type.
/// @param type the Synnax DataType to check.
/// @param target the JSON type to convert to.
/// @returns nil if supported, CONVERSION_ERROR otherwise.
errors::Error check_from_sample_value(const telem::DataType &type, Type target);

/// @brief converts the sample at index of a JSON, UUID, or bytes series to JSON. A JSON
/// sample becomes the value that its text holds, a UUID its text, and bytes padded
/// base64 text.
/// @param series the series that holds the sample.
/// @param index the index of the sample. A negative index counts from the end.
/// @returns the JSON value and nil, or CONVERSION_ERROR if the series has another
/// data type or a JSON sample is not valid JSON.
/// @throws std::runtime_error if the index is out of bounds.
std::pair<nlohmann::json, errors::Error>
from_series(const telem::Series &series, int index);

/// @brief converts a TimeStamp to a JSON value using the given TimeFormat.
/// @param ts the timestamp to convert.
/// @param format the output format.
/// @returns the JSON representation of the timestamp. A Unix format is an exact integer
/// when ts is a whole count of units, and otherwise the double nearest its exact
/// decimal value.
nlohmann::json from_timestamp(telem::TimeStamp ts, TimeFormat format);

/// @brief returns the zero value for a JSON Type.
/// @param format the JSON type (Number → 0, String → "", Boolean → false).
/// @returns the zero JSON value.
nlohmann::json zero_value(Type format);

}
