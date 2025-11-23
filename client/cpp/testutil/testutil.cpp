// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <random>

//// internal
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/xtest/xtest.h"

synnax::Synnax new_test_client() {
    return synnax::Synnax(test_client_config);
}

std::mt19937 random_generator(const std::string &suite_name) {
    std::random_device rd;
    const auto rand_seed = rd();
    std::cout << "Random seed for " << suite_name << " - " << rand_seed << std::endl;
    const std::mt19937 mt(rand_seed);
    return mt;
}

std::string make_unique_channel_name(const std::string& base_name) {
    static std::random_device rd;
    static std::mt19937 gen(rd());
    static std::uniform_int_distribution<> dis(1, 99999999);
    return base_name + "_" + std::to_string(dis(gen));
}

synnax::Channel
create_virtual_channel(const synnax::Synnax &client, const telem::DataType &data_type) {
    auto [ch, err] = client.channels.create(
        make_unique_channel_name("virtual"),
        data_type,
        true
    );
    return ch;
}

std::pair<synnax::Channel, synnax::Channel>
create_indexed_pair(synnax::Synnax &client) {
    auto [idx, err_one] = client.channels.create(
        make_unique_channel_name("index"),
        telem::TIMESTAMP_T,
        0,
        true
    );
    auto [data, err_two] = client.channels.create(
        make_unique_channel_name("data"),
        telem::FLOAT32_T,
        idx.key,
        false
    );
    return {idx, data};
}
