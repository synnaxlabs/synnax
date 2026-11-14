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
#include <map>
#include <set>
#include <string>
#include <tuple>
#include <unordered_map>
#include <utility>
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

    /// @brief finds each slave on bus by its identity: its vendor, product, and serial
    /// number, or with serial number 0, its saved position. A slave at its saved
    /// position keeps it before a moved slave takes a position. Then saves the position
    /// where it found each slave.
    /// @returns the position on bus of each slave, keyed by its saved position, or
    /// errors::TOPOLOGY_MISMATCH when a slave is not on bus, or when two slaves have
    /// one saved position.
    [[nodiscard]] std::pair<std::map<uint16_t, uint16_t>, x::errors::Error>
    bind(const std::vector<slave::DiscoveryResult> &bus) {
        std::vector<slave::Properties *> by_position;
        by_position.reserve(this->by_key.size());
        for (auto &[_, saved]: this->by_key)
            by_position.push_back(&saved);
        std::ranges::sort(by_position, {}, [](const slave::Properties *s) {
            return std::tie(s->position, s->name);
        });
        for (size_t i = 1; i < by_position.size(); ++i)
            if (by_position[i - 1]->position == by_position[i]->position)
                return {
                    {},
                    x::errors::Error(
                        errors::TOPOLOGY_MISMATCH,
                        std::format(
                            "{} and {} are both saved as slave {}.",
                            by_position[i - 1]->name,
                            by_position[i]->name,
                            by_position[i]->position
                        )
                    )
                };
        std::map<uint16_t, uint16_t> moves;
        std::set<uint16_t> bound;
        for (const auto *saved: by_position) {
            const auto at = std::ranges::find(
                bus,
                saved->position,
                [](const slave::DiscoveryResult &s) { return s.properties.position; }
            );
            if (saved->serial == 0) {
                if (auto err = check_model(bus, at, *saved)) return {{}, err};
            } else if (at == bus.end() || !same_unit(at->properties, *saved))
                continue;
            moves[saved->position] = saved->position;
            bound.insert(saved->position);
        }
        for (const auto *saved: by_position) {
            if (moves.contains(saved->position)) continue;
            const auto found = std::ranges::find_if(bus, [&](const auto &s) {
                return !bound.contains(s.properties.position) &&
                       same_unit(s.properties, *saved);
            });
            if (found == bus.end())
                return {
                    {},
                    x::errors::Error(
                        errors::TOPOLOGY_MISMATCH,
                        std::format(
                            "{} (serial {}) is not on the bus.",
                            saved->name,
                            saved->serial
                        )
                    )
                };
            moves[saved->position] = found->properties.position;
            bound.insert(found->properties.position);
        }
        for (auto *saved: by_position)
            saved->position = moves.at(saved->position);
        return {moves, x::errors::NIL};
    }

private:
    /// @brief whether live and saved have one vendor, product, and serial number.
    static bool
    same_unit(const slave::Properties &live, const slave::Properties &saved) {
        return live.vendor_id == saved.vendor_id &&
               live.product_code == saved.product_code && live.serial == saved.serial;
    }

    /// @brief checks that at, the slave on bus at the saved position of saved, is of
    /// its model.
    /// @returns errors::TOPOLOGY_MISMATCH when at is the end of bus or of another
    /// model.
    static x::errors::Error check_model(
        const std::vector<slave::DiscoveryResult> &bus,
        const std::vector<slave::DiscoveryResult>::const_iterator at,
        const slave::Properties &saved
    ) {
        if (at == bus.end())
            return x::errors::Error(
                errors::TOPOLOGY_MISMATCH,
                std::format(
                    "Slave {} ({}) is not on the bus.",
                    saved.position,
                    saved.name
                )
            );
        const auto &actual = at->properties;
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
            return mismatch("product_code", saved.product_code, actual.product_code);
        return x::errors::NIL;
    }
};
}
