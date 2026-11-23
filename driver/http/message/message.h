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
#include <map>
#include <optional>
#include <string_view>
#include <utility>
#include <vector>

#include "client/cpp/channel/types.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/telem.h"

#include "driver/codec/json.h"

namespace driver::http::message {
/// @brief Reader reads the response bodies of one endpoint, each one of several ICD
/// messages, into the channels of that message. It is not safe for concurrent use.
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
        /// @brief indexes holds the index channel of each bound channel once, in
        /// binding order. Virtual channels add none.
        std::vector<Index> indexes;
        /// @brief index_field is the position in the bindings of the index field.
        std::optional<std::size_t> index_field;
    };

    std::vector<Message> messages;
    codec::JSON decoder;

    /// @brief sets the channels, the index channels, and the index field of m from the
    /// bindings of rm, which reads payload, and returns the codec binding of each.
    static std::pair<std::vector<codec::Binding>, x::errors::Error> bind(
        const synnax::icd::ReadMessage &rm,
        const synnax::icd::JSONPayload &payload,
        const std::map<synnax::channel::Key, synnax::channel::Channel> &channels,
        Message &m
    );

public:
    /// @brief creates a Reader for the messages that one endpoint reads.
    /// @param reads the messages to read and the channel of each field.
    /// @param ic the ICD that holds the messages of reads.
    /// @param channels the channels of the bindings of reads.
    /// @returns CONFIGURATION_ERROR when reads is empty, ic has no message that a read
    /// names, a message is not JSON or binds no fields, channels has no channel that a
    /// binding names, the index field does not write to an index channel or takes a
    /// default, another field writes to an index channel, two bindings or two messages
    /// write to one channel (an index channel included), or the decoder rejects the
    /// messages.
    static std::pair<Reader, x::errors::Error> create(
        const std::vector<synnax::icd::ReadMessage> &reads,
        const synnax::icd::ICD &ic,
        const std::map<synnax::channel::Key, synnax::channel::Channel> &channels
    );

    /// @brief returns each channel that the reader writes, index channels last.
    [[nodiscard]] std::vector<synnax::channel::Key> keys() const;

    /// @brief reads body as the message whose match rule it meets, and adds one sample
    /// for each channel of that message to fr. The index channel of the index field
    /// takes its value, and every other index channel takes received.
    /// @returns VALIDATION, adding nothing, when body is not JSON, a value does not
    /// convert, or body meets no message.
    x::errors::Error
    read(std::string_view body, x::telem::TimeStamp received, x::telem::Frame &fr);
};
}
