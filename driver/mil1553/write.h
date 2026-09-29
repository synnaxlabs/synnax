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
#include <span>
#include <vector>

#include "driver/bus/config.h"
#include "driver/bus/write.h"
#include "driver/mil1553/mil1553.h"

namespace driver::mil1553 {
/// @brief sends each encoded message on a channel in its role. A bus controller
/// commands a receive transfer with the message's words. A remote terminal sets the
/// words its terminal answers the message's transmit command with.
class Transmitter final : public bus::Transmitter {
public:
    /// @param cfg the resolved write config, checked against the role in props.
    /// @param props the device's properties.
    /// @param acquire acquires the device's link.
    Transmitter(
        const bus::WriteConfig &cfg,
        synnax::mil1553::Properties props,
        Acquire acquire
    );

    /// @brief acquires the device's link and opens its channel.
    /// @returns transport::CONFIG_ERROR when another task has the device open with
    /// other properties, or the backend's error when the channel cannot open.
    x::errors::Error acquire() override;

    /// @brief releases the link, which closes when no other task uses it.
    void release() override;

    /// @returns the backend's error, after which the link reopens the channel, or an
    /// error naming the terminal when it does not complete the transfer.
    x::errors::Error
    send(std::size_t message, std::span<const std::uint8_t> payload) override;

private:
    std::vector<codec::mil1553::Command> commands;
    synnax::mil1553::Properties props;
    Acquire acquirer;
    std::shared_ptr<Link> link;
};
}
