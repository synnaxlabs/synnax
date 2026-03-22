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
#include <ostream>
#include <vector>

#include "x/cpp/log/log.h"
#include "x/cpp/telem/telem.h"

namespace x::thread::rt {
/// @brief Default real-time priority for SCHED_FIFO on Linux (range 1-99).
constexpr int DEFAULT_PRIORITY = 47;

/// @brief Sentinel value indicating automatic CPU affinity selection.
/// When set, pins to the last available core for RT modes.
constexpr int CPU_AFFINITY_AUTO = -1;

/// @brief Sentinel value indicating CPU pinning is explicitly disabled.
constexpr int CPU_AFFINITY_NONE = -2;

/// @brief Default period for real-time scheduling (1ms).
const telem::TimeSpan DEFAULT_PERIOD = telem::MILLISECOND;

/// @brief Default computation time budget per period (200us).
const telem::TimeSpan DEFAULT_COMPUTATION = telem::MICROSECOND * 200;

/// @brief Default deadline within period (500us).
const telem::TimeSpan DEFAULT_DEADLINE = telem::MICROSECOND * 500;

/// @brief Represents a single RT capability with platform support and permission
/// status.
struct Capability {
    bool supported = false;
    bool permitted = false;

    [[nodiscard]] bool ok() const { return this->supported && this->permitted; }

    operator bool() const { return this->ok(); }

    [[nodiscard]] bool missing_permissions() const {
        return this->supported && !this->permitted;
    }

    friend std::ostream &operator<<(std::ostream &os, const Capability &cap) {
        if (!cap.supported)
            os << "not supported";
        else if (cap.permitted)
            os << "yes";
        else
            os << "no (missing permissions)";
        return os;
    }
};

/// @brief Describes what real-time features the platform supports and whether
/// the current process has the necessary permissions to use them.
struct Capabilities {
    /// Priority-based scheduling (SCHED_FIFO on Linux, SetThreadPriority on Windows)
    Capability priority_scheduling;
    /// Deadline-based scheduling (Linux SCHED_DEADLINE)
    Capability deadline_scheduling;
    /// Time constraint policy (macOS THREAD_TIME_CONSTRAINT_POLICY)
    Capability time_constraint;
    /// Multimedia class scheduler (Windows MMCSS)
    Capability mmcss;
    /// Hard CPU affinity pinning
    Capability cpu_affinity;
    /// Memory page locking (mlockall)
    Capability memory_locking;

    /// @brief Returns true if any RT scheduling feature is available and permitted.
    [[nodiscard]] bool any() const {
        return this->priority_scheduling || this->deadline_scheduling ||
               this->time_constraint || this->mmcss;
    }

    /// @brief Returns true if timing-based RT is available and permitted.
    [[nodiscard]] bool timing_aware() const {
        return this->deadline_scheduling || this->time_constraint;
    }

    /// @brief Returns true if there are supported features lacking permissions.
    [[nodiscard]] bool has_permission_issues() const {
        return this->priority_scheduling.missing_permissions() ||
               this->deadline_scheduling.missing_permissions() ||
               this->memory_locking.missing_permissions();
    }

    /// @brief Returns platform-specific guidance for enabling RT permissions.
    [[nodiscard]] std::string permissions_guidance() const;

    friend std::ostream &operator<<(std::ostream &os, const Capabilities &caps) {
        os << "real-time capabilities:\n";
        os << "  " << log::SHALE() << "priority scheduling" << log::RESET() << ": "
           << caps.priority_scheduling << "\n";
        os << "  " << log::SHALE() << "deadline scheduling" << log::RESET() << ": "
           << caps.deadline_scheduling << "\n";
        os << "  " << log::SHALE() << "time constraint" << log::RESET() << ": "
           << caps.time_constraint << "\n";
        os << "  " << log::SHALE() << "mmcss" << log::RESET() << ": " << caps.mmcss
           << "\n";
        os << "  " << log::SHALE() << "cpu affinity" << log::RESET() << ": "
           << caps.cpu_affinity << "\n";
        os << "  " << log::SHALE() << "memory locking" << log::RESET() << ": "
           << caps.memory_locking;
        if (caps.has_permission_issues()) os << "\n" << caps.permissions_guidance();
        return os;
    }
};

/// @brief Queries platform RT capabilities (cached after first call).
Capabilities capabilities();

/// @brief Configuration for real-time thread properties.
struct Config {
    /// Whether to enable real-time scheduling (SCHED_FIFO on Linux).
    /// Requires CAP_SYS_NICE capability or root privileges.
    bool enabled = false;
    /// Real-time thread priority (1-99 on Linux, higher = more priority).
    /// Only used when enabled is true.
    int priority = DEFAULT_PRIORITY;
    /// CPU core to pin the thread to. Use CPU_AFFINITY_AUTO for automatic
    /// selection (last core) or CPU_AFFINITY_NONE to disable pinning.
    int cpu_affinity = CPU_AFFINITY_NONE;
    /// Whether to lock all current and future memory pages to prevent
    /// page faults during real-time execution. Requires CAP_IPC_LOCK.
    bool lock_memory = false;
    /// How often the thread runs (cycle period). Used for deadline scheduling.
    telem::TimeSpan period = telem::TimeSpan::ZERO();
    /// CPU time budget per period. Used for deadline/time-constraint scheduling.
    telem::TimeSpan computation = telem::TimeSpan::ZERO();
    /// Maximum time to complete work within period. Used for deadline scheduling.
    telem::TimeSpan deadline = telem::TimeSpan::ZERO();
    /// Linux: prefer SCHED_DEADLINE over SCHED_FIFO when timing is specified.
    bool prefer_deadline_scheduler = false;
    /// Windows: use MMCSS Pro Audio class for enhanced scheduling.
    bool use_mmcss = false;

    /// @brief Returns true if timing parameters are specified.
    [[nodiscard]] bool has_timing() const {
        return this->period > telem::TimeSpan::ZERO();
    }

    /// @brief Returns a copy with default timing values if none are specified.
    [[nodiscard]] Config with_timing_defaults() const {
        Config cfg = *this;
        if (!cfg.has_timing()) {
            cfg.period = DEFAULT_PERIOD;
            cfg.computation = DEFAULT_COMPUTATION;
            cfg.deadline = DEFAULT_DEADLINE;
        }
        return cfg;
    }

    friend std::ostream &operator<<(std::ostream &os, const Config &cfg) {
        os << "rt config:\n";
        os << "  " << log::SHALE() << "enabled" << log::RESET() << ": "
           << (cfg.enabled ? "yes" : "no") << "\n";
        if (cfg.enabled) {
            os << "  " << log::SHALE() << "priority" << log::RESET() << ": "
               << cfg.priority << "\n";
            if (cfg.cpu_affinity >= 0)
                os << "  " << log::SHALE() << "cpu affinity" << log::RESET() << ": "
                   << cfg.cpu_affinity << "\n";
            else if (cfg.cpu_affinity == CPU_AFFINITY_AUTO)
                os << "  " << log::SHALE() << "cpu affinity" << log::RESET()
                   << ": auto\n";
            os << "  " << log::SHALE() << "lock memory" << log::RESET() << ": "
               << (cfg.lock_memory ? "yes" : "no") << "\n";
            if (cfg.has_timing()) {
                os << "  " << log::SHALE() << "period" << log::RESET() << ": "
                   << cfg.period << "\n";
                os << "  " << log::SHALE() << "computation" << log::RESET() << ": "
                   << cfg.computation << "\n";
                os << "  " << log::SHALE() << "deadline" << log::RESET() << ": "
                   << cfg.deadline << "\n";
            }
            if (cfg.prefer_deadline_scheduler)
                os << "  " << log::SHALE() << "prefer deadline scheduler"
                   << log::RESET() << ": yes\n";
            if (cfg.use_mmcss)
                os << "  " << log::SHALE() << "use mmcss" << log::RESET() << ": yes\n";
        }
        return os;
    }
};

/// @brief Applies real-time configuration to the current thread on a
/// best-effort basis. Individual failures (e.g. missing CAP_SYS_NICE for
/// SCHED_FIFO) are logged as warnings but do not prevent the thread from
/// running at normal priority.
/// @param cfg The RT configuration to apply.
void apply_config(const Config &cfg);

/// @brief DEPRECATED: Use get_rt_capabilities().any() instead.
/// @brief Checks if the platform supports real-time scheduling.
/// @return true on Linux with appropriate permissions, false on macOS/Windows.
bool has_support();

/// @brief Discovers isolated or suitable RT cores on the current platform.
/// On Linux, parses /sys/devices/system/cpu/isolated; falls back to picking
/// the highest N cores. On other platforms, returns an empty vector.
std::vector<int> discover_rt_cores();

/// @brief RAII handle for an allocated RT core. Releasing the handle returns
/// the core to the Manager's pool. Move-only.
class Handle {
    int core;
    Config resolved;
    std::function<void(int)> release_fn;
    bool released;

public:
    Handle(int core, Config resolved, std::function<void(int)> release_fn);

    Handle(const Handle &) = delete;
    Handle &operator=(const Handle &) = delete;

    Handle(Handle &&other) noexcept;
    Handle &operator=(Handle &&other) noexcept;

    ~Handle();

    /// @brief Applies the RT configuration (with the allocated core) to the
    /// calling thread. Must be called from the thread that should become RT.
    void apply();

    /// @brief Explicitly releases the allocated core back to the pool.
    /// Idempotent, safe to call multiple times.
    void release();

    /// @brief Returns the allocated core number, or CPU_AFFINITY_NONE if
    /// no core was available.
    [[nodiscard]] int allocated_core() const { return this->core; }
};

/// @brief Central RT core manager. Discovers available cores at construction
/// and hands them out via allocate(). Thread-safe.
/// Must be constructed via std::make_shared so that allocate() can capture
/// a weak_ptr for safe release callbacks.
class Manager : public std::enable_shared_from_this<Manager> {
    mutable std::mutex mu;
    std::vector<int> all_cores;
    std::vector<int> available;

    void release_core(int core);

public:
    Manager();

    /// @brief Allocates a core from the pool and returns a Handle with the
    /// given base config. If no cores are available, returns a Handle with
    /// CPU_AFFINITY_NONE.
    /// @param cfg Base RT config, cpu_affinity will be overwritten with
    /// the allocated core.
    Handle allocate(Config cfg);

    /// @brief Returns the number of cores still available in the pool.
    [[nodiscard]] size_t available_cores() const;

    /// @brief Returns the total number of managed cores.
    [[nodiscard]] size_t total_cores() const;
};
}
