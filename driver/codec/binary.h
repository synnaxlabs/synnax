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
#include <utility>
#include <vector>

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

#include "driver/codec/binding.h"
#include "driver/codec/bits.h"

namespace driver::codec {
/// @brief BinaryMessage is a binary message and the fields to decode from its frames.
struct BinaryMessage {
    /// @brief name identifies the message in errors.
    std::string name;
    /// @brief payload is the layout of the message's frames.
    synnax::icd::BinaryPayload payload;
    /// @brief bindings name the fields to decode, in sample order.
    std::vector<Binding> bindings;
};

/// @brief Binary decodes frames that each belong to one of several binary messages into
/// samples. It is not safe for concurrent use.
class Binary {
    /// @brief Type is how a field reads its raw bits.
    enum class Type : std::uint8_t { UNSIGNED, SIGNED, FLOAT32, FLOAT64 };

    struct Field {
        std::string name;
        BitRange bits;
        Type type = Type::UNSIGNED;
        /// @brief scaled is true when the field's scale or offset changes its value.
        bool scaled = false;
        double scale = 1;
        double offset = 0;
        /// @brief put writes the raw bits of an unscaled integer field to a channel of
        /// an integer type, or is null when the field's values go through a float.
        void (*put)(x::telem::Series &dst, std::uint64_t raw) = nullptr;
        /// @brief narrow is true when a float value goes to a float32 channel.
        bool narrow = false;
        /// @brief fallback holds the one sample used when a frame does not hold the
        /// field, or nothing when the field requires one.
        std::optional<x::telem::Series> fallback;

        static std::pair<Field, x::errors::Error>
        create(const synnax::icd::BinaryField &f, const x::telem::DataType &dt);
        /// @brief appends the sample of raw, the raw bits of the field, to dst.
        [[nodiscard]] x::errors::Error
        append(std::uint64_t raw, x::telem::Series &dst) const;
        [[nodiscard]] x::errors::Error
        append_float(double v, x::telem::Series &dst) const;
        /// @brief appends the sample of the field in payload, or its default, to dst.
        [[nodiscard]] x::errors::Error
        decode(std::span<const std::uint8_t> payload, x::telem::Series &dst) const;
    };

    struct Message {
        std::string name;
        std::vector<Field> fields;
        std::vector<x::telem::Series> samples;

        /// @brief decodes the value of each field in payload into samples.
        x::errors::Error decode(std::span<const std::uint8_t> payload);
    };

    std::vector<Message> messages;
    /// @brief fallback is the message with no identifier.
    std::optional<std::size_t> fallback;

public:
    /// @brief creates a Binary decoder for messages that share one source of frames.
    /// @returns CONFIG_ERROR when a message has an identifier, two messages have none,
    /// a binding names no field of its message, a field's layout or default is invalid,
    /// or a field does not convert to the data type of its binding.
    static std::pair<Binary, x::errors::Error>
    create(const std::vector<BinaryMessage> &messages);

    /// @brief finds the message that payload belongs to and decodes one sample for each
    /// of its bindings into its samples.
    /// @returns the index of the message in create order, or nothing when payload
    /// belongs to no message.
    /// @returns VALIDATION when payload is too short for a field with no default, or a
    /// value does not fit its channel. The error names the field, and also the message
    /// when the decoder holds more than one.
    std::pair<std::optional<std::size_t>, x::errors::Error>
    decode(std::span<const std::uint8_t> payload);

    /// @brief returns the samples of the last decode of message: one series for each
    /// binding, in binding order, each holding one sample. They are unspecified after a
    /// decode that does not return message.
    [[nodiscard]] const std::vector<x::telem::Series> &
    samples(const std::size_t message) const {
        return this->messages[message].samples;
    }
};
}
