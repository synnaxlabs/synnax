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
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/channel/types.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/series.h"

#include "driver/codec/text.h"

namespace driver::icd {
/// @brief Writer encodes the commands of the channels of one ICD message into payloads
/// of that message. Payload i of a frame holds command i of each bound channel, and the
/// last good command of a channel with fewer. A field with no command yet sends its
/// default, and the writer encodes nothing until each field with no default has a
/// command. A command that its field refuses is dropped, and the field keeps its last
/// good command. It is not safe for concurrent use.
class Writer {
    /// @brief Binding is the command channel of a field and the field's limits.
    struct Binding {
        synnax::channel::Key channel = 0;
        std::string field;
        std::optional<double> min;
        std::optional<double> max;
    };

    std::string name;
    codec::TextEncoder encoder;
    std::vector<Binding> bindings;
    /// @brief last holds the last good command of each binding, or no sample before its
    /// first.
    std::vector<x::telem::Series> last;
    /// @brief rows is scratch space that holds the commands of each binding in a frame.
    std::vector<const x::telem::Series *> rows;
    /// @brief payload is scratch space for the payload to send.
    std::vector<std::uint8_t> payload;

    /// @returns VALIDATION when binding cannot send v.
    [[nodiscard]] x::errors::Error
    check(std::size_t binding, const x::telem::SampleValue &v);
    /// @brief returns true when each field with no default has a command.
    [[nodiscard]] bool ready() const;

public:
    /// @brief Send sends one payload.
    using Send = std::function<x::errors::Error(std::span<const std::uint8_t>)>;

    /// @brief Result is the outcome of a write.
    struct Result {
        /// @brief error is the error of send, after which the write sends no more
        /// payloads.
        x::errors::Error error;
        /// @brief refused is the VALIDATION error of the first command that the write
        /// dropped.
        x::errors::Error refused;
    };

    /// @brief creates a Writer of the message of ic that wm sends. A value binding
    /// sends its value as its field's default.
    /// @param channels the channels of the channel bindings of wm.
    /// @returns CONFIGURATION_ERROR when ic has no message that wm names, the message
    /// is not text, wm binds no channels, channels has no channel that a binding names,
    /// a value binding names no field, a field has two bindings, a value is not of its
    /// field's kind, a default decodes to a value outside the min and max of its field,
    /// or the encoder refuses the message.
    static std::pair<Writer, x::errors::Error> create(
        const synnax::icd::WriteMessage &wm,
        const synnax::icd::ICD &ic,
        const std::map<synnax::channel::Key, synnax::channel::Channel> &channels
    );

    /// @brief encodes a payload for each row of commands in fr, and passes it to send.
    /// Row i holds command i of each bound channel that has one. A command outside the
    /// min and max of its field, or one that its field cannot hold, is dropped. A row
    /// with no good command sends nothing.
    [[nodiscard]] Result write(const x::telem::Frame &fr, const Send &send);
};
}
