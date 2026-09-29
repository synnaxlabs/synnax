// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cctype>
#include <set>
#include <utility>

#include "driver/serial/scan_task.h"

namespace driver::serial {
namespace {
std::string device_key(const synnax::rack::Key rack, const std::string &path) {
    std::string key = "serial_" + std::to_string(rack) + "_";
    for (const char c: path)
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
        .message = "Port available",
        .time = x::telem::TimeStamp::now(),
        .details = {.rack = rack, .device = dev.key},
    };
}
}

Scanner::Scanner(synnax::task::Task task, Lister list):
    task(std::move(task)), list(std::move(list)) {}

common::ScannerConfig Scanner::config() const {
    return {.make = MAKE, .log_prefix = "[" + INTEGRATION_NAME + ".scan_task] "};
}

std::pair<std::vector<synnax::device::Device>, x::errors::Error>
Scanner::scan(const common::ScannerContext &ctx) {
    auto [ports, err] = this->list();
    if (err) return {{}, err};
    std::set<std::string> paths;
    for (const auto &p: ports)
        paths.insert(p.path);
    std::vector<synnax::device::Device> out;
    std::set<std::string> claimed;
    if (ctx.devices != nullptr)
        for (const auto &[_, tracked]: *ctx.devices) {
            const auto it = tracked.properties.find("port");
            if (it == tracked.properties.end() || !it->second.is_string()) continue;
            const auto port = it->second.get<std::string>();
            if (!paths.contains(port)) continue;
            auto dev = tracked;
            set_present(dev, this->task.rack);
            out.push_back(std::move(dev));
            claimed.insert(port);
        }
    for (const auto &p: ports) {
        if (claimed.contains(p.path)) continue;
        synnax::device::Device dev{
            .key = device_key(this->task.rack, p.path),
            .rack = this->task.rack,
            .location = p.path,
            .make = MAKE,
            .model = "Serial port",
            .name = p.name,
            .properties = {{"port", p.path}},
        };
        set_present(dev, this->task.rack);
        out.push_back(std::move(dev));
    }
    return {std::move(out), x::errors::NIL};
}
}
