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
#include <vector>

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/telem/frame.h"

#include "driver/bus/config.h"
#include "driver/bus/decoder.h"
#include "driver/bus/queue.h"
#include "driver/bus/schedule.h"
#include "driver/common/read_task.h"
#include "driver/mil1553/mil1553.h"

namespace driver::mil1553 {
/// @brief the most transfers one read takes from a monitor or remote terminal.
constexpr std::size_t READ_BATCH = 256;

/// @brief reads messages from a channel in its role. A bus controller commands each
/// transmit message on its period. A monitor or remote terminal decodes each
/// transfer it sees whose address and subaddress match a message.
class Source final : public common::Source {
public:
    /// @param cfg the resolved read config, checked against the role in props.
    /// @param props the device's properties.
    /// @param acquire acquires the device's channel.
    Source(bus::ReadConfig cfg, synnax::mil1553::Properties props, Acquire acquire);

    [[nodiscard]] synnax::framer::WriterConfig writer_config() const override;

    [[nodiscard]] std::vector<synnax::channel::Channel> channels() const override;

    /// @brief acquires the device's link and opens its channel.
    /// @returns transport::CONFIG_ERROR when another task has the device open with
    /// other properties, or the backend's error when the channel cannot open.
    x::errors::Error start() override;

    /// @brief releases the link, which closes when no other task uses it.
    x::errors::Error stop() override;

    /// @brief makes the transfers that are due, or reads the transfers that arrive
    /// within bus::READ_TIMEOUT, and decodes them into fr.
    /// @returns the backend's error, after which the link reopens the channel.
    /// Transfers a terminal did not complete come back as warnings.
    common::ReadResult read(x::breaker::Breaker &breaker, x::telem::Frame &fr) override;

private:
    bus::ReadConfig cfg;
    bus::Decoder decoder;
    synnax::mil1553::Properties props;
    Acquire acquire;
    std::shared_ptr<Link> link;
    bus::Schedule schedule;
    std::vector<std::size_t> due;
    std::vector<Transfer> buf;
    std::vector<std::uint8_t> payload;

    x::errors::Error poll(x::breaker::Breaker &breaker);
    x::errors::Error listen();
    void add(std::size_t message, const Transfer &transfer);
};
}
