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

#include "driver/arinc429/backend.h"
#include "driver/bus/config.h"
#include "driver/bus/write.h"

namespace driver::arinc429 {
/// @brief sends each encoded message as one word on a transmit channel, with its
/// label, SDI, and odd parity set. It reopens the channel after a failed send.
class Output final : public bus::Output {
public:
    /// @param cfg the resolved write config. Each message has an ARINC 429
    /// identifier.
    /// @param backend opens the transmit channel.
    /// @param props the device's properties.
    Output(
        const bus::WriteConfig &cfg,
        std::shared_ptr<Backend> backend,
        synnax::arinc429::Properties props
    );

    /// @returns the backend's error when the channel cannot open.
    x::errors::Error start() override;

    void stop() override;

    x::errors::Error
    send(std::size_t message, std::span<const std::uint8_t> payload) override;

private:
    std::vector<synnax::library::Arinc429Identifier> labels;
    std::shared_ptr<Backend> backend;
    synnax::arinc429::Properties props;
    std::unique_ptr<Channel> channel;
};
}
