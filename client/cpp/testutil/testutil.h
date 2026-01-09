// Copyright 2026 Synnax Labs, Inc.
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

#include "client/cpp/synnax.h"

const synnax::Config test_client_config = {"localhost", 9090, "synnax", "seldon"};

/// @brief instantiates a new client for testing purposes. The cluster is expected to be
/// running on localhost:9090 in insecure mode.
extern synnax::Synnax new_test_client();

/// @brief creates a new random generator for a test suite, and outputs the seed to
/// stdout for reproducibility.
extern std::mt19937 random_generator(const std::string &suite_name);

/// @brief creates a new unique channel name for a test.
/// @param base_name the base name of the channel.
/// @return a unique channel name.
extern std::string make_unique_channel_name(const std::string &base_name);

synnax::channel::Channel create_virtual_channel(
    const synnax::Synnax &client,
    const x::telem::DataType &data_type = x::telem::FLOAT32_T
);

std::pair<synnax::channel::Channel, synnax::channel::Channel>
create_indexed_pair(synnax::Synnax &client);
