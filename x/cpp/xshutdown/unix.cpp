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

#include <mutex>
#include <string>
#include <poll.h>
#include <unistd.h>
#include "xshutdown.h"
#include <signal.h>

namespace xshutdown {

static Listen* active_listener = nullptr;

static void signal_handler(int signal) {
    if (signal == SIGINT && active_listener != nullptr) {
        active_listener->signal_shutdown();
    }
}

void Listen::listen_signal() {
    active_listener = this;
    signal(SIGINT, signal_handler);
}

void Listen::listen_stdin() {
    struct pollfd fds[1];
    fds[0].fd = STDIN_FILENO;
    fds[0].events = POLLIN;
    
    char buffer[256];
    std::string input;
    
    while (true) {
        // Poll with a timeout of 100ms
        int ret = poll(fds, 1, 100);
        
        if (ret < 0) {
            if (errno == EINTR) continue;  // Interrupted by signal, retry
            break;  // Error occurred
        }
        
        // Check if we should stop
        if (should_shutdown()) break;
        
        // If there's data to read
        if (ret > 0 && (fds[0].revents & POLLIN)) {
            ssize_t n = read(STDIN_FILENO, buffer, sizeof(buffer) - 1);
            if (n <= 0) break;
            
            buffer[n] = '\0';
            input += buffer;
            
            // Process complete lines
            size_t pos;
            while ((pos = input.find('\n')) != std::string::npos) {
                std::string line = input.substr(0, pos);
                input.erase(0, pos + 1);
                
                if (line == "STOP") {
                    signal_shutdown();
                    return;
                }
            }
        }
    }
}

void Listen::listen() {
    Listen listener;
    listener.listen_signal();
    listener.listen_stdin();
}

bool Listen::should_shutdown() const {
    std::lock_guard lock(mu);
    return should_stop;
}

void Listen::signal_shutdown() {
    {
        std::lock_guard lock(mu);
        should_stop = true;
    }
    cv.notify_all();
}

} // namespace shutdown
