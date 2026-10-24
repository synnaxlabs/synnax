// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <memory>
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/bus/types.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "client/cpp/synnax.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/telem.h"

#include "driver/codec/framing/framing.h"
#include "driver/codec/matcher.h"
#include "driver/codec/plan.h"

namespace driver::bus {
/// @brief the longest one streamed read holds a device's connection, so that writes to
/// the device wait at most this long and a stopping task exits promptly.
const auto READ_TIMEOUT = 50 * x::telem::MILLISECOND;
/// @brief the longest one transport write blocks.
const auto WRITE_TIMEOUT = 1 * x::telem::SECOND;

/// @brief what carries a task's frames, which decides the identifiers its messages can
/// have.
enum class Medium : std::uint8_t {
    /// @brief a byte stream or datagrams. Messages match by field or prefix.
    BYTES,
    /// @brief CAN frames. Messages are binary, match by CAN identifier, and cannot be
    /// polled.
    CAN,
};

/// @brief decodes the escaped bytes of a binary query. Characters stand for themselves,
/// and \xHH, \n, \r, \t, \0, and \\ stand for one byte each.
/// @returns x::errors::VALIDATION for an unknown or truncated escape.
std::pair<std::vector<std::uint8_t>, x::errors::Error>
unescape(const std::string &escaped);

/// @brief how a read task splits a byte stream into frames.
struct Stream {
    /// @brief splits the stream into frames.
    ::synnax::bus::Framing framing;
    /// @brief the pause after which fixed framing drops a partial frame. It depends on
    /// whether the transport can lose bytes.
    x::telem::TimeSpan gap = x::telem::TimeSpan::max();
};

/// @brief the fields of a read message that share an index channel. The same
/// multiplexor values select every field of a group, so they arrive together.
struct Group {
    /// @brief the index channel stamped with each arrival, or zero for a field whose
    /// channel is virtual.
    synnax::channel::Key index = 0;
    /// @brief the plan slots of the group's fields.
    std::vector<std::size_t> slots;
};

/// @brief an ICD message a read task decodes.
struct ReadMessage {
    /// @brief the message as its ICD defines it.
    synnax::icd::Message definition;
    /// @brief decodes the message. Its slots follow the configured fields.
    codec::Plan plan;
    /// @brief the fields of the message grouped by index channel.
    std::vector<Group> groups;
    /// @brief the channel each plan slot is written to.
    std::vector<synnax::channel::Channel> channels;
    /// @brief how often the message arrives: its poll period when it has a query, its
    /// ICD period otherwise, or zero when unknown.
    x::telem::TimeSpan period;
    /// @brief the wire bytes of the message's query. Empty when the device sends the
    /// message unprompted.
    std::vector<std::uint8_t> query;
    /// @brief matches the reply to the message's query.
    codec::Matcher reply;
};

/// @brief a bus read task config resolved against its ICD and channels.
struct ReadConfig {
    /// @brief the name of the device, for status messages.
    std::string device;
    /// @brief the enabled messages.
    std::vector<ReadMessage> messages;
    /// @brief matches frames to the messages with no query. Its results index streamed.
    codec::Matcher matcher;
    /// @brief the index into messages of each message the matcher knows.
    std::vector<std::size_t> streamed;
    /// @brief the index into messages of each message with a query, in poll order.
    std::vector<std::size_t> polled;
    /// @brief the virtual bytes channel every received frame is written to. Zero when
    /// the task does not stream raw frames.
    synnax::channel::Key raw = 0;
    /// @brief true when the task streams data without saving it.
    bool data_saving_disabled = false;
    /// @brief how often the task sends each query.
    x::telem::Rate poll_rate;
    /// @brief how long the task waits for a reply before it counts a miss.
    x::telem::TimeSpan poll_timeout;
    /// @brief splits the byte stream into frames. Null when each chunk is one frame, as
    /// with UDP datagrams.
    std::unique_ptr<codec::framing::Framer> framer;
    /// @brief the pause after which fixed framing drops a partial frame.
    x::telem::TimeSpan gap = x::telem::TimeSpan::max();

    /// @brief resolves cfg against its ICD and the channels it names, binding
    /// validation errors to their fields on parser.
    /// @param stream how the byte stream splits into frames. Absent for datagrams and
    /// CAN.
    /// @returns x::errors::INTERNAL when a message layout passed the Core's checks but
    /// the Driver cannot compile it, which is a bug in one of the two.
    static std::pair<ReadConfig, x::errors::Error> resolve(
        x::json::Parser &parser,
        const ::synnax::bus::ReadConfig &cfg,
        const ::synnax::bus::PollConfig &poll,
        const std::optional<Stream> &stream,
        const synnax::icd::ICD &ic,
        const std::vector<synnax::channel::Channel> &channels,
        Medium medium = Medium::BYTES
    );

    /// @brief retrieves the device, ICD, and channels cfg names from the Core, then
    /// resolves cfg as resolve does.
    /// @returns x::errors::VALIDATION with field errors when cfg is invalid,
    /// x::errors::INTERNAL as resolve does, or the Core's error when a retrieval fails.
    static std::pair<ReadConfig, x::errors::Error> parse(
        const synnax::Synnax &client,
        x::json::Parser &parser,
        const ::synnax::bus::ReadConfig &cfg,
        const ::synnax::bus::PollConfig &poll,
        const std::optional<Stream> &stream,
        Medium medium = Medium::BYTES
    );
};

/// @brief a command channel bound to a field of a write message.
struct Binding {
    /// @brief the command channel.
    synnax::channel::Key channel = 0;
    /// @brief the plan slot of the field.
    std::size_t slot = 0;
    /// @brief the name of the field.
    std::string field;
};

/// @brief an ICD message a write task encodes.
struct WriteMessage {
    /// @brief the message as its ICD defines it.
    synnax::icd::Message definition;
    /// @brief encodes the message. Its slots are every field of the message, in order.
    codec::Plan plan;
    /// @brief binds command channels to slots.
    std::vector<Binding> bindings;
    /// @brief the bytes that start every encoded payload: the prefix of a text message.
    std::string prefix;
    /// @brief values that every payload starts from: zero for unbound fields, and the
    /// identifier value of a field identifier.
    codec::Values initial;
};

/// @brief a bus write task config resolved against its ICD and channels.
struct WriteConfig {
    /// @brief the name of the device, for status messages.
    std::string device;
    /// @brief the enabled messages.
    std::vector<WriteMessage> messages;
    /// @brief the distinct command channels, in config order.
    std::vector<synnax::channel::Key> commands;
    /// @brief frames each encoded payload. Null when each payload is one datagram.
    std::unique_ptr<codec::framing::Framer> framer;

    /// @brief resolves cfg as ReadConfig::resolve does. On CAN, every message needs a
    /// CAN identifier to send with.
    static std::pair<WriteConfig, x::errors::Error> resolve(
        x::json::Parser &parser,
        const ::synnax::bus::WriteConfig &cfg,
        const std::optional<::synnax::bus::Framing> &framing,
        const synnax::icd::ICD &ic,
        const std::vector<synnax::channel::Channel> &channels,
        Medium medium = Medium::BYTES
    );

    /// @brief resolves cfg against the Core as ReadConfig::parse does.
    static std::pair<WriteConfig, x::errors::Error> parse(
        const synnax::Synnax &client,
        x::json::Parser &parser,
        const ::synnax::bus::WriteConfig &cfg,
        const std::optional<::synnax::bus::Framing> &framing,
        Medium medium = Medium::BYTES
    );
};
}
