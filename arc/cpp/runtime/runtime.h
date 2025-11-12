// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include <map>
#include <memory>

#include "x/cpp/queue/spsc.h"
#include "x/cpp/telem/frame.h"

#include "arc/cpp/runtime/core/types.h"
#include "arc/cpp/runtime/scheduler/scheduler.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/runtime/wasm/module.h"
#include "arc/cpp/runtime/loop/loop.h"

namespace arc::runtime {
struct Config {
    module::Module mod;
    breaker::Config breaker;
    std::function<std::vector<state::ChannelDigest, xerrors::Error>()> retrieve_channels;
};

struct Runtime {
    static constexpr size_t DEFAULT_QUEUE_CAPACITY = 1024;

    breaker::Breaker breaker;
    std::unique_ptr<wasm::Module> mod;
    std::unique_ptr<Scheduler> scheduler;
    std::unique_ptr<state::State> state;
    std::unique_ptr<loop::Loop> time_wheel;

    std::unique_ptr<queue::SPSC<telem::Frame>> inputs;
    std::unique_ptr<queue::SPSC<telem::Frame>> outputs;

    void run() {
        while (this->breaker.running()) {
            this->time_wheel->wait(this->breaker);
            this->scheduler->next();
        }
    }
};

inline std::pair<Runtime, xerrors::Error> load(Config cfg) {
    // Step 1: Initialize state
    // Step 2: Compile WASM module
    // Step 3: Time loop
    // Step 4: Queues
}

}
