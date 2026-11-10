// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <charconv>
#include <tuple>

#include "absl/log/log.h"

#include "client/cpp/connection/checker.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/os/os.h"

#include "driver/rack/rack.h"
#include "driver/version/version.h"

namespace driver::rack {
namespace {
const std::string RACK_SUBJECT_TYPE = "rack";
const std::string API_KEY_NAME = "synnax-driver";

/// @brief returns the cluster key that a connectivity check reports, or the error of a
/// check that did not reach the Core.
std::pair<x::uuid::UUID, x::errors::Error>
parse_cluster_key(const synnax::connection::State &state) {
    if (state.status != synnax::connection::Status::CONNECTED)
        return {x::uuid::UUID(), state.error};
    return x::uuid::UUID::parse(state.cluster_key);
}

/// @brief returns the key of the rack that the subject names, and false when the
/// subject is not a rack.
std::pair<synnax::rack::Key, bool>
parse_rack_key(const synnax::auth::Subject &subject) {
    synnax::rack::Key key = 0;
    const auto *end = subject.key.data() + subject.key.size();
    if (subject.type != RACK_SUBJECT_TYPE ||
        std::from_chars(subject.key.data(), end, key).ptr != end)
        return {0, false};
    return {key, true};
}

/// @brief returns the rack that the API key logs in as and the cluster key of the Core
/// of conn. The rack key is 0 when the Core rejects the key or the key logs in as
/// another subject.
std::pair<RemoteInfo, x::errors::Error>
retrieve_key_rack(synnax::Config conn, const std::string &api_key) {
    conn.api_key = api_key;
    conn.username.clear();
    conn.password.clear();
    const auto client = synnax::Synnax(conn);
    if (const auto err = client.auth->authenticate()) {
        if (err.matches(synnax::auth::ERR_INVALID_CREDENTIALS))
            return {RemoteInfo{}, x::errors::NIL};
        return {RemoteInfo{}, err};
    }
    const auto [rack_key, is_rack] = parse_rack_key(
        client.auth->current_session().subject
    );
    if (!is_rack) return {RemoteInfo{}, x::errors::NIL};
    const auto [cluster_key, err] = parse_cluster_key(client.connectivity->check());
    return {RemoteInfo{.rack_key = rack_key, .cluster_key = cluster_key}, err};
}

std::pair<synnax::rack::Rack, x::errors::Error>
create_rack(const synnax::Synnax &client) {
    const auto [host_name, ok] = x::os::get_hostname();
    return client.racks.create(host_name);
}
}

x::errors::Error Config::enroll() {
    const auto client = synnax::Synnax(this->connection);
    if (const auto err = client.auth->authenticate()) return err;
    const auto [cluster_key, key_err] = parse_cluster_key(client.connectivity->check());
    if (key_err) return key_err;
    std::pair<synnax::rack::Rack, x::errors::Error> res;
    bool created = false;
    if (this->remote_info.rack_key != 0 &&
        this->remote_info.cluster_key == cluster_key) {
        res = client.racks.retrieve(this->remote_info.rack_key);
        if (res.second.matches(x::errors::NOT_FOUND)) {
            LOG(INFO) << "rack " << this->remote_info.rack_key
                      << " not found. Creating a new rack";
            res = create_rack(client);
            created = true;
        }
    } else {
        res = create_rack(client);
        created = true;
    }
    if (res.second) return res.second;
    auto [key, create_err] = client.api_keys.create(
        API_KEY_NAME,
        {RACK_SUBJECT_TYPE, std::to_string(res.first.key)}
    );
    if (create_err) {
        // A retry creates another rack, so remove the one no key logs in as.
        if (created)
            if (const auto del_err = client.racks.del(res.first.key))
                LOG(WARNING) << "failed to remove rack " << res.first.key << ": "
                             << del_err;
        if (!create_err.matches(synnax::auth::ERR_ACCESS_DENIED)) return create_err;
        if (!created) this->rack = res.first;
        return x::errors::Error(
            create_err,
            "User " + this->connection.username +
                " cannot create an API key for rack " + std::to_string(res.first.key) +
                ". Run synnax-driver login as a user that can create racks, or "
                "create an API key for the rack in the Console and set it as "
                "api_key in the Driver configuration."
        );
    }
    this->rack = res.first;
    this->remote_info = {.rack_key = 0, .cluster_key = cluster_key};
    this->connection.api_key = key.token;
    this->connection.username.clear();
    this->connection.password.clear();
    return x::errors::NIL;
}

std::pair<Config, x::errors::Error>
Config::login(x::args::Parser &args, const synnax::Config &conn) {
    Config cfg;
    if (auto err = cfg.load_persisted_state(args)) return {cfg, err};
    const auto &saved = cfg.connection;
    if (!saved.api_key.empty()) {
        // Only the Core that issued the saved key may receive it, and never in
        // plaintext when the key was saved with TLS.
        if (saved.host == conn.host && saved.port == conn.port &&
            (conn.secure || !saved.secure)) {
            const auto [info, err] = retrieve_key_rack(conn, saved.api_key);
            if (err) return {cfg, err};
            if (info.rack_key != 0) cfg.remote_info = info;
        } else
            LOG(WARNING) << "the saved API key is for " << saved.address()
                         << (saved.secure ? " with TLS" : "")
                         << ", so login does not send it to " << conn.address()
                         << " and does not keep the rack of that key";
    }
    cfg.connection = conn;
    if (auto err = cfg.enroll()) return {cfg, err};
    if (auto err = save_api_key(args, cfg.connection)) return {cfg, err};
    return {cfg, save_remote_info(args, cfg.remote_info)};
}

x::errors::Error Config::load_remote() {
    const auto client = synnax::Synnax(this->connection);
    if (const auto err = client.auth->authenticate()) return err;
    const auto core = client.connectivity->check();
    const auto [cluster_key, key_err] = parse_cluster_key(core);
    if (key_err) return key_err;
    const auto subject = client.auth->current_session().subject;
    synnax::rack::Key rack_key = this->remote_info.rack_key;
    bool is_rack = true;
    if (!this->connection.api_key.empty())
        std::tie(rack_key, is_rack) = parse_rack_key(subject);
    if (!is_rack)
        return x::errors::Error(
            x::errors::VALIDATION,
            "the API key logs in as " + subject.type + " " + subject.key +
                ", not a rack. Run 'synnax-driver login' to create a key for a rack"
        );
    auto [rack, err] = client.racks.retrieve(rack_key);
    if (err) return err;
    this->rack = rack;
    this->rack.version = SYNNAX_DRIVER_VERSION;
    this->remote_info.cluster_key = cluster_key;
    if (!synnax::connection::versions_compatible(
            SYNNAX_DRIVER_VERSION,
            core.node_version
        ))
        this->incompatible_core_version = core.node_version;
    // A config can name an integration that this build leaves out.
    const auto built = built_integrations();
    this->rack.integrations.clear();
    for (const auto &i: this->integrations)
        if (std::ranges::find(built, i) != built.end())
            this->rack.integrations.push_back(i);
    return client.racks.dispatch(
        this->rack.key,
        {
            synnax::rack::set({.path = {"version"}, .value = SYNNAX_DRIVER_VERSION}),
            synnax::rack::set(
                {.path = {"integrations"}, .value = this->rack.integrations}
            ),
        }
    );
}
}
