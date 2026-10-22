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
#include <vector>

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/telem.h"

#include "driver/bus/config.h"
#include "driver/bus/decoder.h"
#include "driver/can/link.h"
#include "driver/common/read_task.h"

namespace driver::can {
/// @brief the most frames one read takes from the bus.
constexpr std::size_t MAX_BATCH = 1024;

/// @brief receives frames from a device's CAN bus, decodes each frame whose
/// identifier matches a message the task reads, and writes the values to the
/// messages' channels. Each raw sample is the frame's identifier as a 4-byte
/// little-endian integer, with bit 31 set for an extended identifier, followed by the
/// payload. Frames stamped with the adapter's clock move to the host clock through an
/// Aligner that starts over each time the link reopens. While the bus cannot open, the
/// source waits for it with a warning and never fails.
class Source final : public common::Source {
public:
    /// @param cfg the resolved read config.
    /// @param acquire acquires the device's link on start.
    Source(bus::ReadConfig cfg, Acquire acquire);

    [[nodiscard]] synnax::framer::WriterConfig writer_config() const override;

    [[nodiscard]] std::vector<synnax::channel::Channel> channels() const override;

    /// @brief acquires the link and opens the bus.
    /// @returns CONFIG_ERROR when the backend cannot open the channel with the device's
    /// properties, CRITICAL_HARDWARE_ERROR on an adapter fault, and
    /// transport::CONFIG_ERROR when another task has the device open with other
    /// properties. An unreachable channel is not an error here: read waits for it.
    x::errors::Error start() override;

    /// @brief releases the link. The bus closes when no other task uses it.
    x::errors::Error stop() override;

    /// @brief receives frames for at most bus::READ_TIMEOUT, or until MAX_BATCH
    /// frames arrive, and decodes them into fr.
    /// @returns CRITICAL_HARDWARE_ERROR or CONFIG_ERROR when the bus fails in a way
    /// that reopening does not fix. An unreachable bus, short payloads, and adapter bus
    /// errors come back as warnings.
    common::ReadResult read(x::breaker::Breaker &breaker, x::telem::Frame &fr) override;

private:
    bus::ReadConfig cfg;
    bus::Decoder decoder;
    Acquire acquire;
    std::shared_ptr<Link> link;
    /// @brief the link's open count when the aligner started.
    std::uint64_t opens = 0;
    /// @brief moves adapter times of the open bus onto the host clock.
    Aligner aligner;
    std::vector<std::uint8_t> raw;

    void handle(const Frame &frame);
    x::telem::TimeStamp stamp(const Frame &frame);
};
}
