// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "glog/logging.h"

#include "x/cpp/xthread/rt.h"

namespace xthread {
bool has_rt_support() { return false; }

xerrors::Error apply_rt_config(const RTConfig &cfg) {
    if (cfg.enabled)
        LOG(WARNING) << "[xthread] Real-time scheduling not supported on this platform";
    if (cfg.cpu_affinity >= 0)
        LOG(WARNING) << "[xthread] CPU affinity not supported on this platform";
    if (cfg.lock_memory)
        LOG(WARNING) << "[xthread] Memory locking not supported on this platform";
    return xerrors::NIL;
}
}
