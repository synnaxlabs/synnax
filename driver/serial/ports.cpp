// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstddef>
#include <memory>
#include <mutex>
#include <optional>
#include <string>
#include <unordered_map>
#include <utility>

#include "client/cpp/serial/json.gen.h"
#include "x/cpp/json/json.h"

#include "driver/errors/errors.h"
#include "driver/serial/native.h"
#include "driver/serial/port.h"
#include "driver/serial/ports.h"
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

class Ports::Holders {
public:
    /// @brief Holder is the holder of a port.
    struct Holder {
        /// @brief id is the key of the task or device that holds the port.
        std::string id;
        std::string name;
        /// @brief path is the path through which the holder claimed the port.
        std::string path;
        /// @brief claims is the number of claims that the holder has on the port.
        std::size_t claims = 0;
    };

    /// @brief claims the port with key for the holder with id, which claims it
    /// through path.
    /// @returns the holder of the port when another holder has it.
    std::optional<Holder> take(
        const std::string &key,
        const std::string &path,
        const std::string &id,
        const std::string &name
    );

    /// @brief releases one claim on the port with key.
    void release(const std::string &key);

private:
    std::mutex mu;
    std::unordered_map<std::string, Holder> held;
};

std::optional<Ports::Holders::Holder> Ports::Holders::take(
    const std::string &key,
    const std::string &path,
    const std::string &id,
    const std::string &name
) {
    std::lock_guard lock(this->mu);
    auto &holder = this->held[key];
    if (holder.claims > 0 && holder.id != id) return holder;
    if (holder.claims == 0) holder = {.id = id, .name = name, .path = path};
    holder.claims++;
    return std::nullopt;
}

void Ports::Holders::release(const std::string &key) {
    std::lock_guard lock(this->mu);
    const auto it = this->held.find(key);
    if (--it->second.claims == 0) this->held.erase(it);
}

Ports::Ports():
    readers(std::make_shared<Holders>()), devices(std::make_shared<Holders>()) {}

Ports::Claim::Claim(std::shared_ptr<Holders> holders, std::string key):
    holders(std::move(holders)), key(std::move(key)) {}

Ports::Claim::~Claim() {
    this->holders->release(this->key);
}

std::pair<std::unique_ptr<Ports::Claim>, x::errors::Error>
Ports::claim_read(const std::string &path, const synnax::task::Task &task) {
    auto key = native::identity(path);
    if (const auto holder = this->readers
                                ->take(key, path, task.key.to_string(), task.name))
        return {
            nullptr,
            x::errors::Error(
                driver::errors::CONFIGURATION_ERROR,
                "A serial port runs one read task at a time, and the task '" +
                    holder->name + "' reads from " + holder->path +
                    ". Add these entries to that task."
            ),
        };
    return {
        std::unique_ptr<Claim>(new Claim(this->readers, std::move(key))),
        x::errors::NIL,
    };
}

std::pair<Ports::Lease, x::errors::Error> Ports::acquire(const Device &dev) {
    const auto &path = dev.props.port;
    auto key = native::identity(path);
    if (const auto holder = this->devices->take(key, path, dev.key, dev.name))
        return {
            Lease{},
            x::errors::Error(
                transport::CONFIG_ERROR,
                "The device '" + holder->name + "' already uses serial port " +
                    holder->path +
                    ". Two devices cannot share one port. Change the port of one of "
                    "them."
            ),
        };
    Lease lease{
        .claim = std::unique_ptr<Claim>(new Claim(this->devices, std::move(key))),
    };
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
