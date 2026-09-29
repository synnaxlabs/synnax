// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <functional>
#include <memory>
#include <mutex>
#include <string>
#include <utility>

#include "client/cpp/can/types.gen.h"
#include "x/cpp/errors/errors.h"

#include "driver/bus/registry.h"
#include "driver/can/can.h"

namespace driver::can {
/// @brief opens a CAN bus.
using Opener = std::function<std::pair<std::unique_ptr<Bus>, x::errors::Error>()>;

/// @brief the CAN bus of one device, shared by every task on the device. Its bus
/// opens on first use and reopens after a fault. Safe for concurrent use: a read task
/// may receive while a write task sends.
class Link {
public:
    explicit Link(Opener open): open(std::move(open)) {}

    /// @returns the open bus, opened first when it is closed. The error of the Opener
    /// when the bus cannot open.
    std::pair<std::shared_ptr<Bus>, x::errors::Error> bus();

    /// @brief closes bus when it is still the open bus, so the next call to bus
    /// reopens it. The bus closes once no caller still uses it.
    void close(const std::shared_ptr<Bus> &bus);

private:
    Opener open;
    std::mutex mu;
    std::shared_ptr<Bus> current;
};

/// @brief the links of the CAN integration, one per device.
using Links = bus::Registry<Link>;

/// @brief acquires the shared link to a task's device.
using Acquire = bus::Acquirer<Link>;

/// @returns an Acquire of the link to the device with the given key and properties,
/// opened through backends.
Acquire acquirer(
    std::shared_ptr<Links> links,
    std::string key,
    std::shared_ptr<const Backends> backends,
    synnax::can::Properties props
);
}
