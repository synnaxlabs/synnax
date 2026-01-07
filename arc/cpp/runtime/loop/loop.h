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
#include "x/cpp/notify/notify.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/xerrors/errors.h"

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
}

enum class ExecutionMode {
    /// @brief Continuous polling without sleeping. Lowest latency, 100% CPU.
    /// Uses non-blocking event checks (epoll_wait timeout=0, kevent timeout=0).
    BUSY_WAIT,

    /// @brief Tight polling loop with precise software timing via ::loop::Timer.
    /// Achieves sub-millisecond precision. Lower latency than EVENT_DRIVEN,
    /// higher CPU usage than HYBRID. Best for high-frequency control loops.
    HIGH_RATE,

    /// @brief Real-time event-driven waiting with RT thread configuration.
    /// On Linux: Uses SCHED_FIFO scheduling, CPU affinity, and memory locking.
    /// On other platforms: Falls back to HIGH_RATE (no true RT support).
    /// Uses indefinite blocking on events for deterministic behavior.
    RT_EVENT,

    /// @brief Spin briefly then block on events. Balanced for general-purpose systems.
    /// Spin phase (default 100us) catches immediate data arrivals.
    /// Block phase uses efficient OS primitives with timeout.
    HYBRID,

    /// @brief Block immediately on events. Lowest CPU usage, higher latency.
    /// Uses indefinite blocking on multiplexer (epoll_wait/kevent).
    /// Wakes on: data notification, timer expiry, or external watched notifiers.
    EVENT_DRIVEN,
};

struct Config {
    /// @brief Execution mode determining wait behavior and CPU/latency tradeoffs.
    ExecutionMode mode = ExecutionMode::EVENT_DRIVEN;

    /// @brief Periodic timer interval. Zero disables the timer.
    /// When enabled, wait() returns at least once per interval.
    telem::TimeSpan interval = telem::TimeSpan(0);

    /// @brief Spin duration for HYBRID mode before blocking on events.
    telem::TimeSpan spin_duration = timing::HYBRID_SPIN_DEFAULT;

    /// @brief Real-time priority for RT_EVENT mode (1-99 on Linux for SCHED_FIFO).
    /// -1 means no priority change. Requires CAP_SYS_NICE capability on Linux.
    int rt_priority = -1;

    /// @brief CPU core to pin the loop thread to (-1 = no affinity).
    int cpu_affinity = -1;

    /// @brief Lock all memory pages to prevent page faults in RT path.
    /// Requires CAP_IPC_LOCK capability on Linux. Ignored on other platforms.
    bool lock_memory = false;
};

/// @brief Abstract event loop for the Arc runtime.
/// Provides platform-specific waiting on data notifications, timers, and external
/// events.
struct Loop {
    virtual ~Loop() = default;

    /// @brief Signal that input data is available.
    /// Thread-safe: may be called from any thread.
    /// RT-safe: uses lock-free signaling (eventfd write, kqueue trigger, SetEvent).
    virtual void notify_data() = 0;

    /// @brief Block until data/timer/external event or breaker stops.
    /// Must be called from the runtime thread only.
    /// @param breaker Controls loop termination; wait() returns when breaker stops.
    virtual void wait(breaker::Breaker &breaker) = 0;

    /// @brief Initialize loop resources. Must be called before wait().
    /// Applies RT configuration (priority, affinity, memory lock) if configured.
    /// @return Error if resource allocation fails.
    virtual xerrors::Error start() = 0;

    /// @brief Release loop resources. Safe to call multiple times.
    /// Signals any blocked wait() calls to return.
    virtual void stop() = 0;

    /// @brief Registers an external notifier for multiplexed waiting.
    /// When the notifier is signaled, wait() will return.
    /// Cleanup is automatic when the loop is destroyed (no unwatch needed).
    /// @param notifier The notifier to watch (must have valid fd() on Linux/macOS).
    /// @return true if registration succeeded, false if:
    ///         - notifier.fd() returns -1 (no file descriptor)
    ///         - Platform doesn't support multiplexed watching (Windows, Polling)
    ///         - Registration failed (logged as ERROR)
    virtual bool watch(notify::Notifier &notifier) = 0;
};

/// @brief Creates a platform-specific loop implementation.
/// @param cfg Loop configuration (mode, timing, RT settings).
/// @return Pair of (loop, error). Loop is started on success.
std::pair<std::unique_ptr<Loop>, xerrors::Error> create(const Config &cfg);
}
