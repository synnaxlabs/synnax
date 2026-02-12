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

/// @brief base error for JSON conversion errors.
const errors::Error BASE_ERROR = errors::Error("xjson.conversion", "");
/// @brief error for unsupported conversions.
const errors::Error UNSUPPORTED_ERROR = BASE_ERROR.sub("unsupported");
/// @brief error for unexpected truncation.
const errors::Error TRUNCATION_ERROR = BASE_ERROR.sub("truncation");
/// @brief error for unexpected overflow.
const errors::Error OVERFLOW_ERROR = BASE_ERROR.sub("overflow");
/// @brief error for invalid ISO 8601 timestamp strings.
const errors::Error INVALID_ISO_ERROR = BASE_ERROR.sub("invalid_iso");

/// @brief JSON value type.
enum class Type { Number, String, Boolean };

/// @brief time format for converting between `telem::TimeStamp`s and JSON values.
enum class TimeFormat {
    ISO8601,
    UnixSecond,
    UnixMillisecond,
    UnixMicrosecond,
    UnixNanosecond,
};

/// @brief options for to_sample_value.
struct ReadOptions {
    /// @brief if true, numeric conversions that lose precision (e.g. float → int
    /// truncation, overflow) return an error instead of silently truncating.
    bool strict = false;
    /// @brief the expected time format for JSON → TimeStamp conversions. Ignored when
    /// the target type is not TIMESTAMP_T.
    TimeFormat time_format = TimeFormat::ISO8601;
};

/// @brief converts a JSON value to a SampleValue of the given target DataType.
/// Inspects the JSON value's type at runtime to determine the conversion path.
/// @param value the JSON value to convert.
/// @param target the Synnax DataType to convert to.
/// @param opts conversion options (strictness, time format).
/// @returns the converted SampleValue and nil, or a zero SampleValue and an error.
std::pair<telem::SampleValue, errors::Error> to_sample_value(
    const nlohmann::json &value,
    const telem::DataType &target,
    const ReadOptions &opts = {}
);

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
/// @returns nil if supported, UNSUPPORTED_ERROR otherwise.
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
