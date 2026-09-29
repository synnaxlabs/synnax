// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <cstddef>
#include <cstdint>
#include <span>
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/library/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

#include "driver/codec/bits.h"

namespace driver::codec {
/// @brief Values holds one value per plan slot and whether each slot is present. Get
/// one from Plan::values() and reuse it across frames, so that decode allocates
/// nothing.
class Values {
public:
    Values() = default;

    /// @returns the number of slots.
    [[nodiscard]] std::size_t size() const { return this->slots.size(); }

    /// @returns true when the slot holds a value.
    [[nodiscard]] bool present(const std::size_t slot) const {
        return this->slots[slot].kind != Kind::ABSENT;
    }

    /// @returns the physical value of the slot. Integers beyond 2^53 lose precision
    /// here; use write() to keep them exact.
    [[nodiscard]] double get(const std::size_t slot) const {
        return this->slots[slot].value;
    }

    /// @returns the number of text fields the last decode could not read, either
    /// because the item was missing or because it did not parse as a number.
    [[nodiscard]] std::size_t invalid() const { return this->invalid_; }

    /// @brief sets the physical value of the slot and marks it present.
    void set(std::size_t slot, double value);

    /// @brief sets an exact integer value. Binary integer fields with a scale of 1 and
    /// an offset of 0 encode it without a round trip through double.
    void set(std::size_t slot, std::int64_t value);

    /// @brief sets an exact unsigned integer value. See set(std::size_t, int64_t).
    void set(std::size_t slot, std::uint64_t value);

    /// @brief sets the slot from a sample of a command channel.
    /// @returns ENCODE_ERROR when the sample is a string.
    x::errors::Error set(std::size_t slot, const x::telem::SampleValue &value);

    /// @brief marks every slot absent.
    void clear();

    /// @brief appends the slot's value to the series, cast to the series' data type.
    /// Integer fields with a scale of 1 and an offset of 0 are written exactly.
    /// @returns the number of samples written: 0 when the slot is absent or the series
    /// is full, 1 otherwise.
    std::size_t write(std::size_t slot, x::telem::Series &series) const;

private:
    friend class Plan;

    Values(const std::size_t size, const std::size_t muxes):
        slots(size), muxes(muxes) {}

    /// @brief Kind is how a slot holds its value.
    enum class Kind : std::uint8_t { ABSENT, FLOAT, INT, UINT };

    /// @brief Slot is one value.
    struct Slot {
        /// @brief value is the physical value.
        double value = 0;
        /// @brief integer is the exact integer bits when kind is INT or UINT.
        std::uint64_t integer = 0;
        /// @brief kind is how the slot holds its value.
        Kind kind = Kind::ABSENT;
    };

    /// @brief slots holds one entry per plan slot.
    std::vector<Slot> slots;
    /// @brief muxes holds the raw value of each plan multiplexor in the last decode.
    std::vector<std::int64_t> muxes;
    /// @brief invalid_ is the number of text fields the last decode could not read.
    std::size_t invalid_ = 0;
};

/// @brief Plan decodes a message payload into field values and encodes values into a
/// payload. Compile it once when a task configures. Decode and encode are const and
/// safe to call from several threads with separate Values.
class Plan {
public:
    Plan() = default;

    /// @brief compiles a plan whose slots are every field of the message, in order.
    /// @returns CONFIG_ERROR when the message layout is invalid.
    static std::pair<Plan, x::errors::Error>
    compile(const synnax::library::MessageEntry &message);

    /// @brief compiles a plan whose slots are the given fields, in the given order.
    /// Multiplexors of the given fields are resolved even when not in keys.
    /// @returns CONFIG_ERROR when a key is missing or duplicated, or when the layout of
    /// a field the plan needs is invalid.
    static std::pair<Plan, x::errors::Error> compile(
        const synnax::library::MessageEntry &message,
        const std::vector<synnax::library::FieldKey> &keys
    );

    /// @returns the number of slots.
    [[nodiscard]] std::size_t size() const { return this->keys.size(); }

    /// @returns the key of the field in the slot.
    [[nodiscard]] const synnax::library::FieldKey &key(const std::size_t slot) const {
        return this->keys[slot];
    }

    /// @returns the number of bytes a binary encode writes at least: the message length
    /// when set, and otherwise the bytes the plan's fields span.
    [[nodiscard]] std::size_t length() const { return this->length_; }

    /// @returns values with one absent slot per plan slot.
    [[nodiscard]] Values values() const {
        return Values(this->size(), this->muxes.size());
    }

    /// @brief decodes a payload. Binary fields whose multiplexor does not select them
    /// come out absent. Text fields that are missing or fail to parse come out absent
    /// and are counted in Values::invalid().
    /// @param payload the frame payload, or the line for a text message.
    /// @param values the output, from values(). Its contents are unspecified when
    /// decode returns an error.
    /// @returns SHORT_PAYLOAD_ERROR when the payload does not hold a field that the
    /// multiplexors select.
    x::errors::Error
    decode(std::span<const std::uint8_t> payload, Values &values) const;

    /// @brief encodes values into a payload. For binary messages, grows the payload
    /// with zero bytes to length() and writes each present field that its multiplexor
    /// selects, leaving every other bit as it was. For text messages, replaces the
    /// payload with the delimited items in position order followed by each present
    /// tagged field, skipping absent ones.
    /// @returns ENCODE_ERROR when a binary integer field receives NaN.
    x::errors::Error
    encode(const Values &values, std::vector<std::uint8_t> &payload) const;

    /// @brief encodes values into a binary payload in place, writing each present
    /// field that its multiplexor selects and leaving every other bit as it was.
    /// @param payload holds at least length() bytes, such as a window into a larger
    /// buffer.
    /// @returns ENCODE_ERROR when the plan is text, when the payload is shorter than
    /// length(), or when a binary integer field receives NaN.
    x::errors::Error
    encode(const Values &values, std::span<std::uint8_t> payload) const;

private:
    /// @brief Type is how a binary field interprets its raw bits.
    enum class Type : std::uint8_t { UNSIGNED, SIGNED, FLOAT32, FLOAT64 };

    /// @brief Condition selects a field when a multiplexor holds one of values.
    struct Condition {
        /// @brief multiplexor indexes muxes, or is -1 when the field is always present.
        int multiplexor = -1;
        /// @brief values are the raw multiplexor values that select the field.
        std::vector<std::int64_t> values;
    };

    /// @brief Mux is a field that selects other fields.
    struct Mux {
        /// @brief bits is where the multiplexor lies.
        BitRange bits;
        /// @brief signed_ is true when the raw value is two's complement.
        bool signed_ = false;
        /// @brief condition selects the multiplexor itself.
        Condition condition;
    };

    /// @brief Binary is a compiled binary field.
    struct Binary {
        /// @brief name is the field name, for errors.
        std::string name;
        /// @brief slot is the index of the field's value.
        std::size_t slot = 0;
        /// @brief bits is where the field lies.
        BitRange bits;
        /// @brief type interprets the raw bits.
        Type type = Type::UNSIGNED;
        /// @brief scale multiplies the raw value on decode.
        double scale = 1;
        /// @brief offset is added to the scaled value on decode.
        double offset = 0;
        /// @brief exact is true for integer fields with scale 1 and offset 0.
        bool exact = false;
        /// @brief condition selects the field.
        Condition condition;
    };

    /// @brief Text is a compiled text field.
    struct Text {
        /// @brief slot is the index of the field's value.
        std::size_t slot = 0;
        /// @brief scale multiplies the parsed value on decode.
        double scale = 1;
        /// @brief offset is added to the scaled value on decode.
        double offset = 0;
        /// @brief position is the item index of a delimited field.
        std::uint32_t position = 0;
        /// @brief tag is the text before the value of a tagged field.
        std::string tag;
    };

    /// @returns the raw value of the multiplexor in payload.
    static std::int64_t read(const Mux &mux, const std::uint8_t *payload);

    /// @returns true when each multiplexor in the condition's chain holds one of its
    /// values.
    /// @param raw returns the raw value of the multiplexor at an index of muxes.
    template<typename Raw>
    [[nodiscard]] bool selected(const Condition &condition, const Raw &raw) const {
        for (const Condition *c = &condition; c->multiplexor >= 0;
             c = &this->muxes[c->multiplexor].condition)
            if (std::ranges::find(c->values, raw(c->multiplexor)) == c->values.end())
                return false;
        return true;
    }
    /// @brief compiles the fields of keys from a binary payload.
    /// @returns CONFIG_ERROR when the layout of a field the plan needs is invalid.
    x::errors::Error compile_payload(const synnax::library::BinaryPayload &payload);
    /// @brief compiles the fields of keys from a text payload.
    /// @returns CONFIG_ERROR when the delimiter or a field the plan needs is invalid.
    x::errors::Error compile_payload(const synnax::library::TextPayload &payload);
    x::errors::Error
    decode_binary(std::span<const std::uint8_t> payload, Values &values) const;
    void decode_text(std::span<const std::uint8_t> payload, Values &values) const;
    x::errors::Error encode_binary(const Values &values, std::uint8_t *payload) const;
    void encode_text(const Values &values, std::vector<std::uint8_t> &payload) const;

    /// @brief keys is the field key of each slot.
    std::vector<synnax::library::FieldKey> keys;
    /// @brief text is true for text messages.
    bool text = false;
    /// @brief length_ is the minimum length of an encoded binary payload.
    std::size_t length_ = 0;
    /// @brief binary holds binary fields, with each multiplexor before the fields it
    /// selects.
    std::vector<Binary> binary;
    /// @brief muxes holds the multiplexors the binary fields depend on.
    std::vector<Mux> muxes;
    /// @brief mux_end is the number of payload bytes the multiplexors span.
    std::size_t mux_end = 0;
    /// @brief positional holds delimited fields sorted by position.
    std::vector<Text> positional;
    /// @brief tagged holds tagged fields in declaration order.
    std::vector<Text> tagged;
    /// @brief delimiter splits the items of a text message.
    std::string delimiter;
};
}
