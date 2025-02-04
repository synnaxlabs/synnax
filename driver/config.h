// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <fstream>
#include <filesystem>

/// external
#include "nlohmann/json.hpp"

/// internal
#include "driver/config.h"
#include "driver/opc/opc.h"
#include "driver/ni/ni.h"
#include "client/cpp/synnax.h"
#include "driver/breaker/breaker.h"

#ifdef _WIN32
#include "driver/labjack/labjack.h"
#endif

#include "glog/logging.h"
#include "sequence/task.h"


using json = nlohmann::json;

namespace configd {
struct Config {
    synnax::RackKey rack_key;
    std::string rack_name;
    synnax::Config client_config;
    breaker::Config breaker_config;
    std::vector<std::string> integrations;
    bool debug;

    [[nodiscard]] bool integration_enabled(const std::string &integration) const {
        return std::find(
                   integrations.begin(),
                   integrations.end(),
                   integration
               ) != integrations.end();
    }
};

struct PersistedState {
    synnax::RackKey rack_key;
};

inline std::string get_persisted_state_path() {
#ifdef _WIN32
    const char* appdata = std::getenv("APPDATA");
    if (appdata == nullptr) return "";
    return std::string(appdata) + "\\synnax-driver\\persisted-state.json";
#elif defined(__APPLE__)
    const char* home = std::getenv("HOME");
    if (home == nullptr) return "";
    return std::string(home) + "/Library/Application Support/synnax-driver/persisted-state.json";
#else
    const char* home = std::getenv("HOME");
    if (home == nullptr) return "";
    return std::string(home) + "/.config/synnax-driver/persisted-state.json";
#endif
}

inline std::pair<PersistedState, freighter::Error> load_persisted_state() {
    auto path = get_persisted_state_path();
    if (path.empty())
        return {PersistedState{}, freighter::Error("failed to get home directory")};

    std::filesystem::path dir_path = std::filesystem::path(path).parent_path();
    std::error_code ec;
    std::filesystem::create_directories(dir_path, ec);
    if (ec)
        return {PersistedState{}, freighter::Error("failed to create directory: " + ec.message())};

    std::ifstream file(path);
    if (!file.is_open())
        return {PersistedState{.rack_key = 0}, freighter::NIL};

    try {
        json content = json::parse(file);
        auto parser = config::Parser(content);
        return {PersistedState{
            .rack_key = parser.optional<synnax::RackKey>("rack_key", 0)
        }, freighter::NIL};
    } catch (const json::exception& e) {
        return {PersistedState{}, freighter::Error("failed to parse persisted state: " + std::string(e.what()))};
    }
}

inline freighter::Error save_persisted_state(const PersistedState& state) {
    auto path = get_persisted_state_path();
    if (path.empty()) {
        return freighter::Error("failed to get home directory");
    }

    try {
        const json content = {
            {"rack_key", state.rack_key}
        };
        std::ofstream file(path);
        if (!file.is_open())
            return freighter::Error("failed to open file for writing");
        file << content.dump(4);
        return freighter::NIL;
    } catch (const std::exception& e) {
        return freighter::Error("failed to save persisted state: " + std::string(e.what()));
    }
}

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

    auto default_integrations = std::vector<std::string>{
        opc::INTEGRATION_NAME, ni::INTEGRATION_NAME, sequence::INTEGRATION_NAME
    };
#ifdef _WIN32
        default_integrations.push_back(labjack::INTEGRATION_NAME);
#endif
    auto integrations = p.optional<std::vector<std::string> >(
        "integrations", default_integrations
    );

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
