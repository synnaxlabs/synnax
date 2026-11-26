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
#include "driver/codec/mux.h"

namespace driver::codec {
/// @brief BinaryMessage is a binary message and the fields of its frames to decode or
/// encode.
struct BinaryMessage {
    /// @brief name identifies the message in errors.
    std::string name;
    /// @brief payload is the layout of the message's frames.
    synnax::icd::BinaryPayload payload;
    /// @brief bindings name the fields to decode or encode, in sample order.
    std::vector<Binding> bindings;
};

/// @brief Binary decodes frames that each belong to one of several binary messages into
/// samples. It is not safe for concurrent use.
class Binary {
    struct Field {
        std::string name;
        BitRange bits;
        Encoding encoding = Encoding::UNSIGNED;
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
        Multiplexors::Condition condition;

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
        Multiplexors muxes;
        /// @brief raw holds the raw value of each multiplexor in the last decode.
        std::vector<std::int64_t> raw;
        std::vector<x::telem::Series> samples;

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
    x::errors::Error
    route(const BinaryMessage &m, std::size_t i, const Multiplexors::Layout &fields);
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
    /// belongs to no message. When payload meets the identifiers of two messages, the
    /// first of them.
    /// @returns VALIDATION, with the message, when payload meets the identifiers of two
    /// messages, it is too short for its multiplexors or for a field with no default,
    /// or a value does not fit its channel. A decode error names the field, and also
    /// the message when the decoder holds more than one.
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
    /// unspecified after a decode that does not return message, or returns an error.
    [[nodiscard]] const std::vector<x::telem::Series> &
    samples(const std::size_t message) const {
        return this->messages[message].samples;
    }
};
/// @brief BinaryEncoder encodes samples into the payload of one binary message. It is
/// not safe for concurrent use.
class BinaryEncoder {
    /// @brief Number is the kind of the samples of a field's channel.
    enum class Number : std::uint8_t { FLOAT, SIGNED, UNSIGNED };

    struct Field {
        /// @brief base holds the name and the min and max of the field.
        synnax::icd::BinaryBaseField base;
        BitRange bits;
        Encoding encoding = Encoding::UNSIGNED;
        bool scaled = false;
        double scale = 1;
        double offset = 0;
        Number number = Number::UNSIGNED;
        /// @brief binding is the index of the field's sample, or nothing for a field
        /// that takes its default.
        std::optional<std::size_t> binding;
        /// @brief fallback holds the raw bits of the field's default, or nothing.
        std::optional<std::uint64_t> fallback;
        Multiplexors::Condition condition;

        /// @param dt is the data type of the field's channel, or nothing for a field
        /// with no binding.
        static std::pair<Field, x::errors::Error> create(
            const synnax::icd::BinaryField &f,
            const std::optional<x::telem::DataType> &dt
        );
        /// @brief returns true when samples hold the field's sample, or the field has
        /// a default.
        [[nodiscard]] bool has_value(std::span<const x::telem::Series> samples) const;
        /// @brief returns the raw bits of the field's sample in samples, or of its
        /// default.
        [[nodiscard]] std::pair<std::uint64_t, x::errors::Error>
        raw(std::span<const x::telem::Series> samples) const;
        [[nodiscard]] std::pair<std::uint64_t, x::errors::Error>
        raw(const x::telem::SampleValue &v) const;
        [[nodiscard]] std::pair<std::uint64_t, x::errors::Error>
        raw_float(double v) const;
    };

    /// @brief name identifies the message in errors.
    std::string name;
    std::vector<Field> fields;
    Multiplexors muxes;
    /// @brief mux_fields holds the index in fields of each multiplexor.
    std::vector<std::size_t> mux_fields;
    /// @brief raw holds the raw value of each multiplexor in the last encode.
    std::vector<std::int64_t> raw;
    /// @brief bound holds the index in fields of the field of each binding.
    std::vector<std::size_t> bound;
    std::size_t length = 0;

public:
    /// @brief creates an encoder of message. Each binding names a field that takes a
    /// sample, in sample order, and every other field takes its default. The identifier
    /// field takes the identifier's value.
    /// @returns CONFIG_ERROR when a binding names no field, a field has two bindings, a
    /// binding names the identifier field, a field's layout, default, or multiplexor is
    /// invalid, a field's channel does not hold numbers, a scaled field's channel does
    /// not hold floats, a field that the multiplexors can select has no binding and no
    /// default, or a fixed multiplexor value never selects a bound field.
    static std::pair<BinaryEncoder, x::errors::Error>
    create(const BinaryMessage &message);

    /// @brief encodes samples into payload. A field that the multiplexors do not select
    /// is not written, and bits that no field holds are zero.
    /// @param samples holds one series for each binding, in binding order. Each holds
    /// the field's sample at index 0 and has the data type of its binding, or is empty
    /// for a field that takes its default.
    /// @returns VALIDATION when a selected field has no sample and no default, or a
    /// sample does not fit its field. The error names the message and the field.
    x::errors::Error encode(
        std::span<const x::telem::Series> samples,
        std::vector<std::uint8_t> &payload
    );

    /// @brief returns the error that encode returns when binding holds sample i of s
    /// and its field is selected, or VALIDATION when the raw value of the sample
    /// decodes outside the min and max of the field of binding. When the sample or the
    /// field is float32, a value that decodes to a limit at float32 precision is at
    /// that limit.
    [[nodiscard]] x::errors::Error
    check(std::size_t binding, const x::telem::Series &s, std::size_t i) const;

    /// @brief returns true when each field that the multiplexors select in samples has
    /// a sample or a default, so encode has a value for it.
    /// @param samples is as in encode.
    [[nodiscard]] bool ready(std::span<const x::telem::Series> samples);
};
}
