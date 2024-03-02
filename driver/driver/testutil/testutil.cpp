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

json add_DO_channel_JSON(json &config,
                                std::string name,
                                int cmd_key,
                                int ack_key,
                                int port,
                                int line){

    // first construct the json object for the channel
    json channel;
    channel["name"] = name;
    channel["cmd_key"] = cmd_key;
    channel["ack_key"] = ack_key;
    channel["type"] = "digitalOutput";
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