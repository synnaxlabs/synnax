//
// Created by Emiliano Bonilla on 1/5/24.
//

#include <fstream>
#include <vector>
#include "nlohmann/json.hpp"
#include "driver/rack/rack.h"

using json = nlohmann::json;

static const std::string CONFIG_PATH = "~/.synnax/config.json";

device::Rack::Rack(
    synnax::RackKey key,
    std::uint32_t generation,
    const std::shared_ptr<synnax::Synnax>& client,
    std::unique_ptr<module::Factory> factory
): modules(key, client, std::move(factory)), heartbeat(key, generation, client) {
}

freighter::Error device::Rack::run() {
    auto err = modules.start();
    err = heartbeat.start();
    // TODO: how to detect thread exit.
}