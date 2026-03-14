// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <atomic>
#include <csignal>
#include <string>

#include "glog/logging.h"

#include "gperftools/profiler.h"

namespace x::profile {
/// @brief toggles CPU profiling on/off via gperftools. Thread-safe.
/// Call install() once at startup to register a SIGUSR1 signal handler.
/// Then send SIGUSR1 to the process to start profiling, and again to stop.
/// The profile is written to the given output path.
class Profiler {
    static inline std::atomic<bool> active{false};
    static inline std::string output_path = "/tmp/driver.prof";

public:
    /// @brief installs a SIGUSR1 handler that toggles profiling.
    /// @param path the file path to write the profile to.
    static void install(const std::string &path = "/tmp/driver.prof") {
        output_path = path;
        std::signal(SIGUSR1, toggle);
        LOG(INFO) << "[profile] installed. Send SIGUSR1 to toggle CPU profiling "
                  << "(output: " << path << ")";
    }

    /// @brief starts profiling. No-op if already profiling.
    static void start() {
        if (active.exchange(true)) return;
        ProfilerStart(output_path.c_str());
        LOG(INFO) << "[profile] CPU profiling started -> " << output_path;
    }

    /// @brief stops profiling and flushes output. No-op if not profiling.
    static void stop() {
        if (!active.exchange(false)) return;
        ProfilerStop();
        LOG(INFO) << "[profile] CPU profiling stopped. Saved to " << output_path;
    }

    /// @brief returns true if profiling is currently active.
    static bool is_active() { return active.load(); }

private:
    static void toggle(int) {
        if (active.load())
            stop();
        else
            start();
    }
};
}
