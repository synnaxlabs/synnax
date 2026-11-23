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
#include <map>
#include <optional>
#include <span>
#include <string>
#include <unordered_map>
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

    /// @brief Condition selects a field when its multiplexor holds one of values.
    struct Condition {
        /// @brief mux indexes the multiplexors of the message, or is -1 for a field
        /// that every frame holds.
        int mux = -1;
        std::vector<std::int64_t> values;
    };

    /// @brief Mux is a field that selects which other fields a frame holds.
    struct Mux {
        std::string name;
        BitRange bits;
        bool is_signed = false;
        Condition condition;
    };

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
        Condition condition;

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

    /// @brief Layout maps the key of each ICD field of a message to the field.
    using Layout = std::map<synnax::icd::FieldKey, const synnax::icd::BinaryField *>;

    struct Message {
        std::string name;
        std::vector<Field> fields;
        /// @brief muxes holds the multiplexors of the fields, each after the
        /// multiplexor that selects it.
        std::vector<Mux> muxes;
        /// @brief mux_end is the number of payload bytes that the multiplexors need.
        std::size_t mux_end = 0;
        /// @brief raw holds the raw value of each multiplexor in the last decode.
        std::vector<std::int64_t> raw;
        std::vector<x::telem::Series> samples;

        /// @brief returns the condition that selects field f, and adds the multiplexors
        /// that it reads to muxes. index maps the key of each multiplexor to its index
        /// in muxes, and depth is the number of multiplexors below f in its chain.
        /// @returns CONFIG_ERROR when a multiplexor does not exist, is a float field,
        /// selects on no values, or is in a cycle of multiplexors.
        std::pair<Condition, x::errors::Error> resolve(
            const synnax::icd::BinaryBaseField &f,
            const Layout &fields,
            std::map<synnax::icd::FieldKey, int> &index,
            std::size_t depth
        );
        /// @brief reports whether the multiplexors in the last decode select a field.
        [[nodiscard]] bool selected(const Condition &c) const;
        /// @brief decodes the value of each field in payload into samples.
        x::errors::Error decode(std::span<const std::uint8_t> payload);
    };

    /// @brief Header matches frames by the raw bits of an identifier field. Messages
    /// whose identifier fields read the same bits share a header.
    struct Header {
        BitRange bits;
        std::unordered_map<std::uint64_t, std::size_t> messages;
    };

    /// @brief Masked matches the CAN frames whose identifier, under mask, is id.
    struct Masked {
        std::uint32_t id = 0;
        std::uint32_t mask = 0;
        bool extended = false;
        std::size_t message = 0;
    };

    std::vector<Message> messages;
    /// @brief exact maps a CAN identifier, with bit 31 set when it is extended, to its
    /// message.
    std::unordered_map<std::uint32_t, std::size_t> exact;
    std::vector<Masked> masked;
    std::vector<Header> headers;
    /// @brief fallback is the message with no identifier.
    std::optional<std::size_t> fallback;

    /// @brief adds the identifier of message m, the message at index i, to the rules.
    /// @returns CONFIG_ERROR when the identifier is invalid, it and the identifier of
    /// an earlier message match the same CAN identifier or value, or both messages have
    /// no identifier.
    x::errors::Error route(const BinaryMessage &m, std::size_t i, const Layout &fields);
    x::errors::Error route_can(const synnax::icd::CanIdentifier &id, std::size_t i);
    /// @brief decodes payload as the message at index i.
    std::pair<std::optional<std::size_t>, x::errors::Error>
    decode_as(std::optional<std::size_t> i, std::span<const std::uint8_t> payload);

public:
    /// @brief creates a Binary decoder for messages that share one source of frames. A
    /// frame belongs to the message whose identifier it meets, or else to the message
    /// with no identifier.
    /// @returns CONFIG_ERROR when two messages have no identifier, two identifiers
    /// match the same CAN identifier or the same value of one field layout, an
    /// identifier is invalid, a binding names no field of its message, a field's
    /// layout, default, or multiplexor is invalid, or a field does not convert to the
    /// data type of its binding.
    static std::pair<Binary, x::errors::Error>
    create(const std::vector<BinaryMessage> &messages);

    /// @brief finds the message that payload, a frame from a byte stream or a datagram,
    /// belongs to by its field identifiers, and decodes its bindings into its samples.
    /// A message with a CAN identifier never matches.
    /// @returns the index of the message in create order, or nothing when payload
    /// belongs to no message.
    /// @returns VALIDATION when payload meets the identifiers of two messages, it is
    /// too short for its multiplexors or for a field with no default, or a value does
    /// not fit its channel. A decode error names the field, and also the message when
    /// the decoder holds more than one.
    std::pair<std::optional<std::size_t>, x::errors::Error>
    decode(std::span<const std::uint8_t> payload);

    /// @brief finds the message that a CAN frame belongs to by its arbitration
    /// identifier, and decodes its payload as decode does. A message with a field
    /// identifier never matches.
    /// @param id the arbitration identifier of the frame.
    /// @param extended true when id is a 29-bit extended identifier.
    std::pair<std::optional<std::size_t>, x::errors::Error>
    decode(std::uint32_t id, bool extended, std::span<const std::uint8_t> payload);

    /// @brief returns the samples of the last decode of message: one series for each
    /// binding, in binding order. Each holds one sample, or none for a field that the
    /// multiplexors of the frame do not select and that has no default. They are
    /// unspecified after a decode that does not return message.
    [[nodiscard]] const std::vector<x::telem::Series> &
    samples(const std::size_t message) const {
        return this->messages[message].samples;
    }
};
}
