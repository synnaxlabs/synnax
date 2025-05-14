// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// external
#include "glog/logging.h"
#include "modbus/modbus.h"

/// module
#include "x/cpp/xerrors/errors.h"

/// internal
#include "driver/errors/errors.h"
#include "x/cpp/xjson/xjson.h"

const xerrors::Error CRITICAL_ERROR = driver::CRITICAL_HARDWARE_ERROR.sub("modbus");
const xerrors::Error TEMPORARY_ERROR = driver::TEMPORARY_HARDWARE_ERROR.sub("modbus");

namespace modbus::device {
inline xerrors::Error parse_error(const int code) {
    if (code != -1) return xerrors::NIL;
    const auto err = modbus_strerror(errno);
    return xerrors::Error(CRITICAL_ERROR, err);
}

enum RegisterType {
    InputRegister,
    HoldingRegister,
};

enum BitType {
    Coil,
    DiscreteInput
};

struct Device {
    modbus_t *ctx;

    explicit Device(modbus_t *ctx = nullptr): ctx(ctx) {}

    ~Device() {
        if (ctx != nullptr) { modbus_free(ctx); }
    }

    Device(const Device &) = delete;

    Device &operator=(const Device &) = delete;

    xerrors::Error
    read_bits(BitType t, const int addr, const size_t nb, uint8_t *dest) const {
        if (t == Coil) return parse_error(modbus_read_bits(ctx, addr, nb, dest));

        return parse_error(modbus_read_input_bits(ctx, addr, nb, dest));
    }

    xerrors::Error read_registers(
        RegisterType t,
        const int addr,
        const size_t nb,
        uint16_t *dest
    ) const {
        if (t == HoldingRegister)
            return parse_error(modbus_read_registers(ctx, addr, nb, dest));
        return parse_error(modbus_read_input_registers(ctx, addr, nb, dest));
    }

    xerrors::Error
    write_bits(const int addr, const size_t nb, const uint8_t *src) const {
        return parse_error(modbus_write_bits(ctx, addr, nb, src));
    }

    xerrors::Error
    write_registers(const int addr, const size_t nb, const uint16_t *src) const {
        return parse_error(modbus_write_registers(ctx, addr, nb, src));
    }
};

/// @brief Configuration for a Modbus TCP/IP connection
struct ConnectionConfig {
    /// @brief The hostname or IP address of the Modbus server
    std::string host;
    /// @brief The TCP port of the Modbus server (default is 502)
    uint16_t port;
    /// @brief Whether to swap the byte order within each 16-bit word (endianness)
    bool swap_bytes;
    /// @brief Whether to swap the word order for 32-bit and larger values
    bool swap_words;

    ConnectionConfig() = default;

    ConnectionConfig(
        const std::string &host,
        const uint16_t port,
        const bool swap_bytes = false,
        const bool swap_words = false
    ):
        host(host), port(port), swap_bytes(swap_bytes), swap_words(swap_words) {}

    explicit ConnectionConfig(xjson::Parser parser):
        host(parser.required<std::string>("host")),
        port(parser.required<uint16_t>("port")),
        swap_bytes(parser.required<bool>("swap_bytes")),
        swap_words(parser.required<bool>("swap_words")) {}

    json to_json() const {
        return {
            {"host", host},
            {"port", port},
            {"swap_bytes", swap_bytes},
            {"swap_words", swap_words}
        };
    }
};

class Manager {
    std::unordered_map<std::string, std::weak_ptr<Device>> devices;

public:
    Manager() = default;

    std::pair<std::shared_ptr<Device>, xerrors::Error>
    acquire(const ConnectionConfig &config) {
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

        auto dev = std::make_shared<Device>(ctx);
        devices[config.host] = dev;

        return {dev, xerrors::NIL};
    }
};
}
