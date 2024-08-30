// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <random>
#include <string>
#include "nlohmann/json.hpp"
//// internal
#include "client/cpp/synnax.h"

using json = nlohmann::json;

const synnax::Config test_client_config = {
        "localhost",
        9090,
        "synnax",
        "seldon"};

/// @brief instantiates a new client for testing purposes. The cluster
/// is expected to be running on localhost:9090 in insecure mode.
extern synnax::Synnax new_test_client();

/// @brief creates a new random generator for a test suite, and
/// outputs the seed to stdout for reproducibility.
extern std::mt19937 random_generator(std::string suite_name);

/// @brief adds a digital output channel to a json object passed by ref
/// returns just the JSON of the channel constructed
json add_DO_channel_JSON(json &config,
                         std::string name,
                         int drive_cmd_key,
                         int state_key,
                         int port,
                         int line);

json add_index_channel_JSON(json &config,
                            std::string name,
                            int key);

json add_state_index_channel_JSON(json &config,
                                  std::string name,
                                  int key);

/// @brief adds a  digital input channel to a json object passed by ref
/// returns just the JSON of the channel constructed
json add_DI_channel_JSON(json &config,
                         std::string name,
                         int key,
                         int port,
                         int line);


/// @brief adds a analog input channel to a json object passed by ref
/// returns just the JSON of the channel constructed
json add_AI_channel_JSON(json &config,
                         std::string name,
                         int key,
                         int port,
                         std::float_t min_val,
                         std::float_t max_val,
                         std::string terminal_config,
                         json scale_config); 