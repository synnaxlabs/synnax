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

#include "driver/arinc429/backend.h"
#include "driver/bus/config.h"
#include "driver/bus/decoder.h"
#include "driver/common/read_task.h"

namespace driver::arinc429 {
/// @brief the most words one read takes from the receiver.
constexpr std::size_t READ_BATCH = 512;

/// @brief reads the words that arrive on a receive channel and decodes each word whose
/// label and SDI match a message into the message's channels.
class Source final : public common::Source {
public:
    /// @param cfg the resolved read config.
    /// @param backend opens the receive channel.
    /// @param props the device's properties.
    Source(
        bus::ReadConfig cfg,
        std::shared_ptr<Backend> backend,
        synnax::arinc429::Properties props
    );

    [[nodiscard]] synnax::framer::WriterConfig writer_config() const override;

    [[nodiscard]] std::vector<synnax::channel::Channel> channels() const override;

    /// @brief opens the receive channel.
    /// @returns the backend's error when the channel cannot open.
    x::errors::Error start() override;

    /// @brief closes the receive channel.
    x::errors::Error stop() override;

    /// @brief reads the words that arrive within bus::READ_TIMEOUT and decodes them
    /// into fr.
    /// @returns the backend's error, after which the next read reopens the channel.
    /// Words that fail the parity check or were dropped come back as warnings.
    common::ReadResult read(x::breaker::Breaker &breaker, x::telem::Frame &fr) override;

private:
    bus::ReadConfig cfg;
    bus::Decoder decoder;
    std::shared_ptr<Backend> backend;
    synnax::arinc429::Properties props;
    std::unique_ptr<Channel> channel;
    std::vector<Received> buf;
};
}
