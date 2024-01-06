//
// Created by Emiliano Bonilla on 1/6/24.
//

#include "driver/rack/rack.h"

device::Modules::Modules(synnax::RackKey rack_key, const std::shared_ptr<synnax::Synnax> &client,
                         std::unique_ptr<module::Factory> factory) :
        rack_key(rack_key),
        client(client),
        factory(std::move(factory)),
        running(false),
        exit_err(freighter::NIL) {
}

const std::string MODULE_CONFIGURE_CHANNEL = "sy_module_cfg";
const std::string MODULE_DELETE_CHANNEL = "sy_module_delete";

freighter::Error device::Modules::start() {
    // Fetch the rack
    auto [rack, rack_err] = client->devices.retrieveRack(rack_key.value);
    if (rack_err) return rack_err;

    // Fetch relevant channels.
    auto [mod_cfg, mod_cfg_err] = client->channels.retrieve(MODULE_CONFIGURE_CHANNEL);
    if (mod_cfg_err) return mod_cfg_err;
    module_set_channel = mod_cfg;

    auto [mod_del, mod_del_err] = client->channels.retrieve(MODULE_DELETE_CHANNEL);
    if (mod_del_err) return mod_del_err;
    module_delete_channel = mod_del;

    // Grab all initial modules for the rack.
    auto [mod_list, mod_list_err] = rack.modules.list();
    if (mod_list_err) return mod_list_err;

    // Use the factory to configure the modules.
    for (auto &mod : mod_list) {
        modules[mod.key.value] = factory->configure(client, mod);
    }

    running = true;
    exec_thread = std::thread(&Modules::run, this);
    return freighter::NIL;
}

void device::Modules::run() {
    // Open the streamer.
    std::vector<synnax::ChannelKey> channels = {module_set_channel.key, module_delete_channel.key};
    auto [streamer, err] = client->telem.openStreamer(synnax::StreamerConfig{.channels = channels});
    if (err) {
        if (err.type == freighter::TYPE_UNREACHABLE && breaker.wait()) run();
        exit_err = err;
        return;
    }

    while (running) {
        auto [frame, err] = streamer.read();
        if (err) {
            if (err.type == freighter::TYPE_UNREACHABLE && breaker.wait()) run();
            exit_err = err;
            break;
        }

        for (size_t i = 0; i < frame.size(); i++)
        {
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
    for (auto key : keys) {
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
    for (auto key : keys) {
        auto it = modules.find(key);
        if (it != modules.end()) {
            it->second->stop();
            modules.erase(it);
        }
    }
}