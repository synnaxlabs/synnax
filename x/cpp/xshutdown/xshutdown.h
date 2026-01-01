// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <condition_variable>
#include <mutex>

/// @brief xshutdown implements a utility for listening to various shutdown signals
/// in order to gracefully exit a program. By default, it listens to SIGINT, SIGTERM,
/// and for the user to type STOP into stdin. These can be enabled or disabled as
/// needed.
namespace xshutdown {
/// @brief internal namespace. do not use directly.
namespace priv {
// These need to be declared as extern and defined in os-specific cpp files
// so that we don't cause internal linkage issues.
extern std::mutex shutdown_mutex;
extern std::condition_variable shutdown_cv;
extern bool should_stop;

/// @brief platform specific implementation that registers relevant signal
/// handlers.
void listen_signal();

/// @brief platform specific implementation that listens for stdin input. This
/// function should block as polls inputs until the user types STOP or the
/// should_shutdown function returns true.
void listen_stdin();

/// @brief returns true if the shutdown condition has been signaled.
inline bool should_shutdown() {
    std::lock_guard lock(priv::shutdown_mutex);
    return priv::should_stop;
}
}

/// @brief signals the shutdown condition to all listeners.
inline void signal_shutdown() {
    {
        std::lock_guard lock(priv::shutdown_mutex);
        priv::should_stop = true;
    }
    priv::shutdown_cv.notify_all();
}

/// @brief listens for shutdown signals from SIGINT, SIGTERM, and stdin.
/// @param sig_enabled whether to listen for SIGINT and SIGTERM signals. Default is
/// true.
/// @param stdin_enabled whether to listen for stdin input. Default is true.
inline void listen(const bool sig_enabled = true, const bool stdin_enabled = true) {
    if (sig_enabled) priv::listen_signal();
    if (stdin_enabled) return priv::listen_stdin();
    std::unique_lock lock(priv::shutdown_mutex);
    priv::shutdown_cv.wait(lock, [] { return priv::should_stop; });
}
}
