// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <functional>
#include <utility>

#include <nlohmann/json.hpp>

#include "x/cpp/telem/telem.h"
#include "x/cpp/xerrors/errors.h"

namespace xjson {

/// @brief base error for JSON conversion errors.
const xerrors::Error BASE_ERROR = xerrors::Error("xjson.conversion", "");
/// @brief error for unsupported conversions.
const xerrors::Error UNSUPPORTED_ERROR = BASE_ERROR.sub("unsupported");
/// @brief error for unexpected truncation.
const xerrors::Error TRUNCATION_ERROR = BASE_ERROR.sub("truncation");
/// @brief error for unexpected overflow.
const xerrors::Error OVERFLOW_ERROR = BASE_ERROR.sub("overflow");

/// @brief JSON value type.
enum class Type { Number, String, Boolean };

/// @brief time format for converting between `telem::TimeStamp`s and JSON values.
enum class TimeFormat {
    ISO8601,
    UnixSecondFloat,
    UnixSecondInt,
    UnixMillisecond,
    UnixMicrosecond,
    UnixNanosecond,
};

/// @brief resolved read converter. Takes a JSON value and returns a SampleValue
/// containing the converted value.
using ReadConverter = std::function<
    std::pair<telem::SampleValue, xerrors::Error>(const nlohmann::json &value)>;

/// @brief options for resolve_read_converter.
struct ReadOptions {
    /// @brief if true, numeric conversions that lose precision (e.g. float → int
    /// truncation, overflow) return an error instead of silently truncating.
    bool strict = false;
    /// @brief the expected time format for JSON → TimeStamp conversions. Ignored when
    /// the target type is not TIMESTAMP_T.
    TimeFormat time_format = TimeFormat::UnixNanosecond;
};

/// @brief resolves a read converter for a specific (`Type`, `DataType`) combination.
/// The returned function captures the exact C++ types, so there is no branching on
/// DataType at runtime.
/// @param json_type the JSON value type to convert from.
/// @param target_type the Synnax DataType to convert to.
/// @param opts conversion options (strictness, time format).
/// @returns the converter and nil, or nullptr and an error if unsupported.
std::pair<ReadConverter, xerrors::Error> resolve_read_converter(
    Type json_type,
    const telem::DataType &target_type,
    const ReadOptions &opts = {}
);

/// @brief converts a SampleValue to a JSON value of the given target type.
/// @param value the SampleValue to convert.
/// @param target the JSON type to convert to.
/// @returns the JSON value and nil, or an empty JSON value and an error if unsupported.
std::pair<nlohmann::json, xerrors::Error>
from_sample_value(const telem::SampleValue &value, Type target);

/// @brief checks at config time whether a DataType can be converted to the given JSON
/// Type.
/// @param type the Synnax DataType to check.
/// @param target the JSON type to convert to.
/// @returns nil if supported, UNSUPPORTED_ERROR otherwise.
xerrors::Error check_from_sample_value(const telem::DataType &type, Type target);

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
