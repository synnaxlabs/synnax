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

#include <iostream>
#include <fstream>

#include "driver/config.h"
#include "driver/opc/opc.h"
#include "driver/ni/ni.h"

#ifdef _WIN32
#include "driver/labjack/labjack.h"
#endif

#include "nlohmann/json.hpp"
#include "glog/logging.h"


using json = nlohmann::json;

namespace configd {
struct Config {
    synnax::RackKey rack_key;
    std::string rack_name;
    synnax::Config client_config;
    breaker::Config breaker_config;
    std::vector<std::string> integrations;
    bool debug;
};

inline std::pair<configd::Config, freighter::Error> parse(
    const json &content
) {
    config::Parser p(content);
    auto conn = p.optional_child("connection");
    auto synnax_cfg = synnax::Config{
        .host = conn.optional<std::string>("host", "localhost"),
        .port = conn.optional<std::uint16_t>("port", 9090),
        .username = conn.optional<std::string>("username", "synnax"),
        .password = conn.optional<std::string>("password", "seldon"),
        .ca_cert_file = conn.optional<std::string>("ca_cert_file", ""),
        .client_cert_file = conn.optional<std::string>("client_cert_file", ""),
        .client_key_file = conn.optional<std::string>("client_key_file", ""),
    };

    auto retry = p.optional_child("retry");
    auto breaker_config = breaker::Config{
        .name = "driver",
        .base_interval = synnax::SECOND * retry.optional<int>("base_interval", 1),
        .max_retries = retry.optional<uint32_t>("max_retries", 50),
        .scale = retry.optional<float>("scale", 1.2),
    };

    auto rack = p.optional_child("rack");
    auto rack_key = rack.optional<synnax::RackKey>("key", 0);
    auto rack_name = rack.optional<std::string>("name", "sy_node_1_rack");

#ifdef _WIN32
    auto integrations = p.optional<std::vector<std::string> >(
        "integrations", {opc::INTEGRATION_NAME, ni::INTEGRATION_NAME, labjack::INTEGRATION_NAME});
#else
    auto integrations = p.optional<std::vector<std::string> >(
        "integrations", {opc::INTEGRATION_NAME, ni::INTEGRATION_NAME});
#endif
    auto debug = p.optional<bool>("debug", false);
    if (!p.ok()) return {Config{}, p.error()};
    return {
        configd::Config{
            .rack_key = rack_key,
            .rack_name = rack_name,
            .client_config = synnax_cfg,
            .breaker_config = breaker_config,
            .integrations = integrations,
            .debug = debug,
        },
        freighter::NIL,
    };
}


inline json read(const std::string &path) {
    VLOG(1) << "[driver] reading configuration from " << path;
    std::ifstream file(path);
    json content = json::object();
    if (file.is_open()) {
        std::string content_str;
        file.seekg(0, std::ios::end);
        content_str.resize(file.tellg());
        file.seekg(0, std::ios::beg);
        file.read(&content_str[0], content_str.size());
        file.close();
        content = json::parse(content_str);
    }
    return content;
}
}
