// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <map>
#include <string>
#include <unordered_map>
#include <utility>
#include <vector>

#include "client/cpp/synnax.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"

#include "driver/common/status.h"
#include "driver/ethercat/slave/slave.h"

namespace driver::ethercat {
/// @brief the slaves that the channels of a task name.
struct Slaves {
    /// @brief the network that every slave is on.
    std::string network;
    /// @brief the saved properties of each slave, keyed by device key.
    std::unordered_map<std::string, slave::Properties> by_key;

    /// @brief returns the slave with device key, retrieving it from client on first
    /// use. Sets a field error at path + "device" on cfg and returns nullptr when the
    /// Core does not give the device, its properties do not parse, or it is on another
    /// network than the slaves before it.
    const slave::Properties *resolve(
        synnax::Synnax &client,
        const std::string &key,
        x::json::Parser &cfg,
        const std::string &path = ""
    ) {
        if (const auto it = this->by_key.find(key); it != this->by_key.end())
            return &it->second;
        auto [dev, err] = client.devices.retrieve(key);
        if (err) {
            cfg.field_err(path + "device", err.data);
            return nullptr;
        }
        auto parser = x::json::Parser(dev.properties);
        auto props = slave::Properties::parse(parser);
        if (parser.error()) {
            cfg.field_err(
                path + "device",
                common::config_err_description(parser.error())
            );
            return nullptr;
        }
        if (this->by_key.empty())
            this->network = props.network;
        else if (props.network != this->network) {
            cfg.field_err(path + "device", "all slaves must be on the same network");
            return nullptr;
        }
        return &this->by_key.emplace(key, std::move(props)).first->second;
    }

    /// @brief finds each slave on bus with slave::bind, then saves the position where
    /// it found each slave.
    /// @returns the position on bus of each slave, keyed by its saved position, or the
    /// error of slave::bind.
    [[nodiscard]] std::pair<std::map<uint16_t, uint16_t>, x::errors::Error>
    bind(const std::vector<slave::DiscoveryResult> &bus) {
        std::vector<slave::Identity> saved;
        saved.reserve(this->by_key.size());
        for (const auto &[_, s]: this->by_key)
            saved.push_back({
                .position = s.position,
                .vendor_id = s.vendor_id,
                .product_code = s.product_code,
                .serial = s.serial,
                .name = s.name,
            });
        const auto [positions, err] = slave::bind(saved, bus);
        if (err) return {{}, err};
        std::map<uint16_t, uint16_t> moves;
        size_t i = 0;
        for (auto &[_, s]: this->by_key) {
            moves[s.position] = positions[i];
            s.position = positions[i++];
        }
        return {moves, x::errors::NIL};
    }
};
}
