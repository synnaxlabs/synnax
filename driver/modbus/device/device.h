// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <utility>

#include "glog/logging.h"
#ifdef _WIN32
#include "vendor/libmodbus/modbus/modbus.h"
#else
#include "modbus/modbus.h"
#endif

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"

#include "driver/errors/errors.h"

namespace driver::modbus::device {
const x::errors::Error CRITICAL_ERROR = errors::CRITICAL_HARDWARE_ERROR.sub("modbus");
const x::errors::Error TEMPORARY_ERROR = errors::TEMPORARY_HARDWARE_ERROR.sub("modbus");

/// @brief parses the x::errors compatible representation of the modbus error code.
inline x::errors::Error parse_error(const int code) {
    if (code != -1) return x::errors::NIL;
    const auto err = modbus_strerror(errno);
    return x::errors::Error(CRITICAL_ERROR, err);
}

enum RegisterType {
    InputRegister,
    HoldingRegister,
};

enum BitType { Coil, DiscreteInput };

struct Device {
    modbus_t *ctx;

    explicit Device(modbus_t *ctx = nullptr): ctx(ctx) {}

    ~Device() {
        if (ctx != nullptr) { modbus_free(ctx); }
    }

    Device(const Device &) = delete;

    Device &operator=(const Device &) = delete;

    /// @brief reads from the bit-address space of the device (coils and discrete
    /// inputs).
    /// @param input_type the input type to read from - either ByteType::Coil or
    /// ByteType::DiscreteInput.
    /// @param addr the address to start reading from.
    /// @param nb the number of bits to read.
    /// @param dest the destination buffer to read into.
    x::errors::Error read_bits(
        const BitType input_type,
        const int addr,
        const size_t nb,
        uint8_t *dest
    ) const {
        if (input_type == Coil)
            return parse_error(modbus_read_bits(ctx, addr, static_cast<int>(nb), dest));
        return parse_error(
            modbus_read_input_bits(ctx, addr, static_cast<int>(nb), dest)
        );
    }

    /// @brief reads from the register-address space of the device (holding and input
    /// registers).
    /// @brief input_type the input type to read from - either
    /// RegisterType::HoldingRegister or ByteType::InputRegister.
    x::errors::Error read_registers(
        const RegisterType t,
        const int addr,
        const size_t nb,
        uint16_t *dest
    ) const {
        if (t == HoldingRegister)
            return parse_error(
                modbus_read_registers(ctx, addr, static_cast<int>(nb), dest)
            );
        return parse_error(
            modbus_read_input_registers(ctx, addr, static_cast<int>(nb), dest)
        );
    }

    /// @brief writes to the coils of the device.
    /// @param addr the address to start writing to.
    /// @param nb the number of bits to write.
    /// @param src the source buffer to write from.
    x::errors::Error
    write_bits(const int addr, const size_t nb, const uint8_t *src) const {
        return parse_error(modbus_write_bits(ctx, addr, static_cast<int>(nb), src));
    }

    /// @brief writes to the holding registers of the device.
    /// @param addr the address to start writing to.
    /// @param nb the number of registers to write.
    /// @param src the source buffer to write from.
    x::errors::Error
    write_registers(const int addr, const size_t nb, const uint16_t *src) const {
        return parse_error(
            modbus_write_registers(ctx, addr, static_cast<int>(nb), src)
        );
    }
};

/// @brief Configuration for a Modbus TCP/IP connection
struct ConnectionConfig {
    /// @brief The hostname or IP address of the Modbus server
    std::string host;
    /// @brief The TCP port of the Modbus server (default is 502)
    uint16_t port = 0;
    /// @brief Whether to swap the byte order within each 16-bit word (endianness)
    bool swap_bytes = false;
    /// @brief Whether to swap the word order for 32-bit and larger values
    bool swap_words = false;

    ConnectionConfig() = default;

    ConnectionConfig(
        std::string host,
        const uint16_t port,
        const bool swap_bytes = false,
        const bool swap_words = false
    ):
        host(std::move(host)),
        port(port),
        swap_bytes(swap_bytes),
        swap_words(swap_words) {}

    /// @brief constructs a ConnectionConfig from a JSON object.
    explicit ConnectionConfig(x::json::Parser parser):
        host(parser.field<std::string>("host")),
        port(parser.field<uint16_t>("port")),
        swap_bytes(parser.field<bool>("swap_bytes")),
        swap_words(parser.field<bool>("swap_words")) {}

    /// @brief returns the JSON representation of the configuration.
    [[nodiscard]] x::json::json to_json() const {
        return {
            {"host", host},
            {"port", port},
            {"swap_bytes", swap_bytes},
            {"swap_words", swap_words}
        };
    }
};

/// @brief creates connections to Modbus servers.
class Manager {
public:
    Manager() = default;

    /// @brief acquires a new connection to a Modbus server, returning an error if the
    /// server could not be connected to.
    /// @param config - the configuration for the connection.
    /// @note Each call creates a fresh connection. Connections are not cached or shared
    /// to avoid thread-safety issues (libmodbus is not thread-safe) and stale
    /// connection problems when servers restart.
    std::pair<std::shared_ptr<Device>, x::errors::Error>
    acquire(const ConnectionConfig &config) {
        auto ctx = modbus_new_tcp(config.host.c_str(), config.port);
        if (ctx == nullptr)
            return {nullptr, x::errors::Error(CRITICAL_ERROR, modbus_strerror(errno))};

        if (auto err = parse_error(modbus_connect(ctx))) {
            modbus_free(ctx);
            return {nullptr, err};
        }

        return {std::make_shared<Device>(ctx), x::errors::NIL};
    }
};
}
