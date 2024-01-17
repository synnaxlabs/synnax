// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <unordered_map>
#include <thread>
#include <latch>

#include "synnax/synnax.h"
#include "driver/modules/module.h"
#include "driver/breaker/breaker.h"

#pragma once

namespace device {


class Modules {
public:
    [[maybe_unused]] Modules(
            synnax::RackKey rack_key,
            const std::shared_ptr<synnax::Synnax> &client,
            std::unique_ptr<module::Factory> factory,
            breaker::Breaker breaker
    );

    freighter::Error start(std::latch &latch);

    freighter::Error stop();

    void processModuleSet(const synnax::Series &series, synnax::Writer &comms);

    void processModuleDelete(const synnax::Series &series);

private:
    synnax::RackKey rack_key;
    synnax::Rack internal;

    const std::shared_ptr<synnax::Synnax> client;
    std::unique_ptr<module::Factory> factory;
    std::unique_ptr<synnax::Streamer> streamer;


    std::unordered_map<std::uint64_t, std::unique_ptr<module::Module>> modules;

    synnax::Channel module_set_channel;
    synnax::Channel module_delete_channel;
    synnax::Channel module_comms_channel;

    std::thread exec_thread;
    freighter::Error exit_err;
    breaker::Breaker breaker;

    void run(std::latch &latch);

    freighter::Error runInternal();

    freighter::Error startInternal();

};

class Heartbeat {
public:
    Heartbeat(
            synnax::RackKey rack_key,
            std::uint32_t generation,
            std::shared_ptr<synnax::Synnax> client,
            breaker::Breaker breaker
    );

    freighter::Error start(std::latch &latch);

    freighter::Error stop();

private:
    // Synnax
    synnax::RackKey rack_key;
    const std::shared_ptr<synnax::Synnax> client;
    synnax::Channel rack_heartbeat_channel;

    // Heartbeat
    std::uint32_t generation;
    std::uint32_t version;

    // Breaker
    breaker::Breaker breaker;

    // Threading
    std::atomic<bool> running;
    freighter::Error exit_err;
    std::thread exec_thread;

    void run();
};

class Rack {
public:
    Rack(
            synnax::RackKey key,
            std::uint32_t generation,
            const std::shared_ptr<synnax::Synnax> &client,
            std::unique_ptr<module::Factory> module_factory,
            breaker::Breaker breaker
    );

    freighter::Error run();

private:
    synnax::RackKey key;
    std::uint32_t generation;
    Modules modules;
    Heartbeat heartbeat;
};
}