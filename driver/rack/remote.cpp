// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "x/cpp/xerrors/errors.h"
#include "x/cpp/xos/xos.h"

#include "driver/rack/rack.h"

xerrors::Error rack::Config::load_remote(breaker::Breaker &breaker) {
    std::pair<synnax::Rack, xerrors::Error> res;
    auto client = synnax::Synnax(this->connection);
    if (const auto err = client.auth->authenticate()) return err;
    if (this->remote_info.cluster_key != client.auth->cluster_info.cluster_key &&
        this->remote_info.rack_key != 0) {
        this->remote_info.rack_key = 0;
        this->remote_info.cluster_key = client.auth->cluster_info.cluster_key;
        LOG(INFO) << "cluster identity changed. Creating a new rack";
    }
    if (this->remote_info.rack_key != 0) {
        // if the rack key is non-zero, it means that persisted state or
        // configuration believes there's an existing rack in the cluster, and
        // we should use it as our task manager's rack.
        res = client.racks.retrieve(this->remote_info.rack_key);
        // If we tried to retrieve the rack and it doesn't exist, then we assume
        // that:
        //     1. Someone deleted the rack.
        //     2. The cluster identity has changed.
        //
        // In either case, set the rack key to zero and call the instantiate_rack
        // recursively to create a new rack.
        if (res.second.matches(xerrors::NOT_FOUND)) {
            LOG(INFO) << "Rack " << this->remote_info.rack_key
                      << " not found. Creating a new rack";
            this->remote_info.rack_key = 0;
            return this->load_remote(breaker);
        }
    } else {
        /// If the rack key is zero, we should create a new rack to use.
        const auto [host_name, ok] = xos::get_hostname();
        res = client.racks.create(host_name);
    }
    const xerrors::Error err = res.second;
    // If we can't reach the cluster, keep trying according to the breaker retry logic.
    if (err.matches(freighter::UNREACHABLE) && breaker.wait(err.message()))
        return this->load_remote(breaker);

    this->rack = res.first;
    this->remote_info.rack_key = res.first.key;
    this->remote_info.cluster_key = client.auth->cluster_info.cluster_key;
    return err;
}
