// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <csignal>
#include <mutex>
#include <string>

#include <poll.h>
#include <unistd.h>

#include "x/cpp/shutdown/shutdown.h"

namespace x::shutdown::priv {
std::mutex shutdown_mutex;
std::condition_variable shutdown_cv;
bool should_stop = false;

void signal_handler(const int signal) {
    if (signal == SIGINT || signal == SIGTERM) signal_shutdown();
}

void listen_signal() {
    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);
}

void listen_stdin() {
    pollfd fds[1];
    fds[0].fd = STDIN_FILENO;
    fds[0].events = POLLIN;

    char buffer[256];
    std::string input;

    while (true) {
        const int ret = poll(fds, 1, 100);
        if (ret < 0) {
            if (errno == EINTR) continue;
            break;
        }
        if (should_shutdown()) break;
        if (ret > 0 && fds[0].revents & POLLIN) {
            const ssize_t n = read(STDIN_FILENO, buffer, sizeof(buffer) - 1);
            if (n <= 0) break;

            buffer[n] = '\0';
            input += buffer;

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
}
