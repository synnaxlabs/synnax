// Copyright 2023 Synnax Labs, Inc.
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
#include "synnax/synnax.h"
#include "driver/testutil/testutil.h"

synnax::Synnax new_test_client()
{
    return synnax::Synnax(test_client_config);
}

std::mt19937 random_generator(std::string suite_name)
{
    std::random_device rd;
    auto rand_seed = rd();
    std::cout << "Random seed for " << suite_name << " - " << rand_seed << std::endl;
    std::mt19937 mt(rand_seed);
    std::uniform_real_distribution<double> dist(0, 1);
    return mt;
}

extern json add_DO_channel_JSON(&json config,
                                std::string name,
                                uint32_t cmd_key,
                                uint32_t ack_key,
                                uint32_t port,
                                uint32_t line){

    // first construct the json object for the channel
    json channel;
    channel["name"] = name;
    channel["channel_key"] = cmd_key;
    channel["ack_key"] = ack_key;
    channel["channelType"] = ni::DIGITAL_OUT;
    channel["port"] = port;
    channel["line"] = line;

    // now add json to the channels vector
    // check if the channels array exists
    if(config.find("channels") == config.end()){
        config["channels"] = json::array();
    }
    config["channels"].push_back(channel);
    return  channel;
}