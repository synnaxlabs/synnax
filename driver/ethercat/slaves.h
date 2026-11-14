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

    /// @brief finds, in position order, each slave on bus by its identity: its
    /// vendor, product, and serial number, or with serial number 0, its saved position.
    /// Then saves the position where it found each slave.
    /// @returns the position on bus of each slave, keyed by its saved position, or
    /// errors::TOPOLOGY_MISMATCH when a slave is not on bus, or when two slaves have
    /// one saved position or are at one position on bus.
    [[nodiscard]] std::pair<std::map<uint16_t, uint16_t>, x::errors::Error>
    bind(const std::vector<slave::DiscoveryResult> &bus) {
        std::vector<slave::Properties *> by_position;
        by_position.reserve(this->by_key.size());
        for (auto &[_, saved]: this->by_key)
            by_position.push_back(&saved);
        std::ranges::sort(by_position, {}, [](const slave::Properties *s) {
            return std::tie(s->position, s->name);
        });
        std::map<uint16_t, uint16_t> moves;
        std::map<uint16_t, const slave::Properties *> bound;
        for (size_t i = 0; i < by_position.size(); ++i) {
            const auto &saved = *by_position[i];
            if (i > 0 && by_position[i - 1]->position == saved.position)
                return {
                    {},
                    x::errors::Error(
                        errors::TOPOLOGY_MISMATCH,
                        std::format(
                            "{} and {} are both saved as slave {}.",
                            by_position[i - 1]->name,
                            saved.name,
                            saved.position
                        )
                    )
                };
            const auto [found, err] = find(bus, saved);
            if (err) return {{}, err};
            const auto pos = found->properties.position;
            if (const auto [it, added] = bound.emplace(pos, &saved); !added)
                return {
                    {},
                    x::errors::Error(
                        errors::TOPOLOGY_MISMATCH,
                        std::format(
                            "{} and {} are both slave {} on the bus.",
                            it->second->name,
                            saved.name,
                            pos
                        )
                    )
                };
            moves[saved.position] = pos;
        }
        for (auto *saved: by_position)
            saved->position = moves.at(saved->position);
        return {moves, x::errors::NIL};
    }

private:
    /// @brief returns the slave on bus with the identity of saved.
    /// @returns errors::TOPOLOGY_MISMATCH when bus does not hold it.
    static std::
        pair<std::vector<slave::DiscoveryResult>::const_iterator, x::errors::Error>
        find(
            const std::vector<slave::DiscoveryResult> &bus,
            const slave::Properties &saved
        ) {
        if (saved.serial != 0) {
            const auto same = [&saved](const slave::DiscoveryResult &s) {
                const auto &p = s.properties;
                return p.vendor_id == saved.vendor_id &&
                       p.product_code == saved.product_code && p.serial == saved.serial;
            };
            // Two units can share an SII serial number.
            auto found = std::ranges::find_if(bus, [&](const auto &s) {
                return same(s) && s.properties.position == saved.position;
            });
            if (found == bus.end()) found = std::ranges::find_if(bus, same);
            if (found != bus.end()) return {found, x::errors::NIL};
            return {
                found,
                x::errors::Error(
                    errors::TOPOLOGY_MISMATCH,
                    std::format(
                        "{} (serial {}) is not on the bus.",
                        saved.name,
                        saved.serial
                    )
                )
            };
        }
        const auto found = std::ranges::find(
            bus,
            saved.position,
            [](const slave::DiscoveryResult &s) { return s.properties.position; }
        );
        if (found == bus.end())
            return {
                found,
                x::errors::Error(
                    errors::TOPOLOGY_MISMATCH,
                    std::format(
                        "Slave {} ({}) is not on the bus.",
                        saved.position,
                        saved.name
                    )
                )
            };
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
            return {found, mismatch("vendor_id", saved.vendor_id, actual.vendor_id)};
        if (actual.product_code != saved.product_code)
            return {
                found,
                mismatch("product_code", saved.product_code, actual.product_code)
            };
        return {found, x::errors::NIL};
    }
};
}
