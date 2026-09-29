// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <span>
#include <vector>

#include "x/cpp/json/json.h"

#include "driver/bus/config.h"
#include "driver/bus/write.h"
#include "driver/can/link.h"

namespace driver::can {
/// @returns the frame of each message in the write config, with the message's
/// identifier, format, and length and a zero payload. Binds a field error to parser for
/// each frame that validate rejects on the device's bus.
/// @param fd true when the device's bus runs CAN FD.
[[nodiscard]] std::vector<Frame>
frames(const x::json::Parser &parser, const bus::WriteConfig &cfg, bool fd);

/// @brief sends each message as a frame with the message's CAN identifier on the
/// device's shared bus. A critical hardware error closes the bus, so the next send
/// reopens it.
class Transmitter final : public bus::Transmitter {
public:
    /// @param frames the frame of each message, from frames.
    /// @param acquire acquires the device's link on start.
    Transmitter(std::vector<Frame> frames, Acquire acquire);

    x::errors::Error acquire() override;
    void release() override;

    /// @returns FRAME_ERROR when the payload does not fit a frame on the bus, and the
    /// bus's error when it cannot queue the frame.
    x::errors::Error
    send(std::size_t message, std::span<const std::uint8_t> payload) override;

private:
    /// @brief the identifier and format of each message's frames.
    std::vector<Frame> frames;
    Acquire acquirer;
    std::shared_ptr<Link> link;
};
}
