// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <format>
#include <string>
#include <unordered_map>
#include <vector>

#include "client/cpp/synnax.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"

#include "driver/common/status.h"
#include "driver/ethercat/errors/errors.h"
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

    /// @brief checks, in position order, that each slave is at its saved position on
    /// bus.
    /// @returns errors::TOPOLOGY_MISMATCH when a position holds no slave or another
    /// product.
    [[nodiscard]] x::errors::Error
    validate(const std::vector<slave::DiscoveryResult> &bus) const {
        std::vector<const slave::Properties *> by_position;
        by_position.reserve(this->by_key.size());
        for (const auto &[_, saved]: this->by_key)
            by_position.push_back(&saved);
        std::ranges::sort(by_position, {}, [](const slave::Properties *s) {
            return s->position;
        });
        for (const auto *saved_ptr: by_position) {
            const auto &saved = *saved_ptr;
            const auto found = std::ranges::find(
                bus,
                saved.position,
                [](const slave::DiscoveryResult &s) { return s.properties.position; }
            );
            if (found == bus.end())
                return x::errors::Error(
                    errors::TOPOLOGY_MISMATCH,
                    std::format(
                        "Slave {} ({}) is not on the bus.",
                        saved.position,
                        saved.name
                    )
                );
            const auto &actual = found->properties;
            const auto mismatch = [&](const char *field, uint32_t want, uint32_t got) {
                return x::errors::Error(
                    errors::TOPOLOGY_MISMATCH,
                    std::format(
                        "Slave {} ({}): expected {} {:#010x}, found {:#010x}.",
                        saved.position,
                        saved.name,
                        field,
                        want,
                        got
                    )
                );
            };
            if (actual.vendor_id != saved.vendor_id)
                return mismatch("vendor_id", saved.vendor_id, actual.vendor_id);
            if (actual.product_code != saved.product_code)
                return mismatch(
                    "product_code",
                    saved.product_code,
                    actual.product_code
                );
        }
        return x::errors::NIL;
    }
};
}
