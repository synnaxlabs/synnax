// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <mutex>
#include <utility>

#include "glog/logging.h"
#ifdef _WIN32
#include "vendor/libmodbus/modbus/modbus.h"
#else
#include "modbus/modbus.h"
#endif

#include "x/cpp/xerrors/errors.h"
#include "x/cpp/xjson/xjson.h"

#include "driver/errors/errors.h"

const xerrors::Error CRITICAL_ERROR = driver::CRITICAL_HARDWARE_ERROR.sub("modbus");
const xerrors::Error TEMPORARY_ERROR = driver::TEMPORARY_HARDWARE_ERROR.sub("modbus");

namespace modbus::device {
/// @brief parses the xerrors compatible representation of the modbus error code.
inline xerrors::Error parse_error(const int code) {
    if (code != -1) return xerrors::NIL;
    const auto err = modbus_strerror(errno);
    return xerrors::Error(CRITICAL_ERROR, err);
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
    xerrors::Error read_bits(
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
    xerrors::Error read_registers(
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
    xerrors::Error
    write_bits(const int addr, const size_t nb, const uint8_t *src) const {
        return parse_error(modbus_write_bits(ctx, addr, static_cast<int>(nb), src));
    }

    /// @brief writes to the holding registers of the device.
    /// @param addr the address to start writing to.
    /// @param nb the number of registers to write.
    /// @param src the source buffer to write from.
    xerrors::Error
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
    explicit ConnectionConfig(xjson::Parser parser):
        host(parser.field<std::string>("host")),
        port(parser.field<uint16_t>("port")),
        swap_bytes(parser.field<bool>("swap_bytes")),
        swap_words(parser.field<bool>("swap_words")) {}

    /// @brief returns the JSON representation of the configuration.
    [[nodiscard]] json to_json() const {
        return {
            {"host", host},
            {"port", port},
            {"swap_bytes", swap_bytes},
            {"swap_words", swap_words}
        };
    }
};

/// @brief controls access and caches connections to Modbus servers.
class Manager {
    /// @brief mutex to protect access to the devices map.
    std::mutex mu;
    /// @brief the current set of open Modbus servers.
    std::unordered_map<std::string, std::weak_ptr<Device>> devices;

public:
    Manager() = default;

    /// @brief acquires a connection to a Modbus server, returning an error if the
    /// server could not be connected to.
    /// @param config - the configuration for the connection.
    std::pair<std::shared_ptr<Device>, xerrors::Error>
    acquire(const ConnectionConfig &config) {
        std::lock_guard lock(mu);
        const std::string id = config.host + ":" + std::to_string(config.port);
        const auto it = devices.find(id);
        if (it != devices.end()) {
            const auto existing = it->second.lock();
            if (existing != nullptr) return {existing, xerrors::NIL};
            devices.erase(it);
        }

        auto ctx = modbus_new_tcp(config.host.c_str(), config.port);
        if (ctx == nullptr) return {nullptr, parse_error(errno)};

        if (const auto err = parse_error(modbus_connect(ctx))) {
            modbus_free(ctx);
            return {nullptr, err};
        }

        auto dev = std::shared_ptr<Device>(new Device(ctx));
        devices[id] = dev;

        return {dev, xerrors::NIL};
    }
};
}
