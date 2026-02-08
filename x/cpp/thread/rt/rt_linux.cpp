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

#include "glog/logging.h"
#include <linux/sched.h>
#include <sched.h>
#include <sys/mman.h>
#include <sys/syscall.h>
#include <unistd.h>

#include "x/cpp/thread/rt/rt.h"

namespace x::thread::rt {
namespace {
#ifndef SCHED_DEADLINE
#define SCHED_DEADLINE 6
#endif

struct sched_attr {
    uint32_t size;
    uint32_t sched_policy;
    uint64_t sched_flags;
    int32_t sched_nice;
    uint32_t sched_priority;
    uint64_t sched_runtime;
    uint64_t sched_deadline;
    uint64_t sched_period;
};

int sched_setattr(pid_t pid, const sched_attr *attr, unsigned int flags) {
    return static_cast<int>(syscall(SYS_sched_setattr, pid, attr, flags));
}

bool test_sched_fifo() {
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

bool test_sched_deadline() {
    sched_attr attr = {};
    attr.size = sizeof(attr);
    attr.sched_policy = SCHED_DEADLINE;
    attr.sched_runtime = 100000;
    attr.sched_deadline = 500000;
    attr.sched_period = 1000000;
    if (sched_setattr(0, &attr, 0) == 0) {
        struct sched_param param;
        param.sched_priority = 0;
        sched_setscheduler(0, SCHED_OTHER, &param);
        return true;
    }
    return false;
}

bool test_mlockall() {
    if (mlockall(MCL_CURRENT) == 0) {
        munlockall();
        return true;
    }
    return false;
}

bool apply_deadline_scheduler(const RTConfig &cfg) {
    sched_attr attr = {};
    attr.size = sizeof(attr);
    attr.sched_policy = SCHED_DEADLINE;
    attr.sched_runtime = static_cast<uint64_t>(cfg.computation.nanoseconds());
    attr.sched_deadline = static_cast<uint64_t>(cfg.deadline.nanoseconds());
    attr.sched_period = static_cast<uint64_t>(cfg.period.nanoseconds());
    if (sched_setattr(0, &attr, 0) == 0) {
        VLOG(1) << "[xthread] Set SCHED_DEADLINE: period=" << cfg.period
                << " computation=" << cfg.computation << " deadline=" << cfg.deadline;
        return true;
    }
    LOG(WARNING) << "[xthread] Failed to set SCHED_DEADLINE: " << strerror(errno)
                 << " (falling back to SCHED_FIFO)";
    return false;
}

void apply_sched_fifo(int priority) {
    struct sched_param param;
    param.sched_priority = priority;
    if (sched_setscheduler(0, SCHED_FIFO, &param) == -1)
        LOG(WARNING) << "[xthread] Failed to set SCHED_FIFO priority " << priority
                     << ": " << strerror(errno) << " (requires CAP_SYS_NICE or root)";
    else
        VLOG(1) << "[xthread] Set RT priority to " << priority;
}
}

RTCapabilities get_rt_capabilities() {
    static RTCapabilities caps = [] {
        RTCapabilities c;
        c.priority_scheduling = {true, test_sched_fifo()};
        c.deadline_scheduling = {true, test_sched_deadline()};
        c.cpu_affinity = {true, true};
        c.memory_locking = {true, test_mlockall()};
        return c;
    }();
    return caps;
}

std::string RTCapabilities::permissions_guidance() const {
    std::string guidance;
    if (this->priority_scheduling.missing_permissions()) {
        guidance += "  To enable RT scheduling, either:\n";
        guidance += "    - Run as root (not recommended)\n";
        guidance += "    - Grant capability: sudo setcap cap_sys_nice+ep <binary>\n";
        guidance += "    - Add to /etc/security/limits.conf:\n";
        guidance += "        @realtime  -  rtprio  99\n";
    }
    if (this->memory_locking.missing_permissions()) {
        guidance += "  To enable memory locking:\n";
        guidance += "    - Grant capability: sudo setcap cap_ipc_lock+ep <binary>\n";
        guidance += "    - Or add to /etc/security/limits.conf:\n";
        guidance += "        @realtime  -  memlock  unlimited\n";
    }
    if (!guidance.empty()) {
        guidance = "To enable real-time features:\n" + guidance;
        guidance += "  After editing limits.conf, log out and back in.";
    }
    return guidance;
}

bool has_support() {
    return get_rt_capabilities().any();
}

errors::Error apply_config(const RTConfig &cfg) {
    if (cfg.enabled) {
        bool used_deadline = false;
        if (cfg.prefer_deadline_scheduler && cfg.has_timing())
            used_deadline = apply_deadline_scheduler(cfg);
        if (!used_deadline) apply_sched_fifo(cfg.priority);
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

    return errors::NIL;
}
}
