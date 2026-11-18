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
/// @brief Binding writes the value of one message field to a channel.
struct Binding {
    /// @brief field is the key of the message field to read.
    synnax::icd::FieldKey field;
    /// @brief channel is the channel the value of the field is written to.
    synnax::channel::Channel channel;
};

/// @brief Bound is an ICD message and the channels its fields write to.
struct Bound {
    /// @brief message is the ICD message. Its payload must be JSON.
    synnax::icd::Message message;
    /// @brief index is the field whose channel indexes the others. Absent when the
    /// samples take the time the response arrived.
    std::optional<synnax::icd::FieldKey> index;
    /// @brief bindings name the fields to read and their channels.
    std::vector<Binding> bindings;
};

/// @brief Reader reads the response bodies of one endpoint, each one of several ICD
/// messages, into the channels of that message. It is not safe for concurrent use.
class Reader {
    struct Message {
        /// @brief channels holds the channel of each binding, in binding order.
        std::vector<synnax::channel::Key> channels;
        /// @brief index is the index channel of every binding, or 0 for virtual
        /// channels.
        synnax::channel::Key index = 0;
        /// @brief index_field is the position in the bindings of the field that gives
        /// the timestamp, or nothing when the samples take the arrival time.
        std::optional<std::size_t> index_field;
    };

    std::vector<Message> messages;
    codec::JSON decoder;

    /// @brief sets the index channel and the index field of m from the bindings of b,
    /// and returns the codec binding of each.
    static std::pair<std::vector<codec::Binding>, x::errors::Error>
    bind(const Bound &b, Message &m);

public:
    /// @brief creates a Reader for messages.
    /// @returns CONFIGURATION_ERROR when messages is empty, a message is not JSON or
    /// binds no fields, the bound channels of a message do not share one index channel,
    /// the index field does not write to an index channel, another field does, two
    /// bindings or two messages write to one channel (an index channel included), or
    /// the decoder rejects the messages.
    static std::pair<Reader, x::errors::Error>
    create(const std::vector<Bound> &messages);

    /// @brief reads body as the message whose match rule it meets, and adds one sample
    /// for each channel of that message to fr. The index channel takes the value of the
    /// index field, or received when the message has none.
    /// @returns VALIDATION, adding nothing, when body is not JSON, a value does not
    /// convert, or body meets no message.
    x::errors::Error
    read(std::string_view body, x::telem::TimeStamp received, x::telem::Frame &fr);
};
}
