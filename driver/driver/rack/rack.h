#include <string>
#include <unordered_map>
#include <thread>

#include "synnax/synnax.h"
#include "driver/modules/module.h"
#include "driver/breaker/breaker.h"

#pragma once

namespace device {


class Modules {
public:
    Modules(
            synnax::RackKey rack_key,
            const std::shared_ptr<synnax::Synnax>& client,
            std::unique_ptr<module::Factory> factory
    );

    freighter::Error start();

    void stop();

    void processModuleSet(const synnax::Series &series);
    void processModuleDelete(const synnax::Series &series);
private:
    synnax::RackKey rack_key;
    synnax::Rack internal;

    const std::shared_ptr<synnax::Synnax> client;
    std::unique_ptr<module::Factory> factory;

    std::unordered_map<std::uint64_t, std::unique_ptr<module::Module>> modules;

    synnax::Channel module_set_channel;
    synnax::Channel module_delete_channel;

    std::atomic<bool> running;
    std::thread exec_thread;
    freighter::Error exit_err;
    breaker::Breaker breaker;

    void run();
};

class Heartbeat {
public:
    Heartbeat(
        synnax::RackKey rack_key,
        std::uint32_t generation,
        const std::shared_ptr<synnax::Synnax> &client,
    );

    freighter::Error start();

    void stop();

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
            const std::shared_ptr<synnax::Synnax>& client,
            std::unique_ptr<module::Factory> module_factory
    );

    freighter::Error run();
private:
    Modules modules;
    Heartbeat heartbeat;
};
}