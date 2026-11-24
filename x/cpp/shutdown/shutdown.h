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

/// @brief shutdown implements a utility for listening to various shutdown signals
/// in order to gracefully exit a program. By default, it listens to SIGINT, SIGTERM,
/// and for the user to type STOP into stdin. These can be enabled or disabled as
/// needed.
namespace x::shutdown {
/// @brief internal namespace. do not use directly.
namespace details {
// These need to be declared as extern and defined in os-specific cpp files
// so that we don't cause internal linkage issues.
// NOLINTBEGIN(cppcoreguidelines-avoid-non-const-global-variables): signal handlers can
// only reach global state.
extern std::mutex shutdown_mutex;
extern std::condition_variable shutdown_cv;
extern bool should_stop;
// NOLINTEND(cppcoreguidelines-avoid-non-const-global-variables)

/// @brief platform specific implementation that registers relevant signal
/// handlers.
void listen_signal();

/// @brief platform specific implementation that listens for stdin input. This
/// function should block as polls inputs until the user types STOP or the
/// should_shutdown function returns true.
void listen_stdin();

/// @brief returns true if the shutdown condition has been signaled.
inline bool should_shutdown() {
    std::lock_guard lock(shutdown_mutex);
    return should_stop;
}
}

/// @brief signals the shutdown condition to all listeners.
inline void signal_shutdown() {
    {
        std::lock_guard lock(details::shutdown_mutex);
        details::should_stop = true;
    }
    details::shutdown_cv.notify_all();
}

/// @brief options for listening to shutdown signals.
struct ListenOptions {
    /// @brief when true, SIGINT and SIGTERM do not signal a shutdown.
    bool signals_ignored = false;
    /// @brief when true, typing STOP into stdin does not signal a shutdown.
    bool stdin_ignored = false;
};

/// @brief blocks until a shutdown is signaled by SIGINT, SIGTERM, stdin, or
/// signal_shutdown. On Windows, a broken stdin pipe ends the process at once.
/// @param options the sources of shutdown signals to ignore.
inline void listen(const ListenOptions &options = {}) {
    if (!options.signals_ignored) details::listen_signal();
    if (!options.stdin_ignored) {
        details::listen_stdin();
        return;
    }
    std::unique_lock lock(details::shutdown_mutex);
    details::shutdown_cv.wait(lock, [] { return details::should_stop; });
}
}
