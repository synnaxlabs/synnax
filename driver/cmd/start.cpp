// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std
#include <iostream>
#include <thread>
#include <condition_variable>

/// external
#include "glog/logging.h"

/// internal
#include "cmd.h"
#include "driver/rack/rack.h"

const std::string STOP_COMMAND = "STOP";

std::mutex mtx;
std::condition_variable cv;
bool should_stop = false;

void input_listener() {
    std::string input;
    while (std::getline(std::cin, input)) {
        if (input == STOP_COMMAND) {
            {
                std::lock_guard lock(mtx);
                should_stop = true;
            }
            cv.notify_one();
            break;
        }
    }
}

int cmd::sub::start(int argc, char *argv[]) {
    std::thread listener(input_listener);
    rack::Rack r;
    r.start(argc, argv);
    {
        std::unique_lock lock(mtx);
        cv.wait(lock, [] { return should_stop; });
    }
    r.stop();
    listener.join();
    return 0;
}
