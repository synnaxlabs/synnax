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
#include "glog/logging.h"

/// internal
#include "driver/config.h"
#include "driver/opc/opc.h"
#include "driver/ni/ni.h"
#include "client/cpp/synnax.h"
#include "driver/breaker/breaker.h"
#include "sequence/sequence.h"

#ifdef _WIN32
#include "driver/labjack/labjack.h"
#endif


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
    synnax::Config connection;
};

inline std::string get_persisted_state_path() {
#ifdef _WIN32
    if (const char* appdata = std::getenv("LOCALAPPDATA")) 
        return std::string(appdata) + "\\synnax-driver\\persisted-state.json";
    return "C:\\ProgramData\\synnax-driver\\persisted-state.json";
#elif defined(__APPLE__)
    if (const char *home = std::getenv("HOME"))
        return std::string(home) +
               "/Library/Application Support/synnax-driver/persisted-state.json";
    return "/Library/Application Support/synnax-driver/persisted-state.json";
#else
    return "/var/lib/synnax-driver/persisted-state.json";
#endif
}

inline synnax::Config parse_synnax_config(config::Parser &conn) {
    return synnax::Config{
        .host = conn.optional<std::string>("host", "localhost"),
        .port = conn.optional<std::uint16_t>("port", 9090),
        .username = conn.optional<std::string>("username", "synnax"),
        .password = conn.optional<std::string>("password", "seldon"),
        .ca_cert_file = conn.optional<std::string>("ca_cert_file", ""),
        .client_cert_file = conn.optional<std::string>("client_cert_file", ""),
        .client_key_file = conn.optional<std::string>("client_key_file", ""),
    };
}

inline std::pair<PersistedState, freighter::Error> load_persisted_state() {
    auto path = get_persisted_state_path();
    LOG(INFO) << "Loading persisted state from " << path;
    if (path.empty())
        return {PersistedState{}, freighter::Error("failed to get home directory")};

    std::filesystem::path dir_path = std::filesystem::path(path).parent_path();
    std::error_code ec;

    // Check if directory exists before creating it
    if (!std::filesystem::exists(dir_path)) {
        std::filesystem::create_directories(dir_path, ec);
        if (ec)
            return {
                PersistedState{},
                freighter::Error("failed to create directory: " + ec.message())
            };

        // Set directory permissions to read/write/execute for all users
        std::filesystem::permissions(
            dir_path,
            std::filesystem::perms::owner_all |
            std::filesystem::perms::group_all |
            std::filesystem::perms::others_all,
            ec
        );
        if (ec)
            return {
                PersistedState{},
                freighter::Error("failed to set directory permissions: " + ec.message())
            };
    }

    std::ifstream file(path);
    if (!file.is_open())
        return {PersistedState{.rack_key = 0}, freighter::NIL};

    try {
        json content = json::parse(file);
        auto parser = config::Parser(content);
        auto conn = parser.optional_child("connection");
        return {
            PersistedState{
                .rack_key = parser.optional<synnax::RackKey>("rack_key", 0),
                .connection = parse_synnax_config(conn)
            },
            freighter::NIL
        };
    } catch (const json::exception &e) {
        return {
            PersistedState{},
            freighter::Error(
                "failed to parse persisted state: " + std::string(e.what()))
        };
    }
}

inline freighter::Error save_persisted_state(const PersistedState &state) {
    auto path = get_persisted_state_path();
    if (path.empty()) {
        return freighter::Error("failed to get home directory");
    }

    try {
        const json content = {
            {"rack_key", state.rack_key},
            {
                "connection", {
                    {"host", state.connection.host},
                    {"port", state.connection.port},
                    {"username", state.connection.username},
                    {"password", state.connection.password},
                    {"ca_cert_file", state.connection.ca_cert_file},
                    {"client_cert_file", state.connection.client_cert_file},
                    {"client_key_file", state.connection.client_key_file}
                }
            }
        };

        // Check if file exists before writing
        bool file_exists = std::filesystem::exists(path);

        std::ofstream file(path);
        if (!file.is_open())
            return freighter::Error("failed to open file for writing");
        file << content.dump(4);
        file.close();

        // Only set file permissions if the file was newly created
        if (!file_exists) {
            std::error_code ec;
            std::filesystem::permissions(
                path,
                std::filesystem::perms::owner_read | std::filesystem::perms::owner_write
                |
                std::filesystem::perms::group_read | std::filesystem::perms::group_write
                |
                std::filesystem::perms::others_read |
                std::filesystem::perms::others_write,
                ec
            );
            if (ec)
                return freighter::Error(
                    "failed to set file permissions: " + ec.message());
        }

        return freighter::NIL;
    } catch (const std::exception &e) {
        return freighter::Error(
            "failed to save persisted state: " + std::string(e.what()));
    }
}

inline std::pair<configd::Config, freighter::Error> parse(
    const json &content
) {
    config::Parser p(content);
    auto conn = p.optional_child("connection");
    auto synnax_cfg = parse_synnax_config(conn);

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
