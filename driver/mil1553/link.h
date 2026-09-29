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
#include <mutex>
#include <span>
#include <string>
#include <utility>

#include "client/cpp/mil1553/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "driver/bus/queue.h"
#include "driver/bus/registry.h"
#include "driver/mil1553/backend.h"

namespace driver::mil1553 {
/// @brief the channel of one device, shared by the device's read and write tasks. The
/// channel opens on first use and reopens after any error of the card. Safe for
/// concurrent use: read waits without blocking other calls, and every other call runs
/// one at a time.
class Link {
public:
    /// @param backend opens the channel.
    /// @param props the device's properties.
    Link(std::shared_ptr<Backend> backend, synnax::mil1553::Properties props):
        backend(std::move(backend)), props(std::move(props)) {}

    /// @brief opens the channel when it is closed.
    /// @returns the backend's error when the channel cannot open.
    x::errors::Error open();

    /// @brief makes a transfer as the bus controller. See Channel::transact.
    std::pair<Transfer, x::errors::Error> transact(
        const codec::mil1553::Command &command,
        std::span<const std::uint16_t> words
    );

    /// @brief reads the transfers seen since the last read. See Channel::read.
    std::pair<bus::Batch, x::errors::Error>
    read(std::span<Transfer> out, x::telem::TimeSpan timeout);

    /// @brief sets the words a terminal answers with. See Channel::respond.
    x::errors::Error respond(
        std::uint8_t rt,
        std::uint8_t subaddress,
        std::span<const std::uint16_t> words
    );

private:
    std::shared_ptr<Backend> backend;
    synnax::mil1553::Properties props;
    std::mutex mu;
    std::shared_ptr<Channel> channel;

    /// @returns the open channel. The caller holds mu.
    std::pair<std::shared_ptr<Channel>, x::errors::Error> opened();
};

/// @brief the links of the MIL-STD-1553 integration, one per device.
using Links = bus::Registry<Link>;

/// @brief acquires the shared link to a task's device.
using Acquire = bus::Acquirer<Link>;

/// @returns an Acquire of the link to the device with the given key and properties,
/// opened through backend.
Acquire acquirer(
    std::shared_ptr<Links> links,
    std::string key,
    std::shared_ptr<Backend> backend,
    synnax::mil1553::Properties props
);
}
