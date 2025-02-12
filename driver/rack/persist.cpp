// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std
#include <filesystem>

/// module
#include "x/cpp/kv/kv.h"

// internal
#include "driver/rack/rack.h"

constexpr auto PERSISTED_STATE_FILE_PERMISSIONS = std::filesystem::perms::owner_read |
                                                  std::filesystem::perms::owner_write |
                                                  std::filesystem::perms::group_read |
                                                  std::filesystem::perms::group_write |
                                                  std::filesystem::perms::others_read |
                                                  std::filesystem::perms::others_write;

constexpr auto PERSISTED_STATE_DIR_PERMISSIONS = std::filesystem::perms::owner_all |
                                                 std::filesystem::perms::group_all |
                                                 std::filesystem::perms::others_all;

std::string get_persisted_state_path(const int argc, char *argv[]) {
    std::string p_path;
    for (int i = 2; i < argc; i++) {
        const std::string arg = argv[i];
        if (arg == "--state-file") {
            p_path = argv[++i];
            break;
        }
    }
    if (!p_path.empty()) return p_path;
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

std::pair<std::shared_ptr<kv::KV>, xerrors::Error> open_kv(int argc, char **argv) {
    return kv::JSONFile::open(kv::JSONFileConfig{
        .path = get_persisted_state_path(argc, argv),
        .dir_mode = PERSISTED_STATE_DIR_PERMISSIONS,
        .file_mode = PERSISTED_STATE_FILE_PERMISSIONS,
    });
}

xerrors::Error rack::Config::load_persisted_state(int argc, char **argv) {
    auto [kv, open_err] = open_kv(argc, argv);
    if (open_err) return open_err;

    // Load the connection config.
    std::string conn = "{}";
    const auto c_err = kv->get("conn_params", conn);
    if (c_err && !xerrors::NOT_FOUND.matches(c_err)) return c_err;
    auto conn_parser = config::Parser(conn);
    this->connection.override(conn_parser);

    // Load the cached remote info
    std::string remote_info = "{}";
    const auto r_err = kv->get("remote_info", remote_info);
    if (r_err && !xerrors::NOT_FOUND.matches(r_err)) return r_err;
    auto remote_parser = config::Parser(remote_info);
    this->remote.override(remote_parser);

    return xerrors::NIL;
}

xerrors::Error rack::Config::save_conn_params(
    int argc,
    char **argv,
    const synnax::Config &conn_params
) {
    auto [kv, err] = open_kv(argc, argv);
    return kv->set("conn_params", conn_params.to_json().dump());
}

xerrors::Error rack::Config::save_remote_info(
    int argc,
    char **argv,
    const RemoteInfo &remote_info
) {
    auto [kv, err] = open_kv(argc, argv);
    return kv->set("remote_info", remote_info.to_json().dump());
}

xerrors::Error rack::Config::clear_persisted_state(int argc, char **argv) {
    auto [kv, err] = open_kv(argc, argv);
    if (err) return err;
    if (const auto d1_err = kv->del("conn_params")) return d1_err;
    if (const auto d2_err = kv->del("remote_info")) return d2_err;
    return xerrors::NIL;
}
