// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//
// Created by Emiliano Bonilla on 2/11/25.
//

#pragma once

#include <condition_variable>
#include <mutex>
#include <string>

namespace xshutdown {
namespace priv {
    static std::mutex shutdown_mutex;
    static std::condition_variable shutdown_cv;
    static bool should_stop = false;

    void listen_signal();

    void listen_stdin();
}


inline void listen(bool sigint_enabled = true, bool stdin_enabled = true) {
    if (sigint_enabled) priv::listen_signal();
    if (stdin_enabled) return priv::listen_stdin();
    std::unique_lock lock(priv::shutdown_mutex);
    priv::shutdown_cv.wait(lock, [] { return priv::should_stop; });
}

inline bool should_shutdown() {
    std::lock_guard lock(priv::shutdown_mutex);
    return priv::should_stop;
}

inline void signal_shutdown() {
    {
        std::lock_guard lock(priv::shutdown_mutex);
        priv::should_stop = true;
    }
    priv::shutdown_cv.notify_all();
}
} // namespace xshutdown
