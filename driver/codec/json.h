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
#include <span>
#include <string>
#include <string_view>
#include <unordered_map>
#include <utility>
#include <vector>

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/convert.h"
#include "x/cpp/json/extract.h"
#include "x/cpp/json/field.h"
#include "x/cpp/json/number.h"
#include "x/cpp/json/pointer.h"
#include "x/cpp/json/template.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

#include "driver/codec/binding.h"

namespace driver::codec {
/// @brief JSONMessage is a JSON message and the fields of its documents that take
/// samples.
struct JSONMessage {
    /// @brief name identifies the message in errors.
    std::string name;
    /// @brief payload is the layout of the message's documents.
    synnax::icd::JSONPayload payload;
    /// @brief bindings name the fields that take samples, in sample order.
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

    /// @brief Router finds the message that a document belongs to. Every value rule
    /// tests the value at one pointer, so one lookup on that value finds its message.
    /// Each present rule tests a pointer of its own.
    struct Router {
        struct Present {
            x::json::Pointer pointer;
            std::size_t message = 0;
        };

        /// @brief names holds the name of each message, for errors.
        std::vector<std::string> names;
        /// @brief ruled is true when a message has a value rule.
        bool ruled = false;
        x::json::Pointer pointer;
        std::map<std::string, std::size_t, std::less<>> strings;
        std::array<std::optional<std::size_t>, 2> booleans;
        std::map<Integer, std::size_t> integers;
        std::unordered_map<double, std::size_t> floats;
        std::vector<Present> present;
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
        /// @brief returns the pointers that the rules test, in the order that find
        /// takes their values.
        [[nodiscard]] std::vector<x::json::Pointer> pointers() const;
        /// @brief returns the message of a document, or nothing when it belongs to no
        /// message. values starts with the values of the document at pointers().
        /// @returns VALIDATION, with the first of their messages, when the document
        /// meets two rules.
        std::pair<std::optional<std::size_t>, x::errors::Error>
        find(const std::vector<std::optional<std::string_view>> &values);
        /// @brief returns the message of the value rule that value meets, or nothing.
        std::pair<std::optional<std::size_t>, x::errors::Error>
        find_value(const std::optional<std::string_view> &value);
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
        /// @brief fallback holds the one sample used when a document has no value, or
        /// nothing when the field requires one.
        std::optional<x::telem::Series> fallback;
        /// @brief generate, when set, appends a new sample in place of fallback.
        void (*generate)(x::telem::Series &dst) = nullptr;

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
        std::vector<Failure> failures;

        /// @brief decodes the value of each field in values into samples, and each
        /// field that it cannot decode into failures.
        void decode(const std::vector<std::optional<std::string_view>> &values);
    };

    Router router;
    std::vector<Message> messages;
    x::json::Extractor extractor{{}};
    /// @brief values holds the values at the pointers of the router, then the value of
    /// each field of each message, in message order.
    std::vector<std::optional<std::string_view>> values;

public:
    /// @brief creates a JSON decoder for messages that share one source of documents. A
    /// document belongs to the message whose match rule it meets, or else to the
    /// message with no rule.
    /// @returns CONFIG_ERROR when two value rules test different pointers, one document
    /// value can meet two value rules, a present rule tests a pointer at or above the
    /// pointer of another rule, two messages have no match rule, a binding names no
    /// field of its message, a bound field has a zero scale, or a field does not
    /// convert to the data type of its binding.
    static std::pair<JSON, x::errors::Error>
    create(const std::vector<JSONMessage> &messages);

    /// @brief finds the message that doc belongs to and decodes one sample for each of
    /// its bindings into its samples. A field with no value and no default, or with a
    /// value that does not convert, gets no sample and goes into its failures.
    /// @returns the index of the message in create order, or nothing when doc belongs
    /// to no message or is not JSON. When doc meets two match rules, the first of their
    /// messages.
    /// @returns VALIDATION when doc is not JSON, or, with the message, when it meets
    /// two match rules.
    std::pair<std::optional<std::size_t>, x::errors::Error>
    decode(std::string_view doc);

    /// @brief returns the samples of the last decode of message: one series for each
    /// binding, in binding order, each holding one sample, or none for a field in its
    /// failures. They are unspecified after a decode that does not return message, or
    /// returns an error.
    [[nodiscard]] const std::vector<x::telem::Series> &
    samples(const std::size_t message) const {
        return this->messages[message].samples;
    }

    /// @brief returns the fields that the last decode of message could not decode, in
    /// binding order. They are unspecified after a decode that does not return message,
    /// or returns an error.
    [[nodiscard]] const std::vector<Failure> &
    failures(const std::size_t message) const {
        return this->messages[message].failures;
    }
};

/// @brief JSONEncoder encodes samples into the documents of one JSON message. It is not
/// safe for concurrent use.
class JSONEncoder {
    /// @brief Generator makes a default that takes a new value at each encode.
    enum class Generator : std::uint8_t { NONE, NOW, UUID };

    /// @brief Default is the default of a field: fixed JSON text, or a generator.
    struct Default {
        std::optional<std::string> text;
        Generator generator = Generator::NONE;
        /// @brief time_format is the format of the time that NOW writes.
        x::json::TimeFormat time_format = x::json::TimeFormat::ISO8601;

        /// @brief returns the default of f, or no default when f has none. Its errors
        /// do not name the default.
        static std::pair<Default, x::errors::Error>
        create(const synnax::icd::JSONField &f);
        [[nodiscard]] bool defined() const {
            return this->text.has_value() || this->generator != Generator::NONE;
        }
        /// @returns VALIDATION when there is no default.
        [[nodiscard]] x::errors::Error append(std::string &out) const;
    };

    struct Field {
        std::string name;
        std::string_view kind;
        /// @brief encoder writes the samples of every kind but JSON, whose samples are
        /// JSON text. A converted field uses it only to find enum labels.
        x::json::Encoder encoder;
        /// @brief converted is true when a numeric sample needs a scale, a rounding, a
        /// range check, or a change to a boolean before it is written.
        bool converted = false;
        bool scaled = false;
        double scale = 1;
        double offset = 0;
        /// @brief base holds the min and max of the field.
        synnax::icd::BaseField base;
        /// @brief fallback is the default that an empty series writes.
        Default fallback;

        static std::pair<Field, x::errors::Error>
        create(const synnax::icd::JSONField &f, const x::telem::DataType &dt);
        /// @brief appends JSON text to out: sample i of s.
        [[nodiscard]] x::errors::Error
        append(const x::telem::Series &s, std::size_t i, std::string &out) const;
        /// @brief appends JSON text to out: n, unscaled and rounded to the nearest
        /// integer. It checks the limits against the value that the integer decodes to.
        [[nodiscard]] x::errors::Error
        append_integer(const x::json::Number &n, std::string &out) const;
    };

    /// @brief Slot is a value of the document that changes between encodes.
    struct Slot {
        /// @brief binding is the binding whose sample fills the slot, or nothing for a
        /// generated default.
        std::optional<std::size_t> binding;
        /// @brief fallback is the default of a slot with no binding.
        Default fallback;
    };

    std::string name;
    /// @brief document is the fixed text of each document around its slots.
    x::json::Template document;
    /// @brief fields holds the field of each binding.
    std::vector<Field> fields;
    /// @brief slots holds the slot of each field of document.
    std::vector<Slot> slots;
    /// @brief text is scratch space for the text of an encode or a check.
    std::string text;

public:
    /// @brief creates an encoder of message. Each binding names a field that takes a
    /// sample, in sample order, and every other field takes its default. A document
    /// holds the value of the message's value rule and each field at its pointer. The
    /// fields at or inside the pointer of a present rule give it its value.
    /// @returns CONFIG_ERROR when a binding names no field, a field has two bindings, a
    /// field has no binding and no default, a default is not valid, a bound field has a
    /// zero scale, a binding's data type does not write to its field's kind, a scaled
    /// field's channel is not a float, two pointers are equal or one holds the other, a
    /// pointer holds an array index, or no field is at or inside the pointer of a
    /// present rule.
    static std::pair<JSONEncoder, x::errors::Error> create(const JSONMessage &message);

    /// @brief encodes samples into payload, one document of the message. A scaled field
    /// writes its sample before the scale and offset, and an integer field rounds it to
    /// the nearest integer. A boolean field takes only 0 and 1 from a number channel. A
    /// JSON field writes its sample as it is.
    /// @param samples holds one series for each binding, in binding order. Each holds
    /// the field's sample at index 0 and has the data type of its binding, or is empty
    /// for a field that takes its default.
    /// @returns VALIDATION when a field has no sample and no default, or a field cannot
    /// hold its sample. The error names the message and the field.
    x::errors::Error encode(
        std::span<const x::telem::Series> samples,
        std::vector<std::uint8_t> &payload
    );

    /// @brief returns the error that encode returns when binding holds sample i of s,
    /// or NIL when the field of binding can hold it.
    [[nodiscard]] x::errors::Error
    check(std::size_t binding, const x::telem::Series &s, std::size_t i);

    /// @brief returns true when each field has a sample or a default, so encode has a
    /// value for it.
    /// @param samples is as in encode.
    [[nodiscard]] bool ready(const std::span<const x::telem::Series> samples) const {
        for (std::size_t b = 0; b < samples.size(); b++)
            if (samples[b].empty() && !this->fields[b].fallback.defined()) return false;
        return true;
    }
};
}
