// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <stdexcept>
#include <unordered_map>
#include <utility>

#include "absl/log/log.h"

#include "client/cpp/ni/chassis/chassis.h"
#include "client/cpp/ni/device/device.h"

#include "driver/ni/resources.h"

namespace driver::ni {
namespace {
/// @brief returns the serial number of the scanned hardware, or nothing on simulated
/// hardware or hardware that reports none.
std::optional<std::string> serial(const Device &dev) {
    if (dev.is_simulated || dev.key.empty()) return std::nullopt;
    return dev.key;
}
}

std::string match_key(const Device &scanned) {
    return serial(scanned).value_or(scanned.resource_name);
}

std::vector<synnax::ni::chassis::Chassis>
found_chassis(const std::vector<Device> &scanned) {
    std::vector<synnax::ni::chassis::Chassis> found;
    for (const auto &dev: scanned)
        if (dev.is_chassis)
            found.push_back({
                .rack = dev.rack,
                .name = dev.name,
                .serial = serial(dev).value_or(""),
                .resource_name = dev.resource_name,
                .model = dev.model,
            });
    return found;
}

std::vector<synnax::ni::device::Device>
found_devices(const std::vector<Device> &scanned, const ChassisKey &chassis_key) {
    std::unordered_map<std::string, std::string> link_to_chassis;
    for (const auto &dev: scanned)
        if (dev.is_chassis && !dev.provides_link_name.empty())
            link_to_chassis[dev.provides_link_name] = match_key(dev);
    std::vector<synnax::ni::device::Device> found;
    for (const auto &dev: scanned) {
        if (dev.is_chassis) continue;
        auto &res = found.emplace_back(
            synnax::ni::device::Device{
                .rack = dev.rack,
                .name = dev.name,
                .serial = serial(dev),
                .resource_name = dev.resource_name,
                .model = dev.model,
                .hardware = dev.hardware,
            }
        );
        if (dev.connects_to_link_name.empty()) continue;
        const auto it = link_to_chassis.find(dev.connects_to_link_name);
        if (it != link_to_chassis.end())
            res.chassis = chassis_key(match_key(dev), it->second);
    }
    return found;
}

const std::vector<std::string> Resources::SIGNALS = {
    synnax::ni::chassis::SET_CHANNEL,
    synnax::ni::chassis::DELETE_CHANNEL,
    synnax::ni::device::SET_CHANNEL,
    synnax::ni::device::DELETE_CHANNEL,
};

Resources::Resources(
    std::unique_ptr<common::Store<synnax::ni::chassis::Chassis>> chassis,
    std::unique_ptr<common::Store<synnax::ni::device::Device>> devices,
    const synnax::rack::Key rack
):
    chassis(std::move(chassis), rack, SCAN_LOG_PREFIX),
    devices(std::move(devices), rack, SCAN_LOG_PREFIX) {}

x::errors::Error Resources::sync(const std::vector<Device> &scanned) {
    // A failed chassis sync can leave a chassis without a key, which would detach its
    // modules.
    if (const auto err = this->chassis.sync(found_chassis(scanned))) return err;
    return this->devices.sync(found_devices(
        scanned,
        [this](const std::string &module, const std::string &chassis)
            -> std::optional<synnax::ni::chassis::Key> {
            if (const auto c = this->chassis.get(chassis); c.has_value()) return c->key;
            // Another rack can still hold the chassis, so the module keeps the
            // chassis it has.
            if (const auto d = this->devices.get(module); d.has_value())
                return d->chassis;
            return std::nullopt;
        }
    ));
}

std::optional<synnax::ni::device::Hardware>
Resources::stored_hardware(const Device &scanned) {
    const auto dev = this->devices.get(match_key(scanned));
    if (!dev.has_value()) return std::nullopt;
    return dev->hardware;
}

namespace {
/// @brief passes each resource that a set series holds to set, and logs the samples
/// that do not parse.
template<typename D, typename F>
void each_set(const x::telem::Series &series, const F &set) {
    for (const auto &sample: series.strings()) {
        x::json::Parser parser(sample);
        const auto res = D::parse(parser);
        if (!parser.ok()) {
            LOG(WARNING) << SCAN_LOG_PREFIX
                         << "failed to parse a set signal: " << parser.error();
            continue;
        }
        set(res);
    }
}
}

void Resources::on_signal(
    const synnax::channel::Channel &ch,
    const x::telem::Series &series
) {
    if (ch.name == synnax::ni::chassis::SET_CHANNEL)
        each_set<synnax::ni::chassis::Chassis>(series, [this](const auto &c) {
            this->chassis.on_set(c);
        });
    else if (ch.name == synnax::ni::device::SET_CHANNEL)
        each_set<synnax::ni::device::Device>(series, [this](const auto &d) {
            this->devices.on_set(d);
        });
    else if (ch.name == synnax::ni::chassis::DELETE_CHANNEL)
        for (const auto &key: series.uuids())
            this->chassis.on_delete(key);
    else if (ch.name == synnax::ni::device::DELETE_CHANNEL)
        for (const auto &key: series.uuids())
            this->devices.on_delete(key);
    else
        throw std::invalid_argument("unknown NI signal channel " + ch.name);
}
}
