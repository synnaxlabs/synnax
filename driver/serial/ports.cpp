// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstddef>
#include <cstdint>
#include <memory>
#include <mutex>
#include <optional>
#include <span>
#include <string>
#include <unordered_map>
#include <utility>

#include "client/cpp/serial/json.gen.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/telem.h"

#include "driver/errors/errors.h"
#include "driver/serial/native.h"
#include "driver/serial/port.h"
#include "driver/serial/ports.h"
#include "driver/transport/connection.h"
#include "driver/transport/errors.h"
#include "driver/transport/transport.h"

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
    /// @brief Holder is the holder of a key.
    struct Holder {
        /// @brief id is the key of the task or device that holds the key.
        std::string id;
        std::string name;
        /// @brief claims is the number of claims that the holder has on the key.
        std::size_t claims = 0;
    };

    std::mutex mu;
    std::unordered_map<std::string, Holder> held;

public:
    /// @brief claims key for the holder with id.
    /// @returns the name of the holder of key when another holder has it.
    std::optional<std::string>
    take(const std::string &key, const std::string &id, const std::string &name) {
        std::lock_guard lock(this->mu);
        auto &holder = this->held[key];
        if (holder.claims > 0 && holder.id != id) return holder.name;
        if (holder.claims == 0) holder = {.id = id, .name = name};
        holder.claims++;
        return std::nullopt;
    }

    /// @brief releases one claim on key.
    void release(const std::string &key) {
        std::lock_guard lock(this->mu);
        const auto it = this->held.find(key);
        if (--it->second.claims == 0) this->held.erase(it);
    }
};

namespace {
/// @brief Held is an open port and the claim of its device on the port, which ends when
/// the port closes.
class Held final : public transport::Transport {
    std::unique_ptr<Ports::Claim> claim;
    std::unique_ptr<transport::Transport> port;

public:
    Held(
        std::unique_ptr<Ports::Claim> claim,
        std::unique_ptr<transport::Transport> port
    ):
        claim(std::move(claim)), port(std::move(port)) {}

    std::pair<transport::Chunk, x::errors::Error>
    read(const x::telem::TimeSpan timeout) override {
        return this->port->read(timeout);
    }

    x::errors::Error write(
        const std::span<const std::uint8_t> data,
        const x::telem::TimeSpan timeout
    ) override {
        return this->port->write(data, timeout);
    }
};
}

Ports::Ports():
    readers(std::make_shared<Holders>()), devices(std::make_shared<Holders>()) {}

Ports::Claim::Claim(std::shared_ptr<Holders> holders, std::string key):
    holders(std::move(holders)), key(std::move(key)) {}

Ports::Claim::~Claim() {
    this->holders->release(this->key);
}

std::pair<std::unique_ptr<Ports::Claim>, x::errors::Error>
Ports::claim_read(const std::string &device, const synnax::task::Task &task) {
    if (const auto holder = this->readers
                                ->take(device, task.key.to_string(), task.name))
        return {
            nullptr,
            x::errors::Error(
                driver::errors::CONFIGURATION_ERROR,
                "A serial device runs one read task at a time, and the task '" +
                    *holder +
                    "' reads from this device. Add these entries to that task."
            ),
        };
    return {std::unique_ptr<Claim>(new Claim(this->readers, device)), x::errors::NIL};
}

std::pair<std::shared_ptr<transport::Connection>, x::errors::Error>
Ports::acquire(const Device &dev) {
    auto [conn, err] = this->connections.acquire(
        dev.key,
        dev.props.to_json(),
        {.open = [devices = this->devices, dev]()
             -> std::pair<std::unique_ptr<transport::Transport>, x::errors::Error> {
            auto [key, key_err] = native::identity(dev.props.port);
            if (key_err) return {nullptr, key_err};
            if (const auto holder = devices->take(key, dev.key, dev.name))
                return {
                    nullptr,
                    x::errors::Error(
                        transport::CONFIG_ERROR,
                        "The device '" + *holder + "' has serial port " +
                            dev.props.port +
                            " open. Two devices cannot share one port. Change the "
                            "port of one of them."
                    ),
                };
            auto claim = std::unique_ptr<Claim>(new Claim(devices, std::move(key)));
            auto [port, open_err] = Port::open(dev.props);
            if (open_err) return {nullptr, open_err};
            return {
                std::make_unique<Held>(std::move(claim), std::move(port)),
                x::errors::NIL,
            };
        }}
    );
    if (err) return {nullptr, err};
    auto guard = conn->lock();
    if (auto open_err = guard.transport().second.skip(transport::UNREACHABLE_ERROR))
        return {nullptr, open_err};
    return {std::move(conn), x::errors::NIL};
}
}
