// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <array>
#include <compare>
#include <cstddef>
#include <cstdint>
#include <map>
#include <optional>
#include <string>
#include <string_view>
#include <unordered_map>
#include <utility>
#include <vector>

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/extract.h"
#include "x/cpp/json/field.h"
#include "x/cpp/json/pointer.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

namespace driver::codec {
/// @brief Binding names a field of a message and the data type of the channel it feeds.
struct Binding {
    /// @brief field is the key of the message field to read.
    synnax::icd::FieldKey field;
    /// @brief data_type is the data type of the channel that the field writes to.
    x::telem::DataType data_type;
};

/// @brief JSONMessage is a JSON message and the fields to decode from its documents.
struct JSONMessage {
    /// @brief name identifies the message in errors.
    std::string name;
    /// @brief payload is the layout of the message's documents.
    synnax::icd::JSONPayload payload;
    /// @brief bindings name the fields to decode, in sample order.
    std::vector<Binding> bindings;
};

/// @brief JSON decodes documents that each belong to one of several JSON messages into
/// samples. It is not safe for concurrent use.
class JSON {
    /// @brief Integer is the value of an integer or unsigned match rule.
    struct Integer {
        /// @brief bits holds the value in two's complement.
        std::uint64_t bits = 0;
        bool negative = false;

        auto operator<=>(const Integer &) const = default;
        [[nodiscard]] double to_float64() const;
        [[nodiscard]] std::string to_string() const;
    };

    /// @brief Router finds the message that a document belongs to. Every match rule
    /// tests the value at one pointer, so one lookup on that value finds the message.
    struct Router {
        /// @brief ruled is true when a message has a match rule.
        bool ruled = false;
        x::json::Pointer pointer;
        std::map<std::string, std::size_t, std::less<>> strings;
        std::array<std::optional<std::size_t>, 2> booleans;
        std::map<Integer, std::size_t> integers;
        std::unordered_map<double, std::size_t> floats;
        /// @brief fallback is the message with no match rule.
        std::optional<std::size_t> fallback;
        /// @brief text is scratch space for the text of a string value.
        std::string text;

        static std::pair<Router, x::errors::Error>
        create(const std::vector<JSONMessage> &messages);
        /// @brief maps the value of match to message. It returns the message that
        /// already matches that value, and the value as text. rounded holds an integer
        /// rule for each float that one rounds to.
        std::pair<std::optional<std::size_t>, std::string> claim_match(
            const synnax::icd::JSONMatch &match,
            std::size_t message,
            std::unordered_map<double, Integer> &rounded
        );
        std::optional<std::size_t> claim_integer(
            Integer k,
            std::size_t message,
            std::unordered_map<double, Integer> &rounded
        );
        /// @brief returns the message of a document whose value at the pointer is
        /// value, or nothing when the document belongs to no message.
        std::pair<std::optional<std::size_t>, x::errors::Error>
        find(const std::optional<std::string_view> &value);
    };

    struct Field {
        std::string name;
        x::json::Pointer pointer;
        /// @brief boolean is true for a field of the boolean kind.
        bool boolean = false;
        x::telem::DataType data_type;
        /// @brief scaled is true when the field's scale or offset changes its value.
        bool scaled = false;
        double scale = 1;
        double offset = 0;
        /// @brief number reads the JSON number of a scaled field as a float64 into
        /// scratch.
        x::json::Decoder number;
        x::telem::Series scratch{x::telem::FLOAT64_T, 1};
        /// @brief sample converts a JSON value to a sample of the channel's data type.
        x::json::Decoder sample;
        /// @brief fallback is the sample used when a document has no value, or nothing
        /// when the field requires one.
        std::optional<x::telem::SampleValue> fallback;

        static std::pair<Field, x::errors::Error>
        create(const synnax::icd::JSONField &f, const x::telem::DataType &dt);
        x::errors::Error decode(std::string_view raw, x::telem::Series &dst);
        [[nodiscard]] x::errors::Error
        append_float(double v, x::telem::Series &dst) const;
    };

    struct Message {
        std::string name;
        std::vector<Field> fields;
        /// @brief first_value is the index in JSON::values of the value of the first
        /// field.
        std::size_t first_value = 0;
        std::vector<x::telem::Series> samples;

        /// @brief decodes the value of each field in values into samples.
        x::errors::Error
        decode(const std::vector<std::optional<std::string_view>> &values);
    };

    Router router;
    std::vector<Message> messages;
    x::json::Extractor extractor{{}};
    /// @brief values holds the match value, when a message has a match rule, then the
    /// value of each field of each message, in message order.
    std::vector<std::optional<std::string_view>> values;

public:
    /// @brief creates a JSON decoder for messages that share one source of documents. A
    /// document belongs to the message whose match rule it meets, or else to the
    /// message with no rule.
    /// @returns CONFIG_ERROR when two match rules test different pointers, one document
    /// value can meet two match rules, two messages have no match rule, a binding names
    /// no field of its message, or a field does not convert to the data type of its
    /// binding.
    static std::pair<JSON, x::errors::Error>
    create(const std::vector<JSONMessage> &messages);

    /// @brief finds the message that doc belongs to and decodes one sample for each of
    /// its bindings into its samples.
    /// @returns the index of the message in create order, or nothing when doc belongs
    /// to no message.
    /// @returns VALIDATION when doc is not JSON or a value does not convert. The error
    /// names the field, and also the message when the decoder holds more than one.
    std::pair<std::optional<std::size_t>, x::errors::Error>
    decode(std::string_view doc);

    /// @brief returns the samples of the last decode of message: one series for each
    /// binding, in binding order, each holding one sample. They are unspecified after a
    /// decode that does not return message.
    [[nodiscard]] const std::vector<x::telem::Series> &
    samples(const std::size_t message) const {
        return this->messages[message].samples;
    }
};
}
