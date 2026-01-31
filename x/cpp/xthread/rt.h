// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "x/cpp/xerrors/errors.h"

namespace xthread {
/// @brief Default real-time priority for SCHED_FIFO on Linux (range 1-99).
constexpr int DEFAULT_RT_PRIORITY = 47;

/// @brief Sentinel value indicating automatic CPU affinity selection.
/// When set, pins to the last available core for RT modes.
constexpr int CPU_AFFINITY_AUTO = -1;

/// @brief Sentinel value indicating CPU pinning is explicitly disabled.
constexpr int CPU_AFFINITY_NONE = -2;

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
};

/// @brief Applies real-time configuration to the current thread.
/// @param cfg The RT configuration to apply.
/// @return xerrors::NIL on success, or an error describing what failed.
/// @note On platforms without RT scheduling support (macOS, Windows),
/// this function logs warnings but does not return errors.
xerrors::Error apply_rt_config(const RTConfig &cfg);

/// @brief Checks if the platform supports real-time scheduling.
/// @return true on Linux with appropriate permissions, false on macOS/Windows.
bool has_rt_support();
}
