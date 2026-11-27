// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>
#include <memory>
#include <span>
#include <string>
#include <utility>

#include "client/cpp/serial/json.gen.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/telem.h"

#include "driver/serial/native.h"
#include "driver/serial/port.h"
#include "driver/serial/ports.h"
#include "driver/transport/connection.h"
#include "driver/transport/errors.h"
#include "driver/transport/transport.h"

namespace driver::serial {
namespace {
/// @brief Held is an open port and the claim of its device on the port, which ends when
/// the port closes.
class Held final : public transport::Transport {
    std::unique_ptr<bus::Holders::Claim> claim;
    std::unique_ptr<transport::Transport> port;

public:
    Held(
        std::unique_ptr<bus::Holders::Claim> claim,
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

std::pair<std::shared_ptr<transport::Connection>, x::errors::Error>
Ports::acquire(const Device &dev) {
    return this->connections.acquire(
        dev.key,
        dev.props.to_json(),
        {.open = [devices = this->devices, dev]()
             -> std::pair<std::unique_ptr<transport::Transport>, x::errors::Error> {
            auto [key, key_err] = native::identity(dev.props.port);
            if (key_err) return {nullptr, key_err};
            auto [claim, holder] = bus::Holders::take(devices, key, dev.key, dev.name);
            if (holder.has_value())
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
            auto [port, open_err] = Port::open(dev.props);
            if (open_err) return {nullptr, open_err};
            return {
                std::make_unique<Held>(std::move(claim), std::move(port)),
                x::errors::NIL,
            };
        }}
    );
}
}
