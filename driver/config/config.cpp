// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file 
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source 
// License, use of this software will be governed by the Apache License, Version 2.0, 
// included in the file licenses/APL.txt.

// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/config/config.h"

/// std
#include <fstream>
#include <filesystem>

/// external
#include "glog/logging.h"

/// internal
#include <unistd.h>

#include "driver/opc/opc.h"
#include "driver/ni/ni.h"
#include "driver/sequence/sequence.h"

#ifdef _WIN32
#include "driver/labjack/labjack.h"
#endif

constexpr auto PERSISTED_STATE_FILE_PERMISSIONS = std::filesystem::perms::owner_read |
                                                  std::filesystem::perms::owner_write |
                                                  std::filesystem::perms::group_read |
                                                  std::filesystem::perms::group_write |
                                                  std::filesystem::perms::others_read |
                                                  std::filesystem::perms::others_write;

constexpr auto PERSISTED_STATE_DIR_PERMISSIONS = std::filesystem::perms::owner_all |
                                                 std::filesystem::perms::group_all |
                                                 std::filesystem::perms::others_all;

void parse_synnax_config(config::Parser &p, driver::Config &cfg) {
    cfg.connection.host = p.optional(
        "host",
        cfg.connection.host
    );
    cfg.connection.port = p.optional(
        "port",
        cfg.connection.port
    );
    cfg.connection.username = p.optional(
        "username",
        cfg.connection.username
    );
    cfg.connection.password = p.optional(
        "password",
        cfg.connection.password
    );
    cfg.connection.ca_cert_file = p.optional(
        "ca_cert_file",
        cfg.connection.ca_cert_file
    );
    cfg.connection.client_cert_file = p.optional(
        "client_cert_file",
        cfg.connection.client_cert_file
    );
    cfg.connection.client_key_file = p.optional(
        "client_key_file",
        cfg.connection.client_key_file
    );
}

void parse_retry_config(config::Parser &p, driver::Config &cfg) {
    cfg.breaker_config.name = p.optional(
        "name",
        cfg.breaker_config.name
    );
    // cfg.breaker_config.base_interval = telem::SECOND * p.optional(
    //                                        "base_interval",
    //                                        cfg.breaker_config.base_interval /
    //                                        telem::SECOND
    //                                    );
    cfg.breaker_config.max_retries = p.optional(
        "max_retries",
        cfg.breaker_config.max_retries
    );
    cfg.breaker_config.scale = p.optional(
        "scale",
        cfg.breaker_config.scale
    );
}


xerrors::Error apply_config_arg(driver::Config &cfg, int argc, char **argv) {
    std::string config_path;
    for (int i = 2; i < argc; i++) {
        const std::string arg = argv[i];
        if (arg == "--config") {
            config_path = argv[++i];
            break;
        }
    }
    if (config_path.empty()) {
        LOG(INFO) << "no config file provided";
        return xerrors::NIL;
    }
    auto p = config::Parser::from_file_path(config_path);
    auto conn = p.optional_child("connection");
    parse_synnax_config(conn, cfg);
    auto retry = p.optional_child("retry");
    parse_retry_config(retry, cfg);
    return xerrors::NIL;
}


std::string get_persisted_state_path() {
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

xerrors::Error maybe_create_persisted_state_file(const std::string &path) {
    if (std::filesystem::exists(path)) return xerrors::NIL;
    const std::filesystem::path dir_path = std::filesystem::path(path).parent_path();
    std::error_code ec;
    if (!std::filesystem::exists(dir_path)) {
        std::filesystem::create_directories(dir_path, ec);
        if (ec)
            return xerrors::Error("failed to create directory: " + ec.message());
        std::filesystem::permissions(dir_path, PERSISTED_STATE_DIR_PERMISSIONS, ec);
        if (ec)
            return xerrors::Error(
                "failed to set directory permissions: " + ec.message()
            );
    }
    std::filesystem::permissions(path, PERSISTED_STATE_FILE_PERMISSIONS, ec);
    if (ec)
        return xerrors::Error(
            "failed to set file permissions: " + ec.message()
        );
    return xerrors::NIL;
}


xerrors::Error apply_persisted_state(driver::Config &cfg) {
    const auto path = get_persisted_state_path();
    if (const auto err = maybe_create_persisted_state_file(path)) return err;
    auto parser = config::Parser::from_file_path(path);
    auto conn = parser.optional_child("connection");
    parse_synnax_config(conn, cfg);
    return parser.error();
}

const std::vector<std::string> default_integrations() {
#ifdef _WIN32
    return {
        opc::INTEGRATION_NAME,
        ni::INTEGRATION_NAME,
        labjack::INTEGRATION_NAME,
        sequence::INTEGRATION_NAME
    };
#else
    return {
        opc::INTEGRATION_NAME,
        ni::INTEGRATION_NAME,
        sequence::INTEGRATION_NAME
    };
#endif
}

std::pair<driver::Config, xerrors::Error> driver::Config::load(
    const int argc,
    char ** argv
) {
    Config cfg{
        .connection = {
            .host = "localhost",
            .port = 9090,
            .username = "synnax",
            .password = "seldon",
        },
        .breaker_config = breaker::Config{
            .name = "drier",
            .base_interval = telem::TimeSpan::seconds(1),
            .max_retries = 50,
            .scale = 1.1,
        }
    };
    apply_persisted_state(cfg);
    apply_config_arg(cfg, argc, argv);
    return {cfg, xerrors::NIL};
}



bool driver::Config::integration_enabled(const std::string &integration) const {
    return std::find(
               integrations.begin(),
               integrations.end(),
               integration
           ) != integrations.end();
}

xerrors::Error driver::save_persisted_state(const driver::PersistedState &state) {
    auto path = get_persisted_state_path();
    if (const auto err = maybe_create_persisted_state_file(path)) return err;
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
        std::ofstream file(path);
        if (!file.is_open())
            return xerrors::Error("failed to open file for writing");
        file << content.dump(4);
        file.close();
        return xerrors::NIL;
    } catch (const std::exception &e) {
        return xerrors::Error(
            "failed to save persisted state: " + std::string(e.what()));
    }
}
