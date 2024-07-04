// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std
#include <random>

//// internal
#include "driver/testutil/testutil.h"


synnax::Synnax new_test_client() {
    return synnax::Synnax(test_client_config);
}

std::mt19937 random_generator(std::string suite_name) {
    std::random_device rd;
    auto rand_seed = rd();
    std::cout << "Random seed for " << suite_name << " - " << rand_seed << std::endl;
    std::mt19937 mt(rand_seed);
    std::uniform_real_distribution<double> dist(0, 1);
    return mt;
}



json add_DI_channel_JSON(
    json &config,
    std::string name,
    int key,
    int port,
    int line
) {
    // first construct the json object for the channel
    json channel;
    channel["name"] = name;
    channel["channel_type"] = "digitalInput";
    channel["port"] = port;
    channel["line"] = line;
    channel["channel"] = key;

    // now add json to the channels vector
    // check if the channels array exists
    if (config.find("channels") == config.end()) {
        config["channels"] = json::array();
    }
    config["channels"].push_back(channel);
    return channel;
}

json add_AI_channel_JSON(
    json &config,
    std::string name,
    int key,
    int port,
    std::float_t min_val,
    std::float_t max_val,
    std::string terminal_config,
    json scale_config
) {
    // first construct the json object for the channel
    json channel;
    channel["name"] = name;
    channel["type"] = "ai_voltage";
    channel["port"] = port;
    channel["channel"] = key;
    channel["min_val"] = min_val;
    channel["max_val"] = max_val;
    channel["terminal_config"] = terminal_config;
    channel["units"] = "Volts";
    channel["enabled"] = true;
    channel["key"] = "key";
    channel["custom_scale"] = scale_config;
    channel["enabled"] = true;
    
    // now add json to the channels vector
    // check if the channels array exists
    if (config.find("channels") == config.end()) {
        config["channels"] = json::array();
    }
    config["channels"].push_back(channel);
    return channel;
}


json add_DO_channel_JSON(
    json &config,
    std::string name,
    int drive_cmd_key,
    int state_key,
    int port,
    int line
) {
    // first construct the json object for the channel
    json channel;
    channel["name"] = name;
    channel["channel_key"] = drive_cmd_key;
    channel["state_key"] = state_key;
    channel["channel_type"] = "digitalOutput";
    channel["port"] = port;
    channel["line"] = line;

    // now add json to the channels vector
    // check if the channels array exists
    if (config.find("channels") == config.end()) {
        config["channels"] = json::array();
    }
    config["channels"].push_back(channel);
    return channel;
}
