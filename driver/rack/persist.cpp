// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <filesystem>
#include <string>
#include <utility>

#include "absl/log/log.h"

#include "x/cpp/kv/kv.h"

#include "driver/rack/rack.h"
#include "driver/rack/state.h"

namespace fs = std::filesystem;

namespace driver::rack {
constexpr auto PERSISTED_STATE_FILE_PERMISSIONS = fs::perms::owner_read |
                                                  fs::perms::owner_write;

constexpr auto PERSISTED_STATE_DIR_PERMISSIONS = fs::perms::owner_all;

namespace {
const std::string STATE_FILE = "persisted-state.json";
}

std::pair<std::string, x::errors::Error>
get_persisted_state_path(x::args::Parser &parser) {
    auto p_path = parser.field<std::string>("--state-file", "");
    // A bare file name has no parent directory to open.
    if (!p_path.empty()) return {fs::absolute(p_path).string(), x::errors::NIL};
    const auto dir = state::user_dir();
    if (dir.empty() || parser.flag(MACHINE_STATE_FLAG))
        return {(state::machine_dir() / STATE_FILE).string(), x::errors::NIL};
    // Root never writes in the home of the user that ran sudo.
    if (state::sudo())
        return {
            "",
            x::errors::Error(
                STATE_ERROR,
                "Run synnax-driver without sudo. Sudo is needed only while the Driver "
                "service is installed"
            )
        };
    return {(dir / STATE_FILE).string(), x::errors::NIL};
}

x::errors::Error rack::Config::secure_persisted_state(x::args::Parser &args) {
    const auto [path, err] = get_persisted_state_path(args);
    if (err) return err;
    return state::secure(path, args.field<std::string>("--state-file", "").empty());
}

x::errors::Error rack::Config::move_user_state(x::args::Parser &args) {
    const auto user_dir = state::user_dir();
    if (user_dir.empty() || !args.field<std::string>("--state-file", "").empty())
        return x::errors::NIL;
    const auto machine_dir = state::machine_dir();
    const auto from = user_dir / STATE_FILE;
    const auto to = machine_dir / STATE_FILE;
    std::error_code from_ec;
    std::error_code to_ec;
    const bool move = fs::exists(from, from_ec) && !fs::exists(to, to_ec);
    if (from_ec || to_ec)
        return x::errors::Error(
            STATE_ERROR,
            "failed to find the saved login: " + (from_ec ? from_ec : to_ec).message()
        );
    if (!move) return x::errors::NIL;
    LOG(INFO) << "moving the saved login from " << user_dir << " to " << machine_dir;
    std::error_code ec;
    fs::create_directories(machine_dir, ec);
    if (ec)
        return x::errors::Error(
            STATE_ERROR,
            "failed to create state directory: " + ec.message()
        );
    // Secured before the copy, so the key is never readable by other users.
    if (auto err = state::secure(to, true)) return err;
    fs::copy_file(from, to, ec);
    if (ec)
        return x::errors::Error(
            STATE_ERROR,
            "failed to copy the saved login: " + ec.message()
        );
    // OPC UA servers trust the client certificate in this directory.
    const auto opcua = user_dir / OPCUA_STATE_DIR;
    if (fs::exists(opcua, ec))
        fs::copy(
            opcua,
            machine_dir / OPCUA_STATE_DIR,
            fs::copy_options::recursive | fs::copy_options::skip_symlinks |
                fs::copy_options::skip_existing,
            ec
        );
    if (ec)
        return x::errors::Error(
            STATE_ERROR,
            "failed to copy the OPC UA certificates: " + ec.message()
        );
    if (auto err = state::secure(to, true)) return err;
    fs::remove(from, ec);
    if (!ec) fs::remove_all(opcua, ec);
    if (ec)
        return x::errors::Error(
            STATE_ERROR,
            "failed to remove the saved login: " + ec.message()
        );
    return x::errors::NIL;
}

std::pair<std::shared_ptr<x::kv::KV>, x::errors::Error>
open_kv(x::args::Parser &parser) {
    const auto [path, path_err] = get_persisted_state_path(parser);
    if (path_err) return {nullptr, path_err};
    if (auto err = state::prepare(path)) return {nullptr, err};
    const auto dir = fs::path(path).parent_path();
    std::error_code dir_ec;
    std::error_code file_ec;
    const bool dir_existed = fs::exists(dir, dir_ec);
    const bool file_existed = fs::exists(path, file_ec);
    if (dir_ec || file_ec)
        return {
            nullptr,
            x::errors::Error(
                STATE_ERROR,
                "failed to find state file: " + (dir_ec ? dir_ec : file_ec).message()
            )
        };
    auto [kv, err] = x::kv::JSONFile::open_file(
        x::kv::JSONFileConfig{
            .path = path,
            .dir_mode = PERSISTED_STATE_DIR_PERMISSIONS,
            .file_mode = PERSISTED_STATE_FILE_PERMISSIONS,
        }
    );
    if (err) return {kv, err};
    // State that this run created is not the operator's, so root gives it to the Linux
    // service user, as login does. An existing directory keeps its mode and owner.
    if (parser.field<std::string>("--state-file", "").empty() && !file_existed)
        if (auto sec_err = state::secure(path, !dir_existed)) return {nullptr, sec_err};
    return {kv, x::errors::NIL};
}

x::errors::Error rack::Config::load_persisted_state(x::args::Parser &args) {
    const auto [path, path_err] = get_persisted_state_path(args);
    if (path_err) return path_err;
    this->state_dir = fs::path(path).parent_path();
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

x::errors::Error rack::Config::save_conn_params(
    x::args::Parser &args,
    const synnax::Config &conn_params
) {
    auto [kv, err] = open_kv(args);
    if (err) return err;
    return kv->set("conn_params", conn_params.to_json().dump());
}

x::errors::Error
rack::Config::save_api_key(x::args::Parser &args, const synnax::Config &conn) {
    auto [kv, err] = open_kv(args);
    if (err) return err;
    auto params = conn.to_json();
    params.erase("username");
    params.erase("password");
    return kv->set("conn_params", params.dump());
}

x::errors::Error
rack::Config::save_remote_info(x::args::Parser &args, const RemoteInfo &remote_info) {
    auto [kv, err] = open_kv(args);
    if (err) return err;
    return kv->set("remote_info", remote_info.to_json().dump());
}

x::errors::Error rack::Config::clear_persisted_state(x::args::Parser &args) {
    auto [kv, err] = open_kv(args);
    if (err) return err;
    if (const auto d1_err = kv->del("conn_params")) return d1_err;
    if (const auto d2_err = kv->del("remote_info")) return d2_err;
    return x::errors::NIL;
}
}
