// Copyright 2025 Synnax Labs, Inc.
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
#include "client/cpp/testutil/testutil.h"


synnax::Synnax new_test_client() {
    return synnax::Synnax(test_client_config);
}

std::mt19937 random_generator(const std::string &suite_name) {
    std::random_device rd;
    const auto rand_seed = rd();
    std::cout << "Random seed for " << suite_name << " - " << rand_seed << "\n";
    const std::mt19937 mt(rand_seed);
    return mt;
}
