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
#include <functional>
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
#include "x/cpp/telem/series.h"

#include "driver/codec/binary.h"
#include "driver/codec/json.h"
#include "driver/codec/text.h"

namespace driver::icd {
/// @brief Writer encodes the commands of the channels of one ICD message into payloads
/// of that message. Payload i of a frame holds command i of each bound channel, and the
/// last good command of a channel with fewer. A field with no command yet sends its
/// default, and the writer encodes nothing until each field that the multiplexors
/// select has a command or a default. A command that its field refuses is dropped, and
/// the field keeps its last good command. It is not safe for concurrent use.
class Writer {
    std::variant<codec::BinaryEncoder, codec::TextEncoder, codec::JSONEncoder> encoder;
    /// @brief channels holds the command channel of each binding.
    std::vector<synnax::channel::Key> channels;
    /// @brief last holds the last good command of each binding, or no sample before its
    /// first.
    std::vector<x::telem::Series> last;
    /// @brief commands is scratch space that holds the series of each binding in a
    /// frame, in frame order.
    std::vector<std::vector<const x::telem::Series *>> commands;
    /// @brief payload is scratch space for the payload to send.
    std::vector<std::uint8_t> payload;

    /// @brief Command is the sample at index of series.
    struct Command {
        const x::telem::Series *series;
        std::size_t index;
    };

    /// @brief returns command i of binding in the frame of the current write, or
    /// nothing when binding has fewer.
    [[nodiscard]] std::optional<Command>
    command(std::size_t binding, std::size_t i) const;
    /// @brief points commands at the series of each binding in fr.
    /// @returns the number of rows of commands in fr.
    std::size_t gather(const x::telem::Frame &fr);
    /// @brief keeps the good commands of row, and sets refused to the error of the
    /// first command that it drops when refused is nil.
    /// @returns true when row holds a good command.
    bool apply(std::size_t row, x::errors::Error &refused);

public:
    /// @brief Send sends one payload.
    using Send = std::function<x::errors::Error(std::span<const std::uint8_t>)>;

    /// @brief Result is the outcome of a write.
    struct Result {
        /// @brief error is the error of send or of an encode, after which the write
        /// sends no more payloads.
        x::errors::Error error;
        /// @brief refused is the VALIDATION error of the first command that the write
        /// dropped.
        x::errors::Error refused;
    };

    /// @brief creates a Writer of the message of ic that wm sends. A value binding
    /// sends its value as its field's default.
    /// @param channels the channels of the channel bindings of wm.
    /// @returns CONFIGURATION_ERROR when ic has no message that wm names, wm binds no
    /// channels, channels has no channel that a binding names, a value binding names no
    /// field, a field has two bindings, a value is not of its field's kind, a value
    /// decodes outside the min and max of its field, or the encoder refuses the
    /// message.
    static std::pair<Writer, x::errors::Error> create(
        const synnax::icd::WriteMessage &wm,
        const synnax::icd::ICD &ic,
        const std::map<synnax::channel::Key, synnax::channel::Channel> &channels
    );

    /// @brief encodes a payload for each row of commands in fr, and passes it to send.
    /// The commands of a channel are the samples of its series in fr, in frame order,
    /// and row i holds command i of each bound channel that has one. A command that its
    /// field cannot hold, or a command outside the min and max of its field, is
    /// dropped. A row with no good command sends nothing. After an error, the write
    /// sends nothing more, but each field still keeps its last good command in fr.
    [[nodiscard]] Result write(const x::telem::Frame &fr, const Send &send);

    /// @brief keeps the last good command of each field in fr, as write does, and sends
    /// nothing.
    /// @returns the VALIDATION error of the first command that it dropped.
    [[nodiscard]] x::errors::Error keep(const x::telem::Frame &fr);

    /// @brief encodes one payload of the last good command of each field, and passes it
    /// to send. It sends nothing until each field that the multiplexors select has a
    /// command or a default.
    /// @returns the error of the encode or of send.
    [[nodiscard]] x::errors::Error encode(const Send &send);
};
}
