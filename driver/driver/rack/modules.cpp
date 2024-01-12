// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// Std.
#include <latch>
#include <utility>

/// External.
#include "nlohmann/json.hpp"

/// Internal.
#include "driver/rack/rack.h"

using json = nlohmann::json;

device::Modules::Modules(
        synnax::RackKey rack_key,
        const std::shared_ptr<synnax::Synnax> &client,
        std::unique_ptr<module::Factory> factory,
        breaker::Breaker breaker
) :
        rack_key(rack_key),
        client(client),
        factory(std::move(factory)),
        exit_err(freighter::NIL),
        breaker(std::move(breaker)),
        internal(rack_key, "") {
}

const std::string MODULE_SET_CHANNEL = "sy_module_set";
const std::string MODULE_DELETE_CHANNEL = "sy_module_delete";

freighter::Error device::Modules::start(std::latch &latch) {
    auto err = startInternal();
    if (err) {
        if (err.type == freighter::TYPE_UNREACHABLE && breaker.wait()) start(latch);
        latch.count_down();
        return err;
    }
    breaker.reset();
    exec_thread = std::thread(&Modules::run, this, std::ref(latch));
    return freighter::NIL;
}

freighter::Error device::Modules::startInternal() {
    auto [rack, rack_err] = client->devices.retrieveRack(rack_key.value);
    if (rack_err) return rack_err;
    internal = rack;

    // Fetch module set channel.
    auto [mod_set, mod_set_err] = client->channels.retrieve(MODULE_SET_CHANNEL);
    if (mod_set_err) return mod_set_err;
    module_set_channel = mod_set;

    // Fetch module delete channel.
    auto [mod_del, mod_del_err] = client->channels.retrieve(MODULE_DELETE_CHANNEL);
    if (mod_del_err) return mod_del_err;
    module_delete_channel = mod_del;

    // Fetch module comms channel.
    auto [mod_comms, mod_comms_err] = client->channels.retrieve(
            "sy_node_" + std::to_string(rack.key.node_key()) + "_comms");
    if (mod_comms_err) return mod_comms_err;
    module_comms_channel = mod_comms;
    return freighter::NIL;
}


void device::Modules::run(std::latch &latch) {
    auto err = runInternal();
    if (err) {
        // This is the only error type that we retry on.
        if (err == freighter::TYPE_UNREACHABLE && breaker.wait()) runInternal();
        exit_err = err;
    }
    latch.count_down();
}

freighter::Error device::Modules::stop() {
    streamer->closeSend();
    exec_thread.join();
    return exit_err;
}

freighter::Error device::Modules::runInternal() {
    // Open the streamer.
    std::vector<synnax::ChannelKey> stream_channels = {module_set_channel.key, module_delete_channel.key};
    auto [s, open_err] = client->telem.openStreamer(synnax::StreamerConfig{.channels = stream_channels});
    if (open_err) return open_err;
    streamer = std::make_unique<synnax::Streamer>(std::move(s));

    // Open the writer.
    std::vector<synnax::ChannelKey> write_channels = {module_comms_channel.key};
    auto [writer, writer_err] = client->telem.openWriter(synnax::WriterConfig{.channels = write_channels});
    if (writer_err) return writer_err;

    // If we pass here it means we've re-gained network connectivity and can reset the breaker.
    breaker.reset();

    while (true) {
        auto [frame, read_err] = streamer->read();
        if (read_err) return read_err;
        for (size_t i = 0; i < frame.size(); i++) {
            auto &key = (*frame.columns)[i];
            auto &series = (*frame.series)[i];
            if (key == module_set_channel.key) processModuleSet(series, writer);
            else if (key == module_delete_channel.key) processModuleDelete(series);
        }
    }
    return freighter::NIL;
}

void device::Modules::processModuleSet(const synnax::Series &series, synnax::Writer &comms) {
    auto keys = series.uint64();
    for (auto key: keys) {
        // If a module exists with this key, stop and remove it.
        auto mod_it = modules.find(key);
        if (mod_it != modules.end()) {
            mod_it->second->stop();
            modules.erase(mod_it);
        }
        auto [mod_config, err] = internal.modules.retrieve(key);
        if (err) {
            std::cerr << err.message() << std::endl;
            continue;
        }
        json config_err;
        bool valid_config = true;
        auto driver_mod = factory->configure(client, mod_config, valid_config, config_err);
        if (!valid_config) {
            json config_err_pld;
            config_err_pld["type"] = "config_error";
            config_err_pld["error"] = config_err;
            config_err_pld["module"] = mod_config.key.value;
            auto fr = synnax::Frame(1);
            fr.add(module_comms_channel.key, synnax::Series(std::vector<std::string>{to_string(config_err_pld)}, synnax::JSON));
            comms.write(std::move(fr));
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