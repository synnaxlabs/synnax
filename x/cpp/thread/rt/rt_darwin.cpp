// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "glog/logging.h"
#include <mach/mach.h>
#include <mach/mach_time.h>
#include <mach/thread_policy.h>
#include <pthread.h>
#include <sys/qos.h>

#include "x/cpp/thread/rt/rt.h"

namespace x::thread::rt {
namespace {
uint64_t ns_to_mach_abs(const int64_t ns) {
    static mach_timebase_info_data_t info = [] {
        mach_timebase_info_data_t i;
        mach_timebase_info(&i);
        return i;
    }();
    return static_cast<uint64_t>(ns) * info.denom / info.numer;
}

bool apply_time_constraint(
    const mach_port_t thread_port,
    const uint32_t period,
    const uint32_t computation,
    const uint32_t constraint
) {
    thread_time_constraint_policy_data_t policy;
    policy.period = period;
    policy.computation = computation;
    policy.constraint = constraint;
    policy.preemptible = TRUE;
    const kern_return_t result = thread_policy_set(
        thread_port,
        THREAD_TIME_CONSTRAINT_POLICY,
        reinterpret_cast<thread_policy_t>(&policy),
        THREAD_TIME_CONSTRAINT_POLICY_COUNT
    );
    if (result != KERN_SUCCESS) {
        LOG(WARNING) << "[xthread] Failed to set time constraint policy: "
                     << mach_error_string(result);
        return false;
    }
    VLOG(1) << "[xthread] Set time constraint: period=" << period
            << " computation=" << computation << " constraint=" << constraint;
    return true;
}

void disable_timesharing(const mach_port_t thread_port) {
    thread_standard_policy_data_t policy;
    const kern_return_t result = thread_policy_set(
        thread_port,
        THREAD_STANDARD_POLICY,
        reinterpret_cast<thread_policy_t>(&policy),
        THREAD_STANDARD_POLICY_COUNT
    );
    if (result != KERN_SUCCESS)
        LOG(WARNING) << "[xthread] Failed to disable timesharing: "
                     << mach_error_string(result);
    else
        VLOG(1) << "[xthread] Disabled timesharing";
}

void apply_qos_class() {
    if (pthread_set_qos_class_self_np(QOS_CLASS_USER_INTERACTIVE, 0) != 0)
        LOG(WARNING) << "[xthread] Failed to set QOS_CLASS_USER_INTERACTIVE";
    else
        VLOG(1) << "[xthread] Set QOS_CLASS_USER_INTERACTIVE";
}

void apply_precedence(mach_port_t thread_port, int priority) {
    thread_precedence_policy_data_t precedence;
    precedence.importance = priority;
    const kern_return_t result = thread_policy_set(
        thread_port,
        THREAD_PRECEDENCE_POLICY,
        reinterpret_cast<thread_policy_t>(&precedence),
        THREAD_PRECEDENCE_POLICY_COUNT
    );
    if (result != KERN_SUCCESS)
        LOG(WARNING) << "[xthread] Failed to set thread precedence: "
                     << mach_error_string(result);
    else
        VLOG(1) << "[xthread] Set thread precedence to " << priority;
}
}

Capabilities get_capabilities() {
    return {
        .priority_scheduling = {true, true},
        .deadline_scheduling = {false, false},
        .time_constraint = {true, true},
        .mmcss = {false, false},
        .cpu_affinity = {true, true},
        .memory_locking = {false, false},
    };
}

std::string Capabilities::permissions_guidance() const {
    return "";
}

bool has_support() {
    return get_capabilities().any();
}

errors::Error apply_config(const Config &cfg) {
    const mach_port_t thread_port = pthread_mach_thread_np(pthread_self());

    if (cfg.enabled) {
        apply_qos_class();
        disable_timesharing(thread_port);

        if (cfg.has_timing()) {
            const auto period = static_cast<uint32_t>(
                ns_to_mach_abs(cfg.period.nanoseconds())
            );
            const auto computation = static_cast<uint32_t>(
                ns_to_mach_abs(cfg.computation.nanoseconds())
            );
            const auto constraint = static_cast<uint32_t>(
                ns_to_mach_abs(cfg.deadline.nanoseconds())
            );
            if (!apply_time_constraint(thread_port, period, computation, constraint))
                apply_precedence(thread_port, cfg.priority);
        } else
            apply_precedence(thread_port, cfg.priority);
    }

    if (cfg.cpu_affinity >= 0) {
        thread_affinity_policy_data_t affinity_policy;
        affinity_policy.affinity_tag = cfg.cpu_affinity;
        const kern_return_t result = thread_policy_set(
            thread_port,
            THREAD_AFFINITY_POLICY,
            reinterpret_cast<thread_policy_t>(&affinity_policy),
            THREAD_AFFINITY_POLICY_COUNT
        );
        if (result != KERN_SUCCESS)
            LOG(WARNING) << "[xthread] Failed to set CPU affinity to "
                         << cfg.cpu_affinity << ": " << mach_error_string(result);
        else
            VLOG(1) << "[xthread] Set thread affinity tag to " << cfg.cpu_affinity;
    }

    if (cfg.lock_memory)
        LOG(WARNING) << "[xthread] Memory locking not fully supported on macOS";

    return errors::NIL;
}
}
