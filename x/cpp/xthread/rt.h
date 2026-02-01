// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <ostream>

#include "x/cpp/telem/telem.h"
#include "x/cpp/xerrors/errors.h"
#include "x/cpp/xlog/xlog.h"

namespace xthread {
/// @brief Default real-time priority for SCHED_FIFO on Linux (range 1-99).
constexpr int DEFAULT_RT_PRIORITY = 47;

/// @brief Sentinel value indicating automatic CPU affinity selection.
/// When set, pins to the last available core for RT modes.
constexpr int CPU_AFFINITY_AUTO = -1;

/// @brief Sentinel value indicating CPU pinning is explicitly disabled.
constexpr int CPU_AFFINITY_NONE = -2;

/// @brief Default period for real-time scheduling (1ms).
const telem::TimeSpan DEFAULT_RT_PERIOD = telem::MILLISECOND;

/// @brief Default computation time budget per period (200us).
const telem::TimeSpan DEFAULT_RT_COMPUTATION = telem::MICROSECOND * 200;

/// @brief Default deadline within period (500us).
const telem::TimeSpan DEFAULT_RT_DEADLINE = telem::MICROSECOND * 500;

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
struct RTCapabilities {
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

    friend std::ostream &operator<<(std::ostream &os, const RTCapabilities &caps) {
        os << "real-time capabilities:\n";
        os << "  " << xlog::SHALE() << "priority scheduling" << xlog::RESET() << ": "
           << caps.priority_scheduling << "\n";
        os << "  " << xlog::SHALE() << "deadline scheduling" << xlog::RESET() << ": "
           << caps.deadline_scheduling << "\n";
        os << "  " << xlog::SHALE() << "time constraint" << xlog::RESET() << ": "
           << caps.time_constraint << "\n";
        os << "  " << xlog::SHALE() << "mmcss" << xlog::RESET() << ": " << caps.mmcss
           << "\n";
        os << "  " << xlog::SHALE() << "cpu affinity" << xlog::RESET() << ": "
           << caps.cpu_affinity << "\n";
        os << "  " << xlog::SHALE() << "memory locking" << xlog::RESET() << ": "
           << caps.memory_locking;
        if (caps.has_permission_issues()) os << "\n" << caps.permissions_guidance();
        return os;
    }
};

/// @brief Queries platform RT capabilities (cached after first call).
RTCapabilities get_rt_capabilities();

/// @brief Configuration for real-time thread properties.
struct RTConfig {
    /// Whether to enable real-time scheduling (SCHED_FIFO on Linux).
    /// Requires CAP_SYS_NICE capability or root privileges.
    bool enabled = false;
    /// Real-time thread priority (1-99 on Linux, higher = more priority).
    /// Only used when enabled is true.
    int priority = DEFAULT_RT_PRIORITY;
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
    [[nodiscard]] RTConfig with_timing_defaults() const {
        RTConfig cfg = *this;
        if (!cfg.has_timing()) {
            cfg.period = DEFAULT_RT_PERIOD;
            cfg.computation = DEFAULT_RT_COMPUTATION;
            cfg.deadline = DEFAULT_RT_DEADLINE;
        }
        return cfg;
    }

    friend std::ostream &operator<<(std::ostream &os, const RTConfig &cfg) {
        os << "rt config:\n";
        os << "  " << xlog::SHALE() << "enabled" << xlog::RESET() << ": "
           << (cfg.enabled ? "yes" : "no") << "\n";
        if (cfg.enabled) {
            os << "  " << xlog::SHALE() << "priority" << xlog::RESET() << ": "
               << cfg.priority << "\n";
            if (cfg.cpu_affinity >= 0)
                os << "  " << xlog::SHALE() << "cpu affinity" << xlog::RESET() << ": "
                   << cfg.cpu_affinity << "\n";
            else if (cfg.cpu_affinity == CPU_AFFINITY_AUTO)
                os << "  " << xlog::SHALE() << "cpu affinity" << xlog::RESET()
                   << ": auto\n";
            os << "  " << xlog::SHALE() << "lock memory" << xlog::RESET() << ": "
               << (cfg.lock_memory ? "yes" : "no") << "\n";
            if (cfg.has_timing()) {
                os << "  " << xlog::SHALE() << "period" << xlog::RESET() << ": "
                   << cfg.period << "\n";
                os << "  " << xlog::SHALE() << "computation" << xlog::RESET() << ": "
                   << cfg.computation << "\n";
                os << "  " << xlog::SHALE() << "deadline" << xlog::RESET() << ": "
                   << cfg.deadline << "\n";
            }
            if (cfg.prefer_deadline_scheduler)
                os << "  " << xlog::SHALE() << "prefer deadline scheduler"
                   << xlog::RESET() << ": yes\n";
            if (cfg.use_mmcss)
                os << "  " << xlog::SHALE() << "use mmcss" << xlog::RESET()
                   << ": yes\n";
        }
        return os;
    }
};

/// @brief Applies real-time configuration to the current thread.
/// @param cfg The RT configuration to apply.
/// @return xerrors::NIL on success, or an error describing what failed.
/// @note On platforms without RT scheduling support (macOS, Windows),
/// this function logs warnings but does not return errors.
xerrors::Error apply_rt_config(const RTConfig &cfg);

/// @brief DEPRECATED: Use get_rt_capabilities().any() instead.
/// @brief Checks if the platform supports real-time scheduling.
/// @return true on Linux with appropriate permissions, false on macOS/Windows.
bool has_rt_support();
}
