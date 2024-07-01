// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "nlohmann/json.hpp"
#include "client/cpp/synnax.h"
#include "driver/breaker/breaker.h"
#include "task/task.h"

using json = nlohmann::json;

namespace config {
struct Config {
    synnax::RackKey rack_key;
    std::string rack_name;
    synnax::Config client_config;
    breaker::Config breaker_config;
    std::vector<std::string> integrations;
    bool debug;
};

std::pair<Config, freighter::Error> parse(const json &content);

json read(const std::string &path);
}
