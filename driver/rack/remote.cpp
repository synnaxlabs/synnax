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

#include "absl/log/log.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/os/os.h"

#include "driver/rack/rack.h"

namespace driver::rack {
namespace {
const std::string RACK_SUBJECT_TYPE = "rack";
const std::string API_KEY_NAME = "synnax-driver";

std::pair<x::uuid::UUID, x::errors::Error>
retrieve_cluster_key(const synnax::Synnax &client) {
    const auto state = client.connectivity->check();
    if (state.status != synnax::connection::Status::CONNECTED)
        return {x::uuid::UUID(), state.error};
    return x::uuid::UUID::parse(state.cluster_key);
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
    const auto [cluster_key, key_err] = retrieve_cluster_key(client);
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

x::errors::Error Config::load_remote() {
    const auto client = synnax::Synnax(this->connection);
    if (const auto err = client.auth->authenticate()) return err;
    const auto [cluster_key, key_err] = retrieve_cluster_key(client);
    if (key_err) return key_err;
    const auto subject = client.auth->current_session().subject;
    synnax::rack::Key rack_key = this->remote_info.rack_key;
    const auto *end = subject.key.data() + subject.key.size();
    if (!this->connection.api_key.empty() &&
        (subject.type != RACK_SUBJECT_TYPE ||
         std::from_chars(subject.key.data(), end, rack_key).ptr != end))
        return x::errors::Error(
            x::errors::VALIDATION,
            "the API key logs in as " + subject.type + " " + subject.key +
                ", not a rack. Run 'synnax-driver login' to create a key for a rack"
        );
    auto [rack, err] = client.racks.retrieve(rack_key);
    if (err) return err;
    this->rack = rack;
    this->remote_info.cluster_key = cluster_key;
    // A config can name an integration that this build leaves out.
    const auto built = built_integrations();
    this->rack.integrations.clear();
    for (const auto &i: this->integrations)
        if (std::ranges::find(built, i) != built.end())
            this->rack.integrations.push_back(i);
    return client.racks.create(this->rack);
}
}
