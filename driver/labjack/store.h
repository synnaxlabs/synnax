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
#include <charconv>
#include <cstdint>
#include <memory>
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/labjack/device/device.h"
#include "client/cpp/synnax.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/filter/field.h"
#include "x/cpp/telem/telem.h"

#include "driver/common/sync.h"

namespace driver::labjack {
using synnax::labjack::device::Device;

/// @brief returns stored with the identity group of found: its rack, serial,
/// connection, and model.
inline Device merge_identity(Device stored, const Device &found) {
    stored.rack = found.rack;
    stored.serial = found.serial;
    stored.connection = found.connection;
    stored.model = found.model;
    return stored;
}

/// @brief returns the actions that set each identity field of next that differs from
/// stored.
inline std::vector<synnax::labjack::device::Action>
identity_actions(const Device &stored, const Device &next) {
    std::vector<synnax::labjack::device::Action> acts;
    const auto set = [&](const std::string &field, x::json::json value) {
        acts.push_back(
            synnax::labjack::device::set(
                synnax::labjack::device::SetPayload{
                    .path = {field},
                    .value = std::move(value)
                }
            )
        );
    };
    if (next.rack != stored.rack) set("rack", next.rack);
    if (next.serial != stored.serial) set("serial", next.serial);
    if (next.connection != stored.connection) set("connection", next.connection);
    if (next.model != stored.model) set("model", next.model);
    return acts;
}

/// @brief retrieves the device that a task names by its legacy device key, which is
/// the serial number of the device.
/// @returns an error that matches VALIDATION when the key is not a serial number, or
/// NOT_FOUND when no device has the serial number.
inline std::pair<Device, x::errors::Error>
retrieve_task_device(const synnax::Synnax &client, const std::string &key) {
    std::uint32_t serial = 0;
    const auto *end = key.data() + key.size();
    if (const auto [ptr, ec] = std::from_chars(key.data(), end, serial);
        ec != std::errc() || ptr != end)
        return {
            {},
            x::errors::Error(
                x::errors::VALIDATION,
                "device key " + key + " is not a LabJack serial number"
            )
        };
    return client.labjack.devices.retrieve(
        synnax::labjack::device::Where{.serial = serial},
        x::filter::one
    );
}

/// @brief stores LabJack devices in a Core. It matches found hardware to a stored
/// device by serial number.
class Store final : public common::Store<Device> {
    std::shared_ptr<synnax::Synnax> client;

    [[nodiscard]] const synnax::labjack::device::Client &devices() const {
        return this->client->labjack.devices;
    }

public:
    explicit Store(std::shared_ptr<synnax::Synnax> client): client(std::move(client)) {}

    [[nodiscard]] std::string match_key(const Device &dev) const override {
        return std::to_string(dev.serial);
    }

    [[nodiscard]] Device
    merge_identity(Device stored, const Device &found) const override {
        return labjack::merge_identity(std::move(stored), found);
    }

    std::pair<std::vector<Device>, x::errors::Error>
    retrieve(const synnax::rack::Key rack) override {
        return this->devices().retrieve(
            synnax::labjack::device::Where{.rack = rack},
            synnax::labjack::device::RetrieveOptions{.include_status = true}
        );
    }

    std::pair<std::optional<Device>, x::errors::Error>
    find(const Device &found) override {
        auto [matches, err] = this->devices().retrieve(
            synnax::labjack::device::Where{.serial = found.serial},
            synnax::labjack::device::RetrieveOptions{.include_status = true}
        );
        if (err) return {std::nullopt, err};
        if (matches.empty()) return {std::nullopt, x::errors::NIL};
        // Two Drivers that first find the same hardware in one scan window can each
        // create it. Prefer the row on this rack, then the most recently reported one.
        const auto reported = [](const Device &d) {
            return d.status.has_value() ? d.status->time : x::telem::TimeStamp(0);
        };
        const auto it = std::ranges::max_element(
            matches,
            [&](const Device &a, const Device &b) {
                if ((a.rack == found.rack) != (b.rack == found.rack))
                    return b.rack == found.rack;
                return reported(a) < reported(b);
            }
        );
        return {std::move(*it), x::errors::NIL};
    }

    x::errors::Error create(std::vector<Device> &created) override {
        return this->devices().create(created);
    }

    x::errors::Error update_identity(const Device &stored, const Device &dev) override {
        const auto acts = identity_actions(stored, dev);
        if (acts.empty()) return x::errors::NIL;
        return this->devices().dispatch(dev.key, acts);
    }

    x::errors::Error set_statuses(const std::vector<Device> &present) override {
        std::vector<synnax::labjack::device::Status> statuses;
        statuses.reserve(present.size());
        for (const auto &dev: present) {
            auto &st = statuses.emplace_back(dev.status.value());
            st.owner = synnax::labjack::device::resource_id(dev.key);
            st.details = {.rack = dev.rack, .device = dev.key};
        }
        return this->client->statuses.set(statuses);
    }
};
}
