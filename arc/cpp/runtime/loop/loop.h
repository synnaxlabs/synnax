// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <memory>
#include <string>
#include <thread>

#include "absl/log/log.h"

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/log/log.h"
#include "x/cpp/notify/notify.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/thread/rt/rt.h"

namespace arc::runtime::loop {
/// @brief Named constants for timing parameters used across loop implementations.
namespace timing {
/// @brief Default spin duration for HYBRID mode before blocking (100 microseconds).
/// Balances latency (catches immediate data arrivals) vs CPU usage.
inline const x::telem::TimeSpan HYBRID_SPIN_DEFAULT = 100 * x::telem::MICROSECOND;

/// @brief Fallback poll interval for HIGH_RATE mode when no timer configured.
inline const x::telem::TimeSpan HIGH_RATE_POLL_INTERVAL = 100 * x::telem::MICROSECOND;

/// @brief Timeout for blocking wait in HYBRID mode after spin phase (10 milliseconds).
inline const x::telem::TimeSpan HYBRID_BLOCK_TIMEOUT = 10 * x::telem::MILLISECOND;

/// @brief Threshold below which HIGH_RATE or RT_EVENT should be used.
/// Intervals below 1ms require precise software timing.
inline const x::telem::TimeSpan HIGH_RATE_THRESHOLD = x::telem::MILLISECOND;

/// @brief Upper bound for preferring RT_EVENT on RT-capable systems. Intervals
/// between HIGH_RATE_THRESHOLD and this value use RT_EVENT when RT scheduling
/// is available, falling through to AUTO otherwise.
inline const x::telem::TimeSpan RT_EVENT_THRESHOLD = 3 * x::telem::MILLISECOND;

/// @brief Timer span below which AUTO spins before a deadline on macOS and in the
/// polling loop.
inline const x::telem::TimeSpan HYBRID_THRESHOLD = 5 * x::telem::MILLISECOND;

/// @brief Timer span below which AUTO spins before a deadline on Windows. Without the
/// spin, the timer wakes about 0.5 ms late, which is 1% of 50 ms.
inline const x::telem::TimeSpan WINDOWS_HYBRID_THRESHOLD = 50 * x::telem::MILLISECOND;

/// @brief Timeout for event-driven wait to periodically check breaker.running().
inline const x::telem::TimeSpan EVENT_DRIVEN_TIMEOUT = 100 * x::telem::MILLISECOND;

/// @brief Windows WaitableTimer uses 100-nanosecond units.
inline const x::telem::TimeSpan WINDOWS_TIMER_UNIT = 100 * x::telem::NANOSECOND;

/// @brief Span the Windows loop spins ahead of a deadline in HYBRID and RT_EVENT. The
/// timer alone fires about 0.5 ms late.
inline const x::telem::TimeSpan WINDOWS_DEADLINE_SPIN = x::telem::MILLISECOND;

/// @brief Span the macOS loop spins ahead of a deadline in HYBRID. A kqueue timeout
/// alone fires up to 1 ms late.
inline const x::telem::TimeSpan DARWIN_DEADLINE_SPIN = 1500 * x::telem::MICROSECOND;

}

/// @brief Default RT priority for SCHED_FIFO on Linux (range 1-99).
/// Mid-range priority that preempts normal processes without starving system threads.
constexpr int DEFAULT_RT_PRIORITY = 47;

/// @brief Sentinel for auto CPU affinity. Pins to last core in RT_EVENT mode.
constexpr int CPU_AFFINITY_AUTO = -1;

/// @brief Sentinel for explicitly disabling CPU pinning.
constexpr int CPU_AFFINITY_NONE = -2;

enum class ExecutionMode {
    /// @brief Picks the thread config from the shortest timer. At each wait, spins
    /// before the deadline only when the timer that owns it is short.
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

/// @brief Indicates what caused the loop to wake from wait().
enum class WakeReason {
    /// @brief Timer interval expired.
    Timer,
    /// @brief External input/notifier signaled.
    Input,
    /// @brief Wait timed out (no specific event).
    Timeout,
    /// @brief Breaker stopped or wake() called.
    Shutdown,
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

/// @brief Returns the timer span below which AUTO spins before a deadline on this
/// platform.
x::telem::TimeSpan hybrid_threshold();

/// @brief Returns true when AUTO spins before a deadline of a timer with the given
/// span.
inline bool auto_spins(const x::telem::TimeSpan span) {
    return span < hybrid_threshold();
}

/// @brief Resolves AUTO from the shortest timer span of a program. Returns AUTO when no
/// other mode is necessary. Never returns BUSY_WAIT.
inline ExecutionMode select_mode(const x::telem::TimeSpan shortest_span) {
    if (shortest_span < timing::HIGH_RATE_THRESHOLD)
        return x::thread::rt::has_support() ? ExecutionMode::RT_EVENT
                                            : ExecutionMode::HIGH_RATE;
    if (x::thread::rt::has_support() && shortest_span < timing::RT_EVENT_THRESHOLD)
        return ExecutionMode::RT_EVENT;
    return ExecutionMode::AUTO;
}

struct Config {
    ExecutionMode mode = ExecutionMode::AUTO;
    x::telem::TimeSpan interval = x::telem::TimeSpan(0);
    x::telem::TimeSpan spin_duration = timing::HYBRID_SPIN_DEFAULT;
    int rt_priority = DEFAULT_RT_PRIORITY;
    int cpu_affinity = CPU_AFFINITY_AUTO;
    bool memory_locked = false;

    Config() = default;

    explicit Config(x::json::Parser &parser) {
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
        memory_locked = parser.field<bool>("memory_locked", false);
    }

    Config apply_defaults(const x::telem::TimeSpan shortest_span) const {
        Config cfg = *this;
        if (this->mode == ExecutionMode::AUTO) cfg.mode = select_mode(shortest_span);
        if (this->interval.nanoseconds() == 0 &&
            shortest_span != x::telem::TimeSpan::max())
            cfg.interval = shortest_span;
        // If HIGH_RATE or RT_EVENT is explicitly set without an interval, use a
        // sensible default.
        const bool needs_interval = cfg.mode == ExecutionMode::HIGH_RATE ||
                                    cfg.mode == ExecutionMode::RT_EVENT;
        if (cfg.interval.nanoseconds() == 0 && needs_interval) {
            LOG(WARNING) << "[arc.loop] " << cfg.mode
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

    /// @brief Converts this loop Config to an x::thread::rt::Config for applying RT
    /// settings to the current thread. Platform-specific flags
    /// (prefer_deadline_scheduler, use_mmcss) should be set by the caller after
    /// conversion.
    [[nodiscard]] x::thread::rt::Config rt() const {
        x::thread::rt::Config cfg;
        cfg.enabled = this->rt_priority > 0;
        cfg.priority = this->rt_priority;
        cfg.cpu_affinity = this->cpu_affinity;
        cfg.lock_memory = this->memory_locked;
        if (cfg.enabled && this->interval.nanoseconds() > 0) {
            cfg.period = this->interval;
            cfg.computation = this->interval * 0.2;
            cfg.deadline = this->interval * 0.8;
        }
        return cfg;
    }

    friend std::ostream &operator<<(std::ostream &os, const Config &cfg) {
        os << "  " << x::log::SHALE() << "execution mode" << x::log::RESET() << ": "
           << cfg.mode << "\n";
        if (cfg.interval.nanoseconds() > 0)
            os << "  " << x::log::SHALE() << "interval" << x::log::RESET() << ": "
               << cfg.interval << "\n";
        if (cfg.mode == ExecutionMode::HYBRID)
            os << "  " << x::log::SHALE() << "spin duration" << x::log::RESET() << ": "
               << cfg.spin_duration << "\n";
        if (cfg.mode == ExecutionMode::RT_EVENT) {
            os << "  " << x::log::SHALE() << "rt priority" << x::log::RESET() << ": "
               << cfg.rt_priority << "\n";
            os << "  " << x::log::SHALE() << "lock memory" << x::log::RESET() << ": "
               << (cfg.memory_locked ? "yes" : "no") << "\n";
        }
        if (cfg.cpu_affinity >= 0)
            os << "  " << x::log::SHALE() << "cpu affinity" << x::log::RESET() << ": "
               << cfg.cpu_affinity << "\n";
        return os;
    }
};

/// @brief Returns the span a HIGH_RATE wait sleeps: the interval of cfg, or max_timeout
/// when it is positive and shorter.
inline x::telem::TimeSpan
high_rate_span(const Config &cfg, const x::telem::TimeSpan max_timeout) {
    if (max_timeout.nanoseconds() <= 0) return cfg.interval;
    return std::min(cfg.interval, max_timeout);
}

/// @brief Abstract event loop for the Arc runtime.
/// Provides platform-specific waiting on timers and external events.
struct Loop {
    virtual ~Loop() = default;

    /// @brief Block until timer/external event or breaker stops.
    /// Must be called from the runtime thread only.
    /// @param breaker Controls loop termination; wait() returns when breaker stops.
    /// @param max_timeout Upper bound on how long to sleep. When positive, the loop
    /// will wake after at most this duration even if no input fires. A value of 0 means
    /// no deadline.
    /// @param span the period of the timer that owns the deadline. AUTO spins before
    /// the deadline only when auto_spins(span) is true.
    /// @return WakeReason indicating why wait() returned.
    virtual WakeReason wait(
        x::breaker::Breaker &breaker,
        x::telem::TimeSpan max_timeout = x::telem::TimeSpan(0),
        x::telem::TimeSpan span = x::telem::TimeSpan::max()
    ) = 0;

    /// @brief Initialize loop resources and apply RT configuration. Must be
    /// called before wait() and from the thread that will run the event loop,
    /// since RT scheduling (SCHED_FIFO/DEADLINE, MMCSS) is applied to the
    /// calling thread.
    /// @return Error if resource allocation fails.
    virtual x::errors::Error start() = 0;

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
    virtual bool watch(x::notify::Notifier &notifier) = 0;
};

/// @brief Creates a platform-specific loop implementation. The loop is not
/// started; call start() from the thread that will run the event loop so
/// that RT scheduling is applied to the correct thread.
/// @param cfg Loop configuration.
/// @param rt_handle Optional RT handle from the Manager. When provided, the
/// loop calls handle->apply() instead of apply_config(config.rt()), and the
/// handle's allocated core is used for CPU affinity.
std::unique_ptr<Loop>
create(const Config &cfg, std::shared_ptr<x::thread::rt::Handle> rt_handle = nullptr);
}
