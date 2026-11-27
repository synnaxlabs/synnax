// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <mutex>
#include <optional>
#include <string>
#include <utility>

#include "client/cpp/serial/json.gen.h"
#include "x/cpp/json/json.h"

#include "driver/errors/errors.h"
#include "driver/serial/port.h"
#include "driver/serial/serial.h"
#include "driver/transport/errors.h"

namespace driver::serial {
std::pair<Device, x::errors::Error>
retrieve_device(const synnax::device::Client &devices, const std::string &key) {
    auto [dev, err] = devices.retrieve(key);
    if (err) return {Device{}, err};
    x::json::Parser parser(x::json::json(dev.properties));
    auto props = synnax::serial::Properties::parse(parser);
    return {
        Device{.key = dev.key, .name = dev.name, .props = std::move(props)},
        parser.error(),
    };
}

std::optional<std::string> Ports::Holders::take(
    const std::string &path,
    const std::string &id,
    const std::string &name
) {
    std::lock_guard lock(this->mu);
    auto &holder = this->held[path];
    if (holder.claims > 0 && holder.id != id) return holder.name;
    holder.id = id;
    holder.name = name;
    holder.claims++;
    return std::nullopt;
}

void Ports::Holders::release(const std::string &path) {
    std::lock_guard lock(this->mu);
    const auto it = this->held.find(path);
    if (--it->second.claims == 0) this->held.erase(it);
}

Ports::Claim::Claim(std::shared_ptr<Holders> holders, std::string path):
    holders(std::move(holders)), path(std::move(path)) {}

Ports::Claim::~Claim() {
    this->holders->release(this->path);
}

std::pair<std::unique_ptr<Ports::Claim>, x::errors::Error>
Ports::claim_read(const std::string &path, const synnax::task::Task &task) {
    if (const auto holder = this->readers->take(path, task.key.to_string(), task.name))
        return {
            nullptr,
            x::errors::Error(
                driver::errors::CONFIGURATION_ERROR,
                "A serial port runs one read task at a time, and the task '" + *holder +
                    "' reads from " + path + ". Add these entries to that task."
            ),
        };
    return {std::unique_ptr<Claim>(new Claim(this->readers, path)), x::errors::NIL};
}

std::pair<Ports::Lease, x::errors::Error> Ports::acquire(const Device &dev) {
    const auto &path = dev.props.port;
    if (const auto holder = this->devices->take(path, dev.key, dev.name))
        return {
            Lease{},
            x::errors::Error(
                transport::CONFIG_ERROR,
                "The device '" + *holder + "' already uses serial port " + path +
                    ". Two devices cannot share one port. Change the port of one of "
                    "them."
            ),
        };
    Lease lease{.claim = std::unique_ptr<Claim>(new Claim(this->devices, path))};
    auto [conn, err] = this->connections.acquire(
        dev.key,
        dev.props.to_json(),
        {.open = [props = dev.props] {
            auto [port, open_err] = Port::open(props);
            return std::pair<std::unique_ptr<transport::Transport>, x::errors::Error>{
                std::move(port),
                open_err
            };
        }}
    );
    if (err) return {Lease{}, err};
    auto guard = conn->lock();
    if (auto open_err = guard.transport().second.skip(transport::UNREACHABLE_ERROR))
        return {Lease{}, open_err};
    lease.conn = std::move(conn);
    return {std::move(lease), x::errors::NIL};
}
}
