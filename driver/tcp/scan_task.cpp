// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <utility>

#include "client/cpp/tcp/json.gen.h"

#include "driver/tcp/scan_task.h"

namespace driver::tcp {
Scanner::Scanner(
    synnax::task::Task task,
    const Config &connection,
    std::shared_ptr<bus::Connections> connections
):
    task(std::move(task)),
    connection(connection),
    connections(std::move(connections)) {}

x::errors::Error
Scanner::reach(const std::string &key, const synnax::tcp::Properties &props) {
    if (const auto conn = this->connections->find(key)) return conn->error();
    return Client::open(props, this->connection).second;
}

common::ScannerConfig Scanner::config() const {
    return {.make = MAKE, .log_prefix = "[" + INTEGRATION_NAME + ".scan_task] "};
}

std::pair<std::vector<synnax::device::Device>, x::errors::Error>
Scanner::scan(const common::ScannerContext &ctx) {
    std::vector<synnax::device::Device> out;
    if (ctx.devices == nullptr) return {out, x::errors::NIL};
    for (auto [_, dev]: *ctx.devices) {
        x::json::Parser parser{x::json::json(dev.properties)};
        const auto props = synnax::tcp::Properties::parse(parser);
        synnax::device::Status status{
            .key = synnax::device::status_key(dev),
            .name = dev.name,
            .variant = synnax::status::VARIANT_SUCCESS,
            .message = "Device connected",
            .time = x::telem::TimeStamp::now(),
            .details = {.rack = this->task.rack, .device = dev.key},
        };
        if (!parser.ok()) {
            status.variant = synnax::status::VARIANT_WARNING;
            status.message = "Invalid device properties";
            status.description = parser.error().data;
        } else if (const auto err = this->reach(dev.key, props)) {
            status.variant = synnax::status::VARIANT_WARNING;
            status.message = "Failed to reach device";
            status.description = err.data;
        }
        dev.status = std::move(status);
        out.push_back(std::move(dev));
    }
    return {std::move(out), x::errors::NIL};
}
}
