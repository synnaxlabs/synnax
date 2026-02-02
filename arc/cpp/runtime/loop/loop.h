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
#include <thread>

#include "glog/logging.h"

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/notify/notify.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/xerrors/errors.h"
#include "x/cpp/xjson/xjson.h"
#include "x/cpp/xlog/xlog.h"
#include "x/cpp/xthread/rt.h"

namespace arc::runtime::loop {

/// @brief Named constants for timing parameters used across loop implementations.
namespace timing {
/// @brief Default spin duration for HYBRID mode before blocking (100 microseconds).
/// Balances latency (catches immediate data arrivals) vs CPU usage.
inline const telem::TimeSpan HYBRID_SPIN_DEFAULT = 100 * telem::MICROSECOND;

/// @brief Fallback poll interval for HIGH_RATE mode when no timer configured.
inline const telem::TimeSpan HIGH_RATE_POLL_INTERVAL = 100 * telem::MICROSECOND;

/// @brief Timeout for blocking wait in HYBRID mode after spin phase (10 milliseconds).
inline const telem::TimeSpan HYBRID_BLOCK_TIMEOUT = 10 * telem::MILLISECOND;

/// @brief Minimum meaningful interval for kqueue EVFILT_TIMER on macOS (1 millisecond).
/// Intervals below this threshold use software timing instead.
inline const telem::TimeSpan KQUEUE_TIMER_MIN = telem::MILLISECOND;

/// @brief Threshold below which software timer (HIGH_RATE) is used for precision.
/// Above this, OS timers (timerfd/kqueue/WaitableTimer) provide sufficient precision.
inline const telem::TimeSpan SOFTWARE_TIMER_THRESHOLD = telem::MILLISECOND;

/// @brief Threshold below which HIGH_RATE or RT_EVENT should be used.
/// Intervals below 1ms require precise software timing.
inline const telem::TimeSpan HIGH_RATE_THRESHOLD = telem::MILLISECOND;

/// @brief Threshold below which HYBRID mode is beneficial.
/// Intervals between 1-5ms benefit from spin-then-block approach.
inline const telem::TimeSpan HYBRID_THRESHOLD = 5 * telem::MILLISECOND;

/// @brief Timeout for event-driven wait to periodically check breaker.running().
inline const telem::TimeSpan EVENT_DRIVEN_TIMEOUT = 100 * telem::MILLISECOND;

/// @brief Shorter timeout for non-blocking/polling checks.
inline const telem::TimeSpan POLL_TIMEOUT = 10 * telem::MILLISECOND;

/// @brief Windows WaitableTimer uses 100-nanosecond units.
inline const telem::TimeSpan WINDOWS_TIMER_UNIT = 100 * telem::NANOSECOND;
}

/// @brief Default RT priority for SCHED_FIFO on Linux (range 1-99).
/// Mid-range priority that preempts normal processes without starving system threads.
constexpr int DEFAULT_RT_PRIORITY = 47;

/// @brief Sentinel for auto CPU affinity. Pins to last core in RT_EVENT mode.
constexpr int CPU_AFFINITY_AUTO = -1;

/// @brief Sentinel for explicitly disabling CPU pinning.
constexpr int CPU_AFFINITY_NONE = -2;

enum class ExecutionMode {
    /// @brief Auto-select mode based on timing requirements and platform capabilities.
    AUTO,
    /// @brief Continuous polling without sleeping. Lowest latency, 100% CPU.
    BUSY_WAIT,
    /// @brief Tight polling loop with precise software timing. Sub-millisecond
    /// precision.
    HIGH_RATE,
    /// @brief Real-time event-driven with RT thread configuration (Linux SCHED_FIFO).
    RT_EVENT,
    /// @brief Spin briefly then block on events. Balanced for general-purpose systems.
    HYBRID,
    /// @brief Block immediately on events. Lowest CPU usage, higher latency.
    EVENT_DRIVEN,
};

inline std::ostream &operator<<(std::ostream &os, ExecutionMode mode) {
    switch (mode) {
        case ExecutionMode::AUTO:
            return os << "AUTO";
        case ExecutionMode::BUSY_WAIT:
            return os << "BUSY_WAIT";
        case ExecutionMode::HIGH_RATE:
            return os << "HIGH_RATE";
        case ExecutionMode::RT_EVENT:
            return os << "RT_EVENT";
        case ExecutionMode::HYBRID:
            return os << "HYBRID";
        case ExecutionMode::EVENT_DRIVEN:
            return os << "EVENT_DRIVEN";
        default:
            return os << "UNKNOWN";
    }
}

/// @brief Auto-selects execution mode based on timing requirements and platform.
/// Never returns BUSY_WAIT or AUTO.
inline ExecutionMode
select_mode(const telem::TimeSpan timing_interval, const bool has_intervals) {
    if (!has_intervals) return ExecutionMode::EVENT_DRIVEN;
    if (timing_interval < timing::HIGH_RATE_THRESHOLD)
        return xthread::has_rt_support() ? ExecutionMode::RT_EVENT : ExecutionMode::HIGH_RATE;
    if (timing_interval < timing::HYBRID_THRESHOLD) return ExecutionMode::HYBRID;
    return ExecutionMode::EVENT_DRIVEN;
}

struct Config {
    ExecutionMode mode = ExecutionMode::AUTO;
    telem::TimeSpan interval = telem::TimeSpan(0);
    telem::TimeSpan spin_duration = timing::HYBRID_SPIN_DEFAULT;
    int rt_priority = DEFAULT_RT_PRIORITY;
    int cpu_affinity = CPU_AFFINITY_AUTO;
    bool lock_memory = false;

    Config() = default;

    explicit Config(xjson::Parser &parser) {
        const auto mode_str = parser.field<std::string>("execution_mode", "AUTO");
        if (mode_str == "AUTO")
            mode = ExecutionMode::AUTO;
        else if (mode_str == "BUSY_WAIT")
            mode = ExecutionMode::BUSY_WAIT;
        else if (mode_str == "HIGH_RATE")
            mode = ExecutionMode::HIGH_RATE;
        else if (mode_str == "RT_EVENT")
            mode = ExecutionMode::RT_EVENT;
        else if (mode_str == "HYBRID")
            mode = ExecutionMode::HYBRID;
        else if (mode_str == "EVENT_DRIVEN")
            mode = ExecutionMode::EVENT_DRIVEN;
        else {
            parser.field_err(
                "execution_mode",
                "invalid execution mode: " + mode_str +
                    " (must be AUTO, BUSY_WAIT, HIGH_RATE, RT_EVENT, HYBRID, "
                    "or EVENT_DRIVEN)"
            );
            return;
        }
        rt_priority = parser.field<int>("rt_priority", DEFAULT_RT_PRIORITY);
        cpu_affinity = parser.field<int>("cpu_affinity", CPU_AFFINITY_AUTO);
        lock_memory = parser.field<bool>("lock_memory", false);
    }

    Config apply_defaults(const telem::TimeSpan timing_interval) const {
        Config cfg = *this;
        const bool has_intervals = timing_interval != telem::TimeSpan::max();
        if (this->mode == ExecutionMode::AUTO)
            cfg.mode = select_mode(timing_interval, has_intervals);
        if (this->interval.nanoseconds() == 0 && has_intervals)
            cfg.interval = timing_interval;
        // If HIGH_RATE or RT_EVENT is explicitly set without an interval, use a
        // sensible default.
        const bool needs_interval = cfg.mode == ExecutionMode::HIGH_RATE ||
                                    cfg.mode == ExecutionMode::RT_EVENT;
        if (cfg.interval.nanoseconds() == 0 && needs_interval) {
            LOG(WARNING) << "[loop] " << cfg.mode
                         << " mode requires an interval, defaulting to "
                         << timing::HIGH_RATE_POLL_INTERVAL;
            cfg.interval = timing::HIGH_RATE_POLL_INTERVAL;
        }
        if (this->cpu_affinity == CPU_AFFINITY_AUTO) {
#ifdef SYNNAX_NILINUXRT
            const bool should_pin = cfg.mode == ExecutionMode::RT_EVENT ||
                                    cfg.mode == ExecutionMode::HIGH_RATE ||
                                    cfg.mode == ExecutionMode::HYBRID;
#else
            const bool should_pin = cfg.mode == ExecutionMode::RT_EVENT;
#endif
            if (should_pin) {
                const auto n = std::thread::hardware_concurrency();
                cfg.cpu_affinity = n > 1 ? static_cast<int>(n - 1) : CPU_AFFINITY_NONE;
            }
        }
        return cfg;
    }

    friend std::ostream &operator<<(std::ostream &os, const Config &cfg) {
        os << "  " << xlog::SHALE() << "execution mode" << xlog::RESET() << ": "
           << cfg.mode << "\n";
        if (cfg.interval.nanoseconds() > 0)
            os << "  " << xlog::SHALE() << "interval" << xlog::RESET() << ": "
               << cfg.interval << "\n";
        if (cfg.mode == ExecutionMode::HYBRID)
            os << "  " << xlog::SHALE() << "spin duration" << xlog::RESET() << ": "
               << cfg.spin_duration << "\n";
        if (cfg.mode == ExecutionMode::RT_EVENT) {
            os << "  " << xlog::SHALE() << "rt priority" << xlog::RESET() << ": "
               << cfg.rt_priority << "\n";
            os << "  " << xlog::SHALE() << "lock memory" << xlog::RESET() << ": "
               << (cfg.lock_memory ? "yes" : "no") << "\n";
        }
        if (cfg.cpu_affinity >= 0)
            os << "  " << xlog::SHALE() << "cpu affinity" << xlog::RESET() << ": "
               << cfg.cpu_affinity << "\n";
        return os;
    }
};

/// @brief Abstract event loop for the Arc runtime.
/// Provides platform-specific waiting on timers and external events.
struct Loop {
    virtual ~Loop() = default;

    /// @brief Block until timer/external event or breaker stops.
    /// Must be called from the runtime thread only.
    /// @param breaker Controls loop termination; wait() returns when breaker stops.
    virtual void wait(breaker::Breaker &breaker) = 0;

    /// @brief Initialize loop resources. Must be called before wait().
    /// Applies RT configuration (priority, affinity, memory lock) if configured.
    /// @return Error if resource allocation fails.
    virtual xerrors::Error start() = 0;

    /// @brief Wake up any blocked wait() call.
    /// Used during shutdown to unblock the run thread so it can check
    /// breaker.running(). Thread-safe: may be called from any thread. Does NOT release
    /// resources - that happens in the destructor.
    virtual void wake() = 0;

    /// @brief Registers an external notifier for multiplexed waiting.
    /// When the notifier is signaled, wait() will return. This is the primary
    /// mechanism for data notification - the caller should watch the input queue's
    /// notifier rather than calling a separate notify method.
    /// Cleanup is automatic when the loop is destroyed (no unwatch needed).
    /// @param notifier The notifier to watch.
    /// @return true if registration succeeded, false if registration failed.
    virtual bool watch(notify::Notifier &notifier) = 0;
};

/// @brief Creates a platform-specific loop implementation.
/// @param cfg Loop configuration (mode, timing, RT settings).
/// @return Pair of (loop, error). Loop is started on success.
std::pair<std::unique_ptr<Loop>, xerrors::Error> create(const Config &cfg);
}
