// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <chrono>
#include <cstdint>
#include <map>
#include <memory>
#include <optional>
#include <span>
#include <utility>
#include <vector>

#include "client/cpp/channel/types.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/telem.h"

#include "driver/codec/framing/framing.h"
#include "driver/common/common.h"
#include "driver/icd/writer.h"
#include "driver/transport/connection.h"
#include "driver/transport/transport.h"

namespace driver::icd {
/// @brief Sender sends the messages of the targets of a task on one transport. A target
/// with no period sends a message for each row of commands, and a target with a period
/// sends the last good command of each field once per period. Each message goes out in
/// the framing of its ICD, in one write to the transport. It is not safe for concurrent
/// use.
class Sender {
    /// @brief Delivery sends the message of one target.
    struct Delivery {
        Writer writer;
        /// @brief framer frames each payload, and is null when the ICD has no framing.
        std::unique_ptr<codec::framing::Framer> framer;
        std::optional<x::telem::TimeSpan> period;
        /// @brief next is the time at which a target with a period sends next.
        std::chrono::steady_clock::time_point next;
    };

    std::vector<Delivery> deliveries;
    /// @brief framed is scratch space for the transport bytes of a message.
    std::vector<std::uint8_t> framed;
    transport::Clock now;

    /// @brief returns payload in the framing of d, valid until the next call.
    std::pair<std::span<const std::uint8_t>, x::errors::Error>
    frame(const Delivery &d, std::span<const std::uint8_t> payload);

public:
    /// @brief creates a Sender.
    /// @param targets the targets to send, in task order.
    /// @param icds the ICDs of targets.
    /// @param channels the channels of the channel bindings of targets.
    /// @param datagram true when each transport write sends one datagram. A datagram
    /// with no framing holds one message.
    /// @param now is the steady clock that schedules the targets with a period.
    /// @returns CONFIGURATION_ERROR when icds has no ICD that a target names, a stream
    /// ICD has no framing, or the Writer or framer of a target fails to create.
    static std::pair<Sender, x::errors::Error> create(
        const std::vector<synnax::icd::WriteTarget> &targets,
        const std::vector<synnax::icd::ICD> &icds,
        const std::map<synnax::channel::Key, synnax::channel::Channel> &channels,
        bool datagram,
        transport::Clock now = std::chrono::steady_clock::now
    );

    /// @brief sends on t the messages of fr for each target with no period, as
    /// Writer::write encodes them, and keeps the last good command of each field of a
    /// target with a period.
    /// @returns a warning about the first command that a field refused, and the error
    /// of t, of an encode, or of the framing, after which the write sends nothing more.
    std::pair<common::Warning, x::errors::Error>
    write(transport::Transport &t, const x::telem::Frame &fr);

    /// @brief sends on t the message of each target with a period that is due. A target
    /// that fails waits for its next period, as one that sends does. After an error of
    /// t, the send tries no more targets. After a stall, a target sends one message,
    /// not one for each period it missed.
    /// @returns the wait until the next message is due, or TimeSpan::max() when no
    /// target has a period, and the error of t, or else the first error of an encode or
    /// of the framing.
    std::pair<x::telem::TimeSpan, x::errors::Error> send(transport::Transport &t);
};
}
