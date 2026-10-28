// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <utility>

#include "absl/log/log.h"
#ifdef _WIN32
#include "vendor/libmodbus/modbus/modbus.h"
#else
#include "modbus/modbus.h"
#endif

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"

#include "driver/errors/errors.h"

namespace driver::modbus::device {
/// @brief a fault in the configuration or the device that a retry cannot fix, such
/// as an illegal function or address.
const x::errors::Error CRITICAL_ERROR = errors::CRITICAL_HARDWARE_ERROR.sub("modbus");
/// @brief a fault that can clear on retry, such as a busy server or a gateway that
/// cannot reach its target.
const x::errors::Error TEMPORARY_ERROR = errors::TEMPORARY_HARDWARE_ERROR.sub("modbus");
/// @brief the connection failed, timed out, or lost sync. The device reconnects on
/// the next request.
const x::errors::Error UNREACHABLE = TEMPORARY_ERROR.sub("unreachable");

/// @brief classifies a libmodbus or system error number.
/// @param errnum the value of errno after a failed libmodbus call.
/// @returns CRITICAL_ERROR, TEMPORARY_ERROR, or UNREACHABLE, carrying the message
/// from modbus_strerror.
inline x::errors::Error classify(const int errnum) {
    const std::string message = modbus_strerror(errnum);
    switch (errnum) {
        case EMBXILFUN:
        case EMBXILADD:
        case EMBXILVAL:
        case EMBXNACK:
        case EMBXMEMPAR:
        case EMBMDATA:
        case EINVAL:
        case ENOMEM:
            return {CRITICAL_ERROR, message};
        case EMBXSFAIL:
        case EMBXACK:
        case EMBXSBUSY:
        case EMBXGPATH:
        case EMBXGTAR:
        case EMBBADCRC:
            return {TEMPORARY_ERROR, message};
        default:
            return {UNREACHABLE, message};
    }
}

/// @brief register tables of a Modbus server.
enum RegisterType {
    InputRegister,
    HoldingRegister,
};

/// @brief bit tables of a Modbus server.
enum BitType { Coil, DiscreteInput };

/// @brief a connection to one Modbus TCP server. After a request fails with
/// UNREACHABLE, the device closes its socket and reconnects on the next request.
/// Not safe for concurrent use.
class Device {
    modbus_t *ctx;
    bool connected = false;

    /// @brief connects if needed, runs the request, and closes the socket when the
    /// request leaves the connection unusable.
    template<typename Request>
    x::errors::Error exec(const Request &request) {
        if (!this->connected) {
            if (modbus_connect(this->ctx) == -1) return classify(errno);
            this->connected = true;
        }
        if (request() != -1) return x::errors::NIL;
        auto err = classify(errno);
        if (err.matches(UNREACHABLE)) {
            modbus_close(this->ctx);
            this->connected = false;
        }
        return err;
    }

public:
    /// @brief takes ownership of a context created by modbus_new_tcp_pi.
    /// @param ctx the libmodbus context.
    /// @param connected whether the context has an open connection.
    explicit Device(modbus_t *ctx, const bool connected):
        ctx(ctx), connected(connected) {}

    ~Device() {
        modbus_close(this->ctx);
        modbus_free(this->ctx);
    }

    Device(const Device &) = delete;

    Device &operator=(const Device &) = delete;

    /// @brief reads from the bit-address space of the device.
    /// @param type the table to read from.
    /// @param addr the address to start reading from.
    /// @param nb the number of bits to read.
    /// @param dest the destination buffer, at least nb bytes long.
    x::errors::Error
    read_bits(const BitType type, const int addr, const size_t nb, uint8_t *dest) {
        const int n = static_cast<int>(nb);
        return this->exec([&] {
            if (type == Coil) return modbus_read_bits(this->ctx, addr, n, dest);
            return modbus_read_input_bits(this->ctx, addr, n, dest);
        });
    }

    /// @brief reads from the register-address space of the device.
    /// @param type the table to read from.
    /// @param addr the address to start reading from.
    /// @param nb the number of registers to read.
    /// @param dest the destination buffer, at least nb registers long.
    x::errors::Error read_registers(
        const RegisterType type,
        const int addr,
        const size_t nb,
        uint16_t *dest
    ) {
        const int n = static_cast<int>(nb);
        return this->exec([&] {
            if (type == HoldingRegister)
                return modbus_read_registers(this->ctx, addr, n, dest);
            return modbus_read_input_registers(this->ctx, addr, n, dest);
        });
    }

    /// @brief writes to the coils of the device with function code 15.
    /// @param addr the address to start writing to.
    /// @param nb the number of bits to write.
    /// @param src the source buffer, at least nb bytes long.
    x::errors::Error write_bits(const int addr, const size_t nb, const uint8_t *src) {
        return this->exec([&] {
            return modbus_write_bits(this->ctx, addr, static_cast<int>(nb), src);
        });
    }

    /// @brief writes to the holding registers of the device with function code 16.
    /// @param addr the address to start writing to.
    /// @param nb the number of registers to write.
    /// @param src the source buffer, at least nb registers long.
    x::errors::Error
    write_registers(const int addr, const size_t nb, const uint16_t *src) {
        return this->exec([&] {
            return modbus_write_registers(this->ctx, addr, static_cast<int>(nb), src);
        });
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
        const auto port = std::to_string(config.port);
        auto *ctx = modbus_new_tcp_pi(config.host.c_str(), port.c_str());
        if (ctx == nullptr) return {nullptr, classify(errno)};
        if (modbus_connect(ctx) == -1) {
            auto err = classify(errno);
            modbus_free(ctx);
            return {nullptr, err};
        }
        return {std::make_shared<Device>(ctx, true), x::errors::NIL};
    }
};
}
