// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "rack.h"

bool rack::Rack::should_exit(const xerrors::Error &err) {
    this->run_err = err;
    if (err) {
        if (err.matches(freighter::UNREACHABLE) && breaker.wait(err))
            return false;
        return true;
    }
    return false;
}

void rack::Rack::run(const int argc, char **argv) {
    while (this->breaker.running()) {
        auto [cfg, err] = Config::load(argc, argv, this->breaker);
        if (err) {
            if (this->should_exit(err)) return;
            continue;
        }
        LOG(INFO) << cfg;
        this->task_manager = std::make_unique<task::Manager>(
            cfg.rack,
            cfg.new_client(),
            cfg.new_factory()
        );
        err = this->task_manager->run();
        if (err && this->should_exit(err)) return;
    }
    this->run_err = xerrors::NIL;
}

void rack::Rack::start(int argc, char **argv) {
    this->breaker.start();
    this->run_thread = std::thread([this, argv, argc] {
        this->run(argc, argv);
    });
}

xerrors::Error rack::Rack::stop() {
    if (!this->breaker.running()) return xerrors::NIL;
    this->breaker.stop();
    if (this->task_manager != nullptr) this->task_manager->stop();
    this->run_thread.join();
    return this->run_err;
}
