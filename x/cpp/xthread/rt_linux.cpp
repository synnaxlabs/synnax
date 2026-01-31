// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cerrno>
#include <cstring>
#include <thread>

#include <sched.h>
#include <sys/mman.h>

#include "glog/logging.h"

#include "x/cpp/xthread/rt.h"

namespace xthread {
bool has_rt_support() {
    struct sched_param param;
    param.sched_priority = 1;
    const int orig_policy = sched_getscheduler(0);
    struct sched_param orig_param;
    sched_getparam(0, &orig_param);

    if (sched_setscheduler(0, SCHED_FIFO, &param) == 0) {
        sched_setscheduler(0, orig_policy, &orig_param);
        return true;
    }
    return false;
}

xerrors::Error apply_rt_config(const RTConfig &cfg) {
    if (cfg.enabled) {
        struct sched_param param;
        param.sched_priority = cfg.priority;
        if (sched_setscheduler(0, SCHED_FIFO, &param) == -1)
            LOG(WARNING) << "[xthread] Failed to set SCHED_FIFO priority "
                         << cfg.priority << ": " << strerror(errno)
                         << " (requires CAP_SYS_NICE or root)";
        else
            VLOG(1) << "[xthread] Set RT priority to " << cfg.priority;
    }

    int target_cpu = cfg.cpu_affinity;
    if (target_cpu == CPU_AFFINITY_AUTO) {
        const auto n = std::thread::hardware_concurrency();
        target_cpu = n > 1 ? static_cast<int>(n - 1) : CPU_AFFINITY_NONE;
    }

    if (target_cpu >= 0) {
        cpu_set_t cpuset;
        CPU_ZERO(&cpuset);
        CPU_SET(target_cpu, &cpuset);
        if (sched_setaffinity(0, sizeof(cpuset), &cpuset) == -1)
            LOG(WARNING) << "[xthread] Failed to set CPU affinity to core "
                         << target_cpu << ": " << strerror(errno);
        else
            VLOG(1) << "[xthread] Pinned to CPU " << target_cpu;
    }

    if (cfg.lock_memory) {
        if (mlockall(MCL_CURRENT | MCL_FUTURE) == -1)
            LOG(WARNING) << "[xthread] Failed to lock memory: " << strerror(errno)
                         << " (requires CAP_IPC_LOCK)";
        else
            VLOG(1) << "[xthread] Locked memory pages";
    }

    return xerrors::NIL;
}
}
