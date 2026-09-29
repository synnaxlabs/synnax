// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <functional>
#include <memory>
#include <span>
#include <utility>

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "driver/transport/transport.h"

namespace driver::bus {
/// @brief a connection that bus tasks read from and write to. Calls must not overlap,
/// but any thread may make them. Tasks share one through a Connection.
class Transport {
public:
    virtual ~Transport() = default;

    /// @brief blocks until bytes arrive or the timeout elapses.
    /// @returns the bytes that arrived, valid until the next call, or an empty chunk on
    /// timeout. transport::UNREACHABLE_ERROR when the connection fails.
    virtual std::pair<transport::Chunk, x::errors::Error>
    read(x::telem::TimeSpan timeout) = 0;

    /// @brief blocks until all of data is written or the timeout elapses.
    /// @returns transport::UNREACHABLE_ERROR when the connection fails.
    virtual x::errors::Error
    write(std::span<const std::uint8_t> data, x::telem::TimeSpan timeout) = 0;
};

/// @brief opens a new transport to a device. A Connection calls it on first use and
/// again after the transport fails.
/// @returns transport::UNREACHABLE_ERROR when the device cannot be reached, and
/// transport::CONFIG_ERROR when its properties are invalid.
using Opener = std::function<std::pair<std::unique_ptr<Transport>, x::errors::Error>()>;

/// @brief adapts a serial port, TCP client, or UDP socket to Transport.
template<typename T>
class Adapter final : public Transport {
    std::unique_ptr<T> conn;

public:
    explicit Adapter(std::unique_ptr<T> conn): conn(std::move(conn)) {}

    ~Adapter() override { this->conn->close(); }

    std::pair<transport::Chunk, x::errors::Error>
    read(const x::telem::TimeSpan timeout) override {
        return this->conn->read(timeout);
    }

    x::errors::Error write(
        const std::span<const std::uint8_t> data,
        const x::telem::TimeSpan timeout
    ) override {
        return this->conn->write(data, timeout);
    }
};

/// @returns an Opener that opens T with props through T::open.
template<typename T, typename Properties>
Opener opener(Properties props) {
    return [props = std::move(props)]()
               -> std::pair<std::unique_ptr<Transport>, x::errors::Error> {
        auto [conn, err] = T::open(props);
        if (err) return {nullptr, err};
        return {std::make_unique<Adapter<T>>(std::move(conn)), x::errors::NIL};
    };
}
}
