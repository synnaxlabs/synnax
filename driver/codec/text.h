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
#include <cstdint>
#include <optional>
#include <span>
#include <string>
#include <string_view>
#include <utility>
#include <vector>

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

#include "driver/codec/binding.h"
#include "driver/codec/field.h"

namespace driver::codec {
/// @brief TextMessage is a text message and the fields of its lines that take samples.
struct TextMessage {
    /// @brief name identifies the message in errors.
    std::string name;
    /// @brief payload is the layout of the message's lines.
    synnax::icd::TextPayload payload;
    /// @brief bindings name the fields that take samples, in sample order.
    std::vector<Binding> bindings;
};

/// @brief Text decodes lines that each belong to one of several text messages into
/// samples. It is not safe for concurrent use.
class Text {
    struct Field {
        std::string name;
        bool tagged = false;
        /// @brief tag is the text before the value of a tagged field.
        std::string tag;
        /// @brief longer holds the tags of the message that start with tag and are
        /// longer, whose items belong to their own fields.
        std::vector<std::string> longer;
        /// @brief position is the item index of a delimited field.
        std::uint32_t position = 0;
        /// @brief scaled is true when the field's scale or offset changes its value.
        bool scaled = false;
        double scale = 1;
        double offset = 0;
        /// @brief narrow is true when the field writes to a float32 channel.
        bool narrow = false;
        /// @brief fallback holds the one sample used when a line has no value for the
        /// field, or nothing when the field requires one.
        std::optional<x::telem::Series> fallback;

        static std::pair<Field, x::errors::Error> create(
            const synnax::icd::TextField &f,
            const x::telem::DataType &dt,
            std::string_view delimiter
        );
        [[nodiscard]] x::errors::Error
        append_float(double v, x::telem::Series &dst) const;
        /// @brief appends the sample of the field in items, the items of one line, or
        /// its default, to dst.
        [[nodiscard]] x::errors::Error
        decode(const std::vector<std::string_view> &items, x::telem::Series &dst) const;
    };

    struct Message {
        std::string name;
        std::string delimiter;
        std::vector<Field> fields;
        std::vector<x::telem::Series> samples;
    };

    /// @brief Prefixed is a message whose lines start with prefix.
    struct Prefixed {
        std::string prefix;
        std::size_t message = 0;
    };

    std::vector<Message> messages;
    /// @brief prefixed holds the messages with a prefix, longest prefix first.
    std::vector<Prefixed> prefixed;
    /// @brief fallback is the message with no prefix.
    std::optional<std::size_t> fallback;
    /// @brief items is scratch space for the items of a line.
    std::vector<std::string_view> items;

    /// @brief routes the lines that start with prefix to the message at index i.
    /// @returns CONFIG_ERROR when an earlier message has the same prefix.
    x::errors::Error
    route(const std::string &name, const std::string &prefix, std::size_t i);

public:
    /// @brief creates a Text decoder for messages that share one source of lines. A
    /// line belongs to the message with the longest prefix that starts it, or else to
    /// the message with no prefix.
    /// @returns CONFIG_ERROR when two messages have the same prefix or no prefix, a
    /// delimiter or tag is empty, a tag contains its delimiter or starts with
    /// whitespace, a binding names no field of its message, a default is not finite or
    /// does not fit its channel, or a field does not convert to the data type of its
    /// binding.
    static std::pair<Text, x::errors::Error>
    create(const std::vector<TextMessage> &messages);

    /// @brief finds the message that line belongs to and decodes one sample for each of
    /// its bindings into its samples. The line splits into items on the delimiter of
    /// its message. A delimited field reads the item at its position, and a tagged
    /// field reads the rest of the first item whose longest tag of the message is its
    /// tag. A value is a decimal number, with any spaces, tabs, or carriage returns
    /// around it. An empty or missing item has no value.
    /// @returns the index of the message in create order, or nothing when line belongs
    /// to no message. A line of only spaces, tabs, and carriage returns belongs to
    /// none.
    /// @returns VALIDATION when a field with no default has no value in line, a value
    /// is not a number, or a scaled value is not finite or does not fit its channel.
    /// The error names the field, and also the message when the decoder holds more than
    /// one.
    std::pair<std::optional<std::size_t>, x::errors::Error>
    decode(std::string_view line);

    /// @brief returns the samples of the last decode of message: one series for each
    /// binding, in binding order, each holding one sample. They are unspecified after a
    /// decode that does not return message.
    [[nodiscard]] const std::vector<x::telem::Series> &
    samples(const std::size_t message) const {
        return this->messages[message].samples;
    }
};
/// @brief TextEncoder encodes samples into the line of one text message. It is not safe
/// for concurrent use.
class TextEncoder {
    struct Field {
        synnax::icd::TextBaseField base;
        /// @brief tag is the text before the value of a tagged field, or empty for a
        /// delimited field.
        std::string tag;
        bool scaled = false;
        Number number = Number::FLOAT64;
        /// @brief binding is the index of the field's sample, or nothing for a field
        /// that takes its default.
        std::optional<std::size_t> binding;
        std::optional<double> fallback;

        /// @brief creates a field that writes f. dt is the data type of the field's
        /// channel, or nothing for a field that takes its default.
        static std::pair<Field, x::errors::Error> create(
            const synnax::icd::TextField &f,
            const std::optional<x::telem::DataType> &dt,
            std::string_view delimiter
        );
        /// @brief appends the text of the field's sample in samples, or of its default,
        /// to out.
        [[nodiscard]] x::errors::Error
        append(std::span<const x::telem::Series> samples, std::string &out) const;
        [[nodiscard]] x::errors::Error
        append(const x::telem::SampleValue &v, std::string &out) const;
    };

    /// @brief Item is one item of a line before its tagged fields.
    struct Item {
        /// @brief text is the part of the prefix that fills the item.
        std::string text;
        /// @brief field is the index in fields of the delimited field at the item, or
        /// -1 for an item that holds only text.
        int field = -1;
    };

    std::string name;
    std::string delimiter;
    std::vector<Field> fields;
    std::vector<Item> items;
    /// @brief tagged holds the index in fields of each tagged field, in field order.
    std::vector<std::size_t> tagged;
    /// @brief bound holds the index in fields of the field of each binding.
    std::vector<std::size_t> bound;
    /// @brief line is scratch space for the text of a line.
    std::string line;

public:
    /// @brief creates an encoder of message. Each binding names a field that takes a
    /// sample, in sample order, and every other field takes its default.
    /// @returns CONFIG_ERROR when the delimiter is empty, a binding names no field, a
    /// field has two bindings, a tag is empty, contains the delimiter, or starts with
    /// whitespace, two fields have the same position, a position is past 65535, the
    /// prefix fills the position of a field, a default is not finite, a field's channel
    /// does not hold numbers, a scaled field's channel does not hold floats, or a field
    /// has no binding and no default.
    static std::pair<TextEncoder, x::errors::Error> create(const TextMessage &message);

    /// @brief encodes samples into payload, a line that starts with the prefix. The
    /// line holds an item for each position up to the last delimited field, with the
    /// prefix in the first items and an empty item where no field is. A tagged field
    /// comes after them as its tag and value. A value is the shortest decimal text that
    /// reads back as the sample after its scale and offset come off. A float32 sample
    /// counts as its shortest decimal text, so it gains no float64 digits.
    /// @param samples holds one series for each binding, in binding order. Each holds
    /// the field's sample at index 0 and has the data type of its binding, or is empty
    /// for a field that takes its default.
    /// @returns VALIDATION when a field has no sample and no default, or a value is not
    /// finite. The error names the field.
    x::errors::Error encode(
        std::span<const x::telem::Series> samples,
        std::vector<std::uint8_t> &payload
    );

    /// @brief returns the error that encode returns when binding holds sample i of s,
    /// or VALIDATION when the sample is outside the min and max of the field of
    /// binding. A float32 sample that equals a limit at float32 precision is at that
    /// limit.
    [[nodiscard]] x::errors::Error
    check(std::size_t binding, const x::telem::Series &s, std::size_t i);

    /// @brief returns true when the field of binding has a default, which encode writes
    /// for an empty series.
    [[nodiscard]] bool has_default(std::size_t binding) const {
        return this->fields[this->bound[binding]].fallback.has_value();
    }
};
}
