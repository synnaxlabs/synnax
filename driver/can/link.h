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
#include <functional>
#include <memory>
#include <mutex>
#include <string>
#include <utility>

#include "client/cpp/can/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/mono_clock.h"
#include "x/cpp/telem/telem.h"

#include "driver/bus/registry.h"
#include "driver/can/can.h"

namespace driver::can {
/// @brief opens a CAN bus.
using Opener = std::function<std::pair<std::unique_ptr<Bus>, x::errors::Error>()>;

/// @brief the wait after the first failed open of a link's bus. Each failed open
/// doubles the wait, up to MAX_REOPEN_WAIT.
inline const x::telem::TimeSpan MIN_REOPEN_WAIT = 250 * x::telem::MILLISECOND;
/// @brief the longest wait between two opens of a link's bus.
inline const x::telem::TimeSpan MAX_REOPEN_WAIT = 5 * x::telem::SECOND;

/// @returns the time of a steady clock, which never steps when the wall clock does.
x::telem::TimeStamp steady_now();

/// @brief a bus that a Link opened.
struct Opened {
    /// @brief the open bus. Hold it only for the length of a call on it.
    std::shared_ptr<Bus> bus;
    /// @brief the number of times the link has opened a bus, counting this one. A
    /// change means the link reopened.
    std::uint64_t opens = 0;
};

/// @brief the CAN bus of one device, shared by every task on the device. It opens the
/// bus on first use, drops it on an unreachable or critical fault, and opens it again
/// with one wait between attempts that every task on the device shares. Safe for
/// concurrent use: a read task may receive while a write task sends.
class Link {
public:
    /// @param channel the name of the device's channel, for status messages.
    /// @param open opens the bus.
    /// @param now the clock that spaces open attempts.
    Link(std::string channel, Opener open, x::telem::NowFunc now = steady_now);

    /// @returns the open bus, opening it first when the link holds none and the wait
    /// after the last failed open has passed. While the bus cannot open, the error of
    /// the last attempt: UNREACHABLE_ERROR that reads "waiting for <channel>: <reason>"
    /// before the link first opens and "reconnecting to <channel>: <reason>" after, or
    /// the CONFIG_ERROR or CRITICAL_HARDWARE_ERROR of the opener.
    std::pair<Opened, x::errors::Error> bus();

    /// @brief takes the error of a call on an opened bus. An unreachable or critical
    /// error drops the bus when it is still the open one, and the next call to bus
    /// opens it again at once. The dropped bus closes once no caller still uses it.
    /// @returns err, with an unreachable error reworded as "reconnecting to <channel>:
    /// <reason>".
    x::errors::Error report(const Opened &opened, const x::errors::Error &err);

private:
    const std::string channel;
    const Opener open;
    const x::telem::NowFunc now;
    std::mutex mu;
    std::shared_ptr<Bus> current;
    std::uint64_t opens = 0;
    /// @brief the error of the last failed open, while the link holds no bus.
    x::errors::Error failure;
    /// @brief when the link may next try to open.
    x::telem::TimeStamp next_attempt;
    /// @brief the wait after the next failed open.
    x::telem::TimeSpan wait = MIN_REOPEN_WAIT;

    /// @returns err as the link shows it while the bus cannot open. Requires mu.
    [[nodiscard]] x::errors::Error shown(const x::errors::Error &err) const;
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
