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
#include <string>
#include <string_view>
#include <utility>
#include <vector>

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

#include "driver/codec/binding.h"

namespace driver::codec {
/// @brief TextMessage is a text message and the fields to decode from its lines.
struct TextMessage {
    /// @brief name identifies the message in errors.
    std::string name;
    /// @brief payload is the layout of the message's lines.
    synnax::icd::TextPayload payload;
    /// @brief bindings name the fields to decode, in sample order.
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
    /// field reads the rest of the first item that starts with its tag. A value is a
    /// decimal number, with any spaces, tabs, or carriage returns around it. An empty
    /// or missing item has no value.
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
}
