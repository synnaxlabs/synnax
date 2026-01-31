// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>

#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#include <windows.h>

#include "glog/logging.h"

#include "x/cpp/xthread/rt.h"

namespace xthread {
bool has_rt_support() { return false; }

xerrors::Error apply_rt_config(const RTConfig &cfg) {
    if (cfg.enabled) {
        int win_priority;
        if (cfg.priority >= 90)
            win_priority = THREAD_PRIORITY_TIME_CRITICAL;
        else if (cfg.priority >= 70)
            win_priority = THREAD_PRIORITY_HIGHEST;
        else if (cfg.priority >= 50)
            win_priority = THREAD_PRIORITY_ABOVE_NORMAL;
        else
            win_priority = THREAD_PRIORITY_NORMAL;

        if (!SetThreadPriority(GetCurrentThread(), win_priority))
            LOG(WARNING) << "[xthread] Failed to set thread priority: "
                         << GetLastError();
        else
            VLOG(1) << "[xthread] Set thread priority to " << win_priority;
    }

    if (cfg.cpu_affinity >= 0) {
        const DWORD_PTR mask = static_cast<DWORD_PTR>(1) << cfg.cpu_affinity;
        if (!SetThreadAffinityMask(GetCurrentThread(), mask))
            LOG(WARNING) << "[xthread] Failed to set CPU affinity to "
                         << cfg.cpu_affinity << ": " << GetLastError();
        else
            VLOG(1) << "[xthread] Set CPU affinity to core " << cfg.cpu_affinity;
    }

    if (cfg.lock_memory)
        LOG(WARNING) << "[xthread] Memory locking on Windows requires VirtualLock API";

    return xerrors::NIL;
}
}
