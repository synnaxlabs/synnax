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

class Listen {
private:
    mutable std::mutex mu;
    std::condition_variable cv;
    bool should_stop = false;

    // Platform specific implementations
    void listen_signal();
    void listen_stdin();

public:
    // Default constructor needed
    Listen() : should_stop(false) {}
    
    // Main entry point that starts both signal and stdin listeners
    static void listen();
    
    // Check if we should stop
    bool should_shutdown() const;
    
    // Signal shutdown
    void signal_shutdown();
};

} // namespace shutdown
