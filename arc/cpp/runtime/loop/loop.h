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

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/xerrors/errors.h"

namespace arc::runtime::loop {
enum class ExecutionMode {
    /// @brief Continuous polling without sleeping. Lowest latency, 100% CPU.
    BUSY_WAIT,
    /// @brief Tight polling loop with minimal sleep using precise timing.
    /// Low latency with reduced CPU usage compared to BUSY_WAIT.
    HIGH_RATE,
    /// @brief RT-safe event-driven waiting. Requires RT kernel support.
    /// Low latency with minimal CPU usage on RT systems.
    RT_EVENT,
    /// @brief Spin briefly, then block on events. Balanced approach for
    /// general-purpose systems.
    HYBRID,
    /// @brief Block immediately on events. Lowest CPU usage, higher latency.
    EVENT_DRIVEN,
};

struct Config {
    ExecutionMode mode = ExecutionMode::EVENT_DRIVEN;
    telem::TimeSpan interval = telem::TimeSpan(0);
    telem::TimeSpan spin_duration = telem::MICROSECOND * 100;
    int rt_priority = -1;
    int cpu_affinity = -1;
    bool lock_memory = false;
};

struct Loop {
    virtual ~Loop() = default;

    virtual void notify_data() = 0;

    virtual void wait(breaker::Breaker &breaker) = 0;

    virtual xerrors::Error start() = 0;

    virtual void stop() = 0;
};

std::pair<std::unique_ptr<Loop>, xerrors::Error> create(const Config &cfg);

}
