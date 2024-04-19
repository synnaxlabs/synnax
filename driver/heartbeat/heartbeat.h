// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "freighter/freighter.h"

#pragma once

namespace Heartbeat {
class Heartbeat {
public:
    Heartbeat(
        RackKey rack_key,
        std::shared_ptr<Synnax> client,
        breaker::Config breaker_config
    );

    freighter::Error start(std::atomic<bool> &done);

    freighter::Error stop();

private:
    // Synnax
    RackKey rack_key;
    const std::shared_ptr<Synnax> client;

    Channel channel;

    // Heartbeat
    std::uint32_t version;

    // Breaker
    breaker::Breaker breaker;

    // Threading
    std::atomic<bool> running;
    std::thread run_thread;
    freighter::Error run_err;

    void run(std::atomic<bool> &done);

    freighter::Error runGuarded();

    freighter::Error startGuarded();
};
}
