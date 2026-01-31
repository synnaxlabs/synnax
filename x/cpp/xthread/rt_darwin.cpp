// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <pthread.h>

#include <mach/mach.h>
#include <mach/thread_policy.h>

#include "glog/logging.h"

#include "x/cpp/xthread/rt.h"

namespace xthread {
bool has_rt_support() { return false; }

xerrors::Error apply_rt_config(const RTConfig &cfg) {
    const mach_port_t thread_port = pthread_mach_thread_np(pthread_self());

    if (cfg.enabled) {
        thread_precedence_policy_data_t precedence;
        precedence.importance = cfg.priority;
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
            VLOG(1) << "[xthread] Set thread precedence to " << cfg.priority;
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

    return xerrors::NIL;
}
}
