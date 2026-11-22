// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <string>
#include <string_view>
#include <unordered_set>
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/convert.h"
#include "x/cpp/json/number.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

namespace x::json {
/// @brief Enum pairs a JSON string label with one sample. A numeric sample converts to
/// an integer or boolean data type that holds it exactly, or to the nearest value of a
/// float data type, failing past its range. A string sample converts only to a string.
struct Enum {
    /// @brief label is the JSON string that stands for the sample.
    std::string label;
    /// @brief value is the sample. A boolean is a uint8 of 0 or 1.
    telem::SampleValue value;
};

/// @brief Field is how the values at one place in a JSON document convert to samples.
struct Field {
    /// @brief data_type is the data type of the samples.
    telem::DataType data_type;
    /// @brief time_format is how a timestamp sample is written in the document.
    TimeFormat time_format = TimeFormat::ISO8601;
    /// @brief enums label the samples of a numeric, boolean, or string data type. A
    /// Decoder reads a label as its sample.
    std::vector<Enum> enums;
    /// @brief range is an integer data type, or UNKNOWN_T. When set, a Decoder rejects
    /// a JSON number that range does not hold exactly, such as 1.5 or -1 for UINT64_T.
    telem::DataType range = telem::UNKNOWN_T;
};

/// @brief Decoder converts the JSON values of a Field into samples. It is not safe for
/// concurrent use.
class Decoder {
    /// @brief Label is an enum label and its sample in the data type of the field.
    struct Label {
        std::string label;
        /// @brief number is the sample of a numeric data type.
        Number number;
        /// @brief text is the sample of a string data type.
        std::string text;
    };

    Field field;
    std::vector<Label> labels;
    /// @brief text is scratch space for the decoded text of a JSON string.
    std::string text;
    /// @brief open and names are scratch space for compacting a JSON value.
    std::vector<std::size_t> open;
    std::vector<std::unordered_set<std::string>> names;

    errors::Error append_string(std::string_view raw, telem::Series &dst);
    errors::Error append_json(std::string_view raw, telem::Series &dst);
    errors::Error append_uuid(std::string_view raw, telem::Series &dst);
    errors::Error append_bytes(std::string_view raw, telem::Series &dst);
    errors::Error append_numeric(std::string_view raw, telem::Series &dst);
    errors::Error append_timestamp(std::string_view raw, telem::Series &dst);
    errors::Error append_number(const Number &n, telem::Series &dst) const;
    /// @brief writes n, which the data type of the field holds, to dst.
    void write_number(const Number &n, telem::Series &dst) const;
    [[nodiscard]] const Label *find_label(std::string_view text) const;

public:
    /// @brief creates a Decoder for no field. Assign one from create before use.
    Decoder() = default;

    /// @brief creates a Decoder for field.
    /// @returns VALIDATION when no JSON value converts to the data type of field, an
    /// enum value does not convert to it, or the range of field is not an integer data
    /// type.
    static std::pair<Decoder, errors::Error> create(Field field);

    /// @brief appends the sample of raw, one valid JSON value, to dst. dst holds the
    /// data type of the field and has room for one more sample.
    /// - A timestamp takes a string in ISO 8601 format, and a number or a string
    ///   holding one in a Unix format.
    /// - A string takes a string, or the label of an enum. JSON takes any value,
    ///   compacted but spelled as raw.
    /// - A UUID takes its string form.
    /// - Bytes take a string of standard, padded base64.
    /// - A numeric or boolean sample takes a number that it can hold exactly, a
    ///   boolean as 0 or 1, or the label of an enum.
    /// @returns VALIDATION, appending nothing, when raw does not convert.
    errors::Error append(std::string_view raw, telem::Series &dst);

    /// @brief appends the sample of v to dst, as append does for a JSON number equal to
    /// v.
    /// @returns VALIDATION, appending nothing, when v is not finite, the data type is
    /// not numeric, or v does not convert to it.
    errors::Error append_float(double v, telem::Series &dst) const;
};
}
