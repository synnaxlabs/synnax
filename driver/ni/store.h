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
#include <memory>
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/ni/chassis/types.gen.h"
#include "client/cpp/ni/device/types.gen.h"
#include "client/cpp/synnax.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/filter/field.h"

#include "driver/common/sync.h"

namespace driver::ni {
/// @brief returns the identity group of the chassis.
inline synnax::ni::chassis::Identity identity(const synnax::ni::chassis::Chassis &c) {
    return {
        .rack = c.rack,
        .serial = c.serial,
        .resource_name = c.resource_name,
        .model = c.model,
    };
}

/// @brief returns the identity group of the device.
inline synnax::ni::device::Identity identity(const synnax::ni::device::Device &d) {
    return {
        .rack = d.rack,
        .serial = d.serial,
        .resource_name = d.resource_name,
        .model = d.model,
        .chassis = d.chassis,
        .hardware = d.hardware,
    };
}

/// @brief returns stored with the identity group of found.
inline synnax::ni::chassis::Chassis merge_identity(
    synnax::ni::chassis::Chassis stored,
    const synnax::ni::chassis::Chassis &found
) {
    stored.rack = found.rack;
    stored.serial = found.serial;
    stored.resource_name = found.resource_name;
    stored.model = found.model;
    return stored;
}

/// @brief returns stored with the identity group of found. It keeps the stored
/// hardware when found holds none, as a scan reads the hardware only when the stored
/// hardware is incomplete.
inline synnax::ni::device::Device merge_identity(
    synnax::ni::device::Device stored,
    const synnax::ni::device::Device &found
) {
    stored.rack = found.rack;
    stored.serial = found.serial;
    stored.resource_name = found.resource_name;
    stored.model = found.model;
    stored.chassis = found.chassis;
    if (found.hardware.has_value()) stored.hardware = found.hardware;
    return stored;
}

/// @brief the types and client that a Store of the resource D uses.
template<typename D>
struct Resource;

template<>
struct Resource<synnax::ni::chassis::Chassis> {
    using Where = synnax::ni::chassis::Where;
    using RetrieveOptions = synnax::ni::chassis::RetrieveOptions;
    using IdentityUpdate = synnax::ni::chassis::IdentityUpdate;
    using Status = synnax::ni::chassis::Status;
    static constexpr auto client = &synnax::ni::Client::chassis;

    static void set_owner(Status &status, const synnax::ni::chassis::Chassis &c) {
        status.owner = synnax::ni::chassis::resource_id(c.key);
        status.details = {.rack = c.rack, .chassis = c.key};
    }
};

template<>
struct Resource<synnax::ni::device::Device> {
    using Where = synnax::ni::device::Where;
    using RetrieveOptions = synnax::ni::device::RetrieveOptions;
    using IdentityUpdate = synnax::ni::device::IdentityUpdate;
    using Status = synnax::ni::device::Status;
    static constexpr auto client = &synnax::ni::Client::devices;

    static void set_owner(Status &status, const synnax::ni::device::Device &d) {
        status.owner = synnax::ni::device::resource_id(d.key);
        status.details = {.rack = d.rack, .device = d.key};
    }
};

/// @brief stores the NI resources of one type in a Core. It matches found hardware to
/// a stored resource by serial number, or by resource name when the hardware has no
/// serial number, as on simulated hardware.
/// @tparam D `synnax::ni::chassis::Chassis` or `synnax::ni::device::Device`.
template<typename D>
class Store final : public common::Store<D> {
    using R = Resource<D>;
    std::shared_ptr<synnax::Synnax> client;

    [[nodiscard]] const auto &resources() const { return this->client->ni.*R::client; }

public:
    explicit Store(std::shared_ptr<synnax::Synnax> client): client(std::move(client)) {}

    [[nodiscard]] std::string match_key(const D &resource) const override {
        return resource.serial.empty() ? resource.resource_name : resource.serial;
    }

    [[nodiscard]] D merge_identity(D stored, const D &found) const override {
        return ni::merge_identity(std::move(stored), found);
    }

    std::pair<std::vector<D>, x::errors::Error>
    retrieve(const synnax::rack::Key rack) override {
        return this->resources().retrieve(
            typename R::Where{.rack = rack},
            typename R::RetrieveOptions{.include_status = true}
        );
    }

    std::pair<std::optional<D>, x::errors::Error> find(const D &found) override {
        typename R::Where where;
        if (found.serial.empty())
            where.resource_name = found.resource_name;
        else
            where.serial = found.serial;
        auto [matches, err] = this->resources().retrieve(
            where,
            typename R::RetrieveOptions{.include_status = true}
        );
        if (err) return {std::nullopt, err};
        if (matches.empty()) return {std::nullopt, x::errors::NIL};
        // Two Drivers that first find the same hardware in one scan window can each
        // create it. Prefer the row on this rack, then the most recently reported one.
        const auto reported = [](const D &res) {
            return res.status.has_value() ? res.status->time : x::telem::TimeStamp(0);
        };
        const auto it = std::ranges::max_element(matches, [&](const D &a, const D &b) {
            if ((a.rack == found.rack) != (b.rack == found.rack))
                return b.rack == found.rack;
            return reported(a) < reported(b);
        });
        return {std::move(*it), x::errors::NIL};
    }

    x::errors::Error create(std::vector<D> &created) override {
        return this->resources().create(created);
    }

    x::errors::Error update_identity(const D &resource) override {
        typename R::IdentityUpdate update;
        static_cast<decltype(identity(resource)) &>(update) = identity(resource);
        return this->resources().update(resource.key, update).second;
    }

    x::errors::Error set_statuses(const std::vector<D> &present) override {
        std::vector<typename R::Status> statuses;
        statuses.reserve(present.size());
        for (const auto &res: present)
            R::set_owner(statuses.emplace_back(res.status.value()), res);
        return this->client->statuses.set(statuses);
    }
};

/// @brief stores NI chassis.
using ChassisStore = Store<synnax::ni::chassis::Chassis>;
/// @brief stores NI devices.
using DeviceStore = Store<synnax::ni::device::Device>;
}
