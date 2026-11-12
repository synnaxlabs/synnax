// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "absl/log/log.h"

#include "client/cpp/errors/errors.h"
#include "x/cpp/thread/rt/rt.h"
#include "x/cpp/thread/thread.h"

#include "rack.h"

namespace driver::rack {
bool Rack::should_exit(
    const x::errors::Error &err,
    const std::function<void()> &on_shutdown
) {
    if (!err) return false;
    if (synnax::errors::is_temporarily_unavailable(err) && this->breaker.wait(err))
        return false;
    // A stop came first, so the error did not stop the rack.
    if (!this->breaker.running()) return true;
    this->run_err = err;
    if (on_shutdown) on_shutdown();
    return true;
}

Rack::~Rack() {
    stop();
}

void Rack::run(x::args::Parser &args, const std::function<void()> &on_shutdown) {
    x::thread::set_name("rack");
    LOG(INFO) << x::thread::rt::capabilities();
    while (this->breaker.running()) {
        auto [cfg, err] = Config::load(args, this->breaker);
        if (err) {
            if (this->should_exit(err, on_shutdown)) return;
            continue;
        }
        VLOG(1) << "loaded config. starting task manager";
        if (!this->breaker.running()) return;
        auto rt_manager = std::make_shared<x::thread::rt::Manager>();
        const auto client = cfg.new_client();
        this->task_manager = std::make_unique<task::Manager>(
            cfg.rack,
            client,
            cfg.new_factory(client, rt_manager),
            cfg.manager
        );
        err = this->task_manager->run([this]() { this->breaker.reset(); });
        if (err && this->should_exit(err, on_shutdown)) return;
    }
    if (this->task_manager != nullptr) this->task_manager->stop();
}

void Rack::start(x::args::Parser &args, std::function<void()> on_shutdown) {
    this->breaker.start();
    this->run_thread = std::thread(
        [this, args, callback = std::move(on_shutdown)]() mutable {
            this->run(args, callback);
        }
    );
}

x::errors::Error Rack::stop() {
    if (!this->breaker.stop()) return x::errors::NIL;
    if (this->task_manager != nullptr) this->task_manager->stop();
    this->run_thread.join();
    return this->run_err;
}
}
