// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <latch>
#include "driver/rack/rack.h"

device::Modules::Modules(
        synnax::RackKey rack_key,
        const std::shared_ptr<synnax::Synnax> &client,
        std::unique_ptr<module::Factory> factory,
        breaker::Breaker breaker
) :
        rack_key(rack_key),
        client(client),
        factory(std::move(factory)),
        running(false),
        exit_err(freighter::NIL),
        breaker(breaker),
        internal(rack_key, "") {
}

const std::string MODULE_SET_CHANNEL = "sy_module_cfg";
const std::string MODULE_DELETE_CHANNEL = "sy_module_delete";

freighter::Error device::Modules::start(std::latch &latch) {
    auto err = startInternal();
    if (err) {
        if (err.type == freighter::TYPE_UNREACHABLE && breaker.wait()) start(latch);
        latch.count_down();
        return err;
    }
    breaker.reset();
    running = true;
    exec_thread = std::thread(&Modules::run, this, std::ref(latch));
    return freighter::NIL;
}

freighter::Error device::Modules::startInternal() {
    auto [rack, rack_err] = client->devices.retrieveRack(rack_key.value);
    if (rack_err) return rack_err;

    // Fetch relevant channels.
    auto [mod_set, mod_set_err] = client->channels.retrieve(MODULE_SET_CHANNEL);
    if (mod_set_err) return mod_set_err;
    module_set_channel = mod_set;

    auto [mod_del, mod_del_err] = client->channels.retrieve(MODULE_DELETE_CHANNEL);
    if (mod_del_err) return mod_del_err;
    module_delete_channel = mod_del;
}


void device::Modules::run(std::latch &latch) {
    auto err = runInternal();
    if (err) {
        // This is the only error type that we retry on.
        if (err.type == freighter::TYPE_UNREACHABLE && breaker.wait()) runInternal();
        exit_err = err;
    }
}

freighter::Error device::Modules::runInternal() {
    // Open the streamer.
    std::vector<synnax::ChannelKey> channels = {module_set_channel.key, module_delete_channel.key};
    auto [streamer, open_err] = client->telem.openStreamer(synnax::StreamerConfig{.channels = channels});
    if (open_err) {
        return open_err;
    }

    // If we pass here it means we've re-gained network connectivity and can reset the breaker.
    breaker.reset();

    while (running) {
        auto [frame, read_err] = streamer.read();
        if (read_err) {
            return read_err;
        }
        for (size_t i = 0; i < frame.size(); i++) {
            auto &key = (*frame.columns)[i];
            auto &series = (*frame.series)[i];
            if (key == module_set_channel.key)
                processModuleSet(series);
            else if (key == module_delete_channel.key)
                processModuleDelete(series);
        }
    }
}

void device::Modules::processModuleSet(const synnax::Series &series) {
    auto keys = series.uint64();
    for (auto key: keys) {
        auto it = modules.find(key);
        if (it != modules.end()) {
            it->second->stop();
            modules.erase(it);
        }
        auto [mod_config, err] = internal.modules.retrieve(key);
        if (err) {
            std::cerr << err.message() << std::endl;
            continue;
        }
        auto [driver_mod, config_err] = factory->configure(client, mod_config);
        if (err) {
            std::cerr << config_err.message() << std::endl;
            continue;
        }
        modules[key] = std::move(driver_mod);
    }
}

void device::Modules::processModuleDelete(const synnax::Series &series) {
    auto keys = series.uint64();
    for (auto key: keys) {
        auto it = modules.find(key);
        if (it != modules.end()) {
            it->second->stop();
            modules.erase(it);
        }
    }
}