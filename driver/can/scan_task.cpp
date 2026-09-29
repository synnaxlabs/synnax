// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cctype>
#include <map>
#include <set>
#include <string>
#include <utility>

#include "absl/log/log.h"

#include "client/cpp/can/json.gen.h"

#include "driver/can/factory.h"
#include "driver/can/scan_task.h"

namespace driver::can {
namespace {
std::string device_key(const synnax::rack::Key rack, const Channel &ch) {
    std::string key = "can_" + std::to_string(rack) + "_" + ch.backend + "_";
    for (const char c: ch.name)
        key += std::isalnum(static_cast<unsigned char>(c)) || c == '.' || c == '-'
                 ? c
                 : '_';
    return key;
}

void set_present(synnax::device::Device &dev, const synnax::rack::Key rack) {
    dev.status = synnax::device::Status{
        .key = synnax::device::status_key(dev),
        .name = dev.name,
        .variant = synnax::status::VARIANT_SUCCESS,
        .message = "Channel available",
        .time = x::telem::TimeStamp::now(),
        .details = {.rack = rack, .device = dev.key},
    };
}

/// @returns the backend and channel a tracked device opens, or nullopt when its
/// properties name none.
std::optional<std::pair<std::string, std::string>>
channel_of(const synnax::device::Device &dev) {
    const auto backend = dev.properties.find("backend");
    const auto channel = dev.properties.find("channel");
    if (backend == dev.properties.end() || channel == dev.properties.end() ||
        !backend->second.is_string() || !channel->second.is_string())
        return std::nullopt;
    return std::pair{
        backend->second.get<std::string>(),
        channel->second.get<std::string>()
    };
}
}

Scanner::Scanner(synnax::task::Task task, std::shared_ptr<const Backends> backends):
    task(std::move(task)), backends(std::move(backends)) {}

common::ScannerConfig Scanner::config() const {
    return {.make = MAKE, .log_prefix = "[" + INTEGRATION_NAME + ".scan_task] "};
}

std::pair<std::vector<synnax::device::Device>, x::errors::Error>
Scanner::scan(const common::ScannerContext &ctx) {
    std::map<std::pair<std::string, std::string>, Channel> found;
    for (const auto &[name, backend]: *this->backends) {
        auto [channels, err] = backend->scan();
        if (err) {
            VLOG(1) << this->config().log_prefix << "skipping " << name << ": " << err;
            continue;
        }
        for (auto &ch: channels)
            found.emplace(std::pair{ch.backend, ch.name}, std::move(ch));
    }
    std::vector<synnax::device::Device> out;
    std::set<std::pair<std::string, std::string>> claimed;
    if (ctx.devices != nullptr)
        for (const auto &[_, tracked]: *ctx.devices) {
            const auto ch = channel_of(tracked);
            if (!ch.has_value() || !found.contains(*ch)) continue;
            auto dev = tracked;
            set_present(dev, this->task.rack);
            out.push_back(std::move(dev));
            claimed.insert(*ch);
        }
    for (const auto &[id, ch]: found) {
        if (claimed.contains(id)) continue;
        synnax::can::Properties props;
        props.backend = ch.backend;
        props.channel = ch.name;
        synnax::device::Device dev{
            .key = device_key(this->task.rack, ch),
            .rack = this->task.rack,
            .location = ch.name,
            .make = MAKE,
            .model = ch.backend,
            .name = ch.description.empty() ? ch.name : ch.description,
            .properties = props.to_json().get<x::json::json::object_t>(),
        };
        set_present(dev, this->task.rack);
        out.push_back(std::move(dev));
    }
    return {std::move(out), x::errors::NIL};
}
}
