// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <filesystem>

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

std::string get_persisted_state_path(x::args::Parser &parser) {
    auto p_path = parser.field<std::string>("--state-file", "");
    if (!p_path.empty()) return p_path;
#ifdef _WIN32
    if (const char *appdata = std::getenv("LOCALAPPDATA"))
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

std::pair<std::shared_ptr<x::kv::KV>, x::errors::Error> open_kv(x::args::Parser &parser) {
    return x::kv::JSONFile::open(
        x::kv::JSONFileConfig{
            .path = get_persisted_state_path(parser),
            .dir_mode = PERSISTED_STATE_DIR_PERMISSIONS,
            .file_mode = PERSISTED_STATE_FILE_PERMISSIONS,
        }
    );
}

x::errors::Error driver::rack::Config::load_persisted_state(x::args::Parser &args) {
    auto [kv, open_err] = open_kv(args);
    if (open_err) return open_err;

    // Load the connection config.
    std::string conn = "{}";
    const auto c_err = kv->get("conn_params", conn);
    if (c_err && !x::errors::NOT_FOUND.matches(c_err)) return c_err;
    auto conn_parser = x::json::Parser(conn);
    this->connection.override(conn_parser);

    // Load the cached remote info
    std::string remote_info = "{}";
    const auto r_err = kv->get("remote_info", remote_info);
    if (r_err && !x::errors::NOT_FOUND.matches(r_err)) return r_err;
    auto remote_parser = x::json::Parser(remote_info);
    this->remote_info.override(remote_parser);

    return x::errors::NIL;
}

x::errors::Error
driver::rack::Config::save_conn_params(x::args::Parser &args, const synnax::Config &conn_params) {
    auto [kv, err] = open_kv(args);
    return kv->set("conn_params", conn_params.to_json().dump());
}

x::errors::Error
driver::rack::Config::save_remote_info(x::args::Parser &args, const RemoteInfo &remote_info) {
    auto [kv, err] = open_kv(args);
    return kv->set("remote_info", remote_info.to_json().dump());
}

x::errors::Error driver::rack::Config::clear_persisted_state(x::args::Parser &args) {
    auto [kv, err] = open_kv(args);
    if (err) return err;
    if (const auto d1_err = kv->del("conn_params")) return d1_err;
    if (const auto d2_err = kv->del("remote_info")) return d2_err;
    return x::errors::NIL;
}
