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
#include <utility>
#include <variant>
#include <vector>

#include "client/cpp/channel/types.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/telem.h"

#include "driver/codec/binary.h"
#include "driver/codec/json.h"
#include "driver/codec/text.h"

namespace driver::icd {
/// @brief Reader reads payloads, each one of several ICD messages of one format, into
/// the channels of that message. It is not safe for concurrent use.
class Reader {
    /// @brief Index is an index channel that a message writes.
    struct Index {
        synnax::channel::Key key = 0;
        /// @brief field is the position in the bindings of the field that gives the
        /// timestamp, or nothing when the index takes the arrival time.
        std::optional<std::size_t> field;
    };
    struct Message {
        /// @brief channels holds the channel of each binding, in binding order.
        std::vector<synnax::channel::Key> channels;
        /// @brief groups holds the position in indexes of the index of each binding,
        /// or nothing for a virtual channel.
        std::vector<std::optional<std::size_t>> groups;
        /// @brief indexes holds the index channel of each bound channel once, in
        /// binding order. Virtual channels add none.
        std::vector<Index> indexes;
        /// @brief index_field is the position in the bindings of the index field.
        std::optional<std::size_t> index_field;
    };
    std::vector<Message> messages;
    std::variant<codec::Binary, codec::Text, codec::JSON> decoder;
    /// @brief present is scratch space that marks the index groups of a read whose
    /// channels all have a sample.
    std::vector<bool> present;
    /// @brief filled is scratch space that marks the bindings of a read that take its
    /// fill value.
    std::vector<bool> filled;
    /// @brief last is the message of the last read, or nothing when it returned none.
    std::optional<std::size_t> last;

    /// @brief sets the channels, the index groups, and the index field of m from the
    /// bindings of rm, which reads msg, and returns the codec binding of each.
    static std::pair<std::vector<codec::Binding>, x::errors::Error> bind(
        const synnax::icd::ReadMessage &rm,
        const synnax::icd::Message &msg,
        const std::map<synnax::channel::Key, synnax::channel::Channel> &channels,
        Message &m
    );

public:
    Reader() = default;
    Reader(Reader &&) = default;
    Reader &operator=(Reader &&) = default;
    // The decoder holds vectors of move-only fields, which hide from type traits that
    // it cannot be copied. Without these, MSVC grows a vector of readers by copying.
    Reader(const Reader &) = delete;
    Reader &operator=(const Reader &) = delete;

    /// @brief creates a Reader for messages that share one source of payloads.
    /// @param reads the messages to read and the channel of each field.
    /// @param ic the ICD that holds the messages of reads.
    /// @param channels the channels of the bindings of reads.
    /// @returns CONFIGURATION_ERROR when ic has no message that a read names, two
    /// messages have different formats, a message binds no fields, channels has no
    /// channel that a binding names, the index field does not write to an index channel
    /// or takes a default, another field writes to an index channel, two bindings or
    /// two messages write to one channel (an index channel included), different
    /// multiplexor values select two fields on one index channel, or the decoder
    /// rejects the messages.
    static std::pair<Reader, x::errors::Error> create(
        const std::vector<synnax::icd::ReadMessage> &reads,
        const synnax::icd::ICD &ic,
        const std::map<synnax::channel::Key, synnax::channel::Channel> &channels
    );

    /// @brief returns each channel that the reader writes, index channels last.
    [[nodiscard]] std::vector<synnax::channel::Key> channels() const;

    /// @brief reads payload as the message it belongs to, and adds one sample for each
    /// channel of that message to fr. The index channel of the index field takes its
    /// value, and every other index channel takes received. An index channel and the
    /// channels it indexes take no sample when the multiplexors of payload select none
    /// of their fields, or when the decoder cannot decode one of their fields. A
    /// virtual channel whose field the decoder cannot decode takes no sample. failures
    /// lists the fields that the decoder could not decode.
    /// @param cap the capacity of each series that read adds to fr. A series that fr
    /// already holds for a channel takes the sample instead, and must have room for it.
    /// @param fill when set, the sample of a field on a float channel that the decoder
    /// cannot decode, which keeps its index group.
    /// @returns the position in reads of the message, or nothing, adding nothing, when
    /// payload belongs to no message.
    /// @returns VALIDATION, adding nothing, when the decoder cannot decode payload.
    std::pair<std::optional<std::size_t>, x::errors::Error> read(
        std::span<const std::uint8_t> payload,
        x::telem::TimeStamp received,
        x::telem::Frame &fr,
        std::size_t cap = 1,
        std::optional<double> fill = std::nullopt
    );

    /// @brief returns the fields of the last read that its decoder could not decode, by
    /// position in the bindings of the message that it returned. Empty after a read
    /// that returned no message or an error, and for a format other than JSON.
    [[nodiscard]] std::span<const codec::Failure> failures() const;
};
}
