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

#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/xerrors/errors.h"

namespace xjson {

const xerrors::Error BASE_ERROR = xerrors::Error("xjson.conversion", "");
const xerrors::Error UNSUPPORTED_ERROR = BASE_ERROR.sub("unsupported");
const xerrors::Error TRUNCATION_ERROR = BASE_ERROR.sub("truncation");
const xerrors::Error OVERFLOW_ERROR = BASE_ERROR.sub("overflow");

enum class Type { Number, String, Boolean };

enum class TimeFormat {
    ISO8601,
    UnixSecondFloat,
    UnixSecondInt,
    UnixMillisecond,
    UnixMicrosecond,
    UnixNanosecond,
};

/// Resolved read converter. Takes a JSON value extracted from a response and returns a
/// single-sample Series containing the converted value.
using ReadConverter = std::function<
    std::pair<telem::Series, xerrors::Error>(const nlohmann::json &value)>;

/// Resolve a read converter for a specific (Type, DataType, strict) combination.
/// The returned function captures the exact C++ types — no branching on DataType at
/// runtime. Returns an error if the conversion is unsupported.
std::pair<ReadConverter, xerrors::Error> resolve_read_converter(
    Type json_type,
    const telem::DataType &target_type,
    bool strict = false
);

/// Convert a SampleValue to a JSON value of the given target type.
std::pair<nlohmann::json, xerrors::Error>
from_sample_value(const telem::SampleValue &value, Type target);

/// Config-time check that a DataType can be converted to the given JSON type.
/// Returns UNSUPPORTED_ERR if the conversion is not supported.
xerrors::Error check_from_sample_value(const telem::DataType &type, Type target);

/// Convert a TimeStamp to a JSON value using the given time format.
nlohmann::json from_timestamp(telem::TimeStamp ts, TimeFormat format);

/// Get zero value for a JSON format type.
///
/// Type::Number → 0, Type::String → "", Type::Boolean → false
nlohmann::json zero_value(Type format);

}
