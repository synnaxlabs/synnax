// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <random>
#include <string>
#include "nlohmann/json.hpp"
#include "client/cpp/synnax.h"

using json = nlohmann::json;

const synnax::Config test_client_config = {
    "localhost",
    9090,
    "synnax",
    "seldon"
};

/// @brief instantiates a new client for testing purposes. The cluster
/// is expected to be running on localhost:9090 in insecure mode.
inline extern std::shared_ptr<synnax::Synnax> new_test_client() {
    return std::make_shared<synnax::Synnax>(test_client_config);
}

/// @brief creates a new random generator for a test suite, and
/// outputs the seed to stdout for reproducibility.
extern std::mt19937 random_generator(std::string suite_name) {
    std::random_device rd;
    auto rand_seed = rd();
    std::cout << "Random seed for " << suite_name << " - " << rand_seed << std::endl;
    std::mt19937 mt(rand_seed);
    std::uniform_real_distribution<double> dist(0, 1);
    return mt;
}
