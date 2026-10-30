// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <chrono>
#include <condition_variable>
#include <cstdint>
#include <memory>
#include <mutex>
#include <optional>
#include <string>
#include <thread>
#include <unordered_map>
#include <utility>

#ifdef _WIN32
#include "vendor/libmodbus/modbus/modbus.h"
#else
#include "modbus/modbus.h"
#endif

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/telem.h"

#include "driver/errors/errors.h"

namespace driver::modbus::device {
/// @brief a fault that a retry cannot fix, such as an illegal function or address.
const x::errors::Error CRITICAL_ERROR = errors::CRITICAL_HARDWARE_ERROR.sub("modbus");
/// @brief a fault that can clear on retry, such as a busy server or a gateway that
/// cannot reach its target.
const x::errors::Error TEMPORARY_ERROR = errors::TEMPORARY_HARDWARE_ERROR.sub("modbus");
/// @brief the unit did not answer: the connection failed, timed out, or lost sync, or
/// the unit is demoted. The next request after a lost connection reconnects.
const x::errors::Error UNREACHABLE = TEMPORARY_ERROR.sub("unreachable");

/// @brief the timed-out attempts in a row that demote a unit.
const std::size_t DEMOTE_AFTER = 3;
/// @brief the default time that the requests of a demoted unit fail at once.
const x::telem::TimeSpan DEMOTION = x::telem::SECOND * 10;
/// @brief the default time that a probe waits for its turn on a busy connection.
const x::telem::TimeSpan PROBE_WAIT = x::telem::SECOND * 10;

/// @brief classifies a libmodbus or system error number.
/// @param errnum the value of errno after a failed libmodbus call.
/// @returns CRITICAL_ERROR, TEMPORARY_ERROR, or UNREACHABLE, carrying the message from
/// modbus_strerror.
inline x::errors::Error classify(const int errnum) {
    const std::string message = modbus_strerror(errnum);
    switch (errnum) {
        case EMBXILFUN:
        case EMBXILADD:
        case EMBXILVAL:
        case EMBMDATA:
        case EINVAL:
        case ENOMEM:
            return {CRITICAL_ERROR, message};
        case EMBXSFAIL:
        case EMBXACK:
        case EMBXSBUSY:
        case EMBXNACK:
        case EMBXMEMPAR:
        case EMBXGPATH:
        case EMBXGTAR:
        case EMBBADCRC:
            return {TEMPORARY_ERROR, message};
        default:
            return {UNREACHABLE, message};
    }
}

/// @returns true if the error number is an exception response from the unit itself.
/// Gateway exceptions 0A and 0B mean the unit did not answer.
inline bool answered(const int errnum) {
    switch (errnum) {
        case EMBXILFUN:
        case EMBXILADD:
        case EMBXILVAL:
        case EMBXSFAIL:
        case EMBXACK:
        case EMBXSBUSY:
        case EMBXNACK:
        case EMBXMEMPAR:
            return true;
        default:
            return false;
    }
}

/// @brief register tables of a Modbus server.
enum RegisterType {
    InputRegister,
    HoldingRegister,
};

/// @brief bit tables of a Modbus server.
enum BitType { Coil, DiscreteInput };

/// @brief the function codes a device writes with.
enum class WriteCodes {
    /// @brief FC15 and FC16, one request for each run of adjacent addresses.
    Multiple,
    /// @brief FC05 and FC06, one request for each coil or register.
    Single,
};

/// @brief the endpoint of a Modbus TCP server and the unit a device addresses on it.
struct ConnectionConfig {
    /// @brief the hostname, IPv4 address, or IPv6 address of the server.
    std::string host;
    /// @brief the TCP port of the server.
    std::uint16_t port = 0;
    /// @brief whether to swap the byte order within each 16-bit word.
    bool swap_bytes = false;
    /// @brief whether to swap the word order of 32-bit and larger values.
    bool swap_words = false;
    /// @brief the unit the device addresses. 255 addresses the server itself.
    std::uint8_t unit_id = MODBUS_TCP_SLAVE;
    /// @brief the connect and response timeout.
    x::telem::TimeSpan timeout = x::telem::SECOND;
    /// @brief the immediate retries of a request after a temporary fault.
    std::size_t retries = 1;

    ConnectionConfig() = default;

    ConnectionConfig(
        std::string host,
        const std::uint16_t port,
        const bool swap_bytes = false,
        const bool swap_words = false
    ):
        host(std::move(host)),
        port(port),
        swap_bytes(swap_bytes),
        swap_words(swap_words) {}

    /// @brief parses the connection from its JSON representation. Absent optional
    /// fields take their defaults.
    explicit ConnectionConfig(x::json::Parser parser):
        host(parser.field<std::string>("host")),
        port(parser.field<std::uint16_t>("port")),
        swap_bytes(parser.field<bool>("swap_bytes")),
        swap_words(parser.field<bool>("swap_words")) {
        if (const auto transport = parser.field<std::string>("transport", "tcp");
            transport != "tcp")
            parser.field_err("transport", "unsupported transport: " + transport);
        const auto unit_id = parser.field<int>("unit_id", MODBUS_TCP_SLAVE);
        if (unit_id < 0 || unit_id > 255)
            parser.field_err("unit_id", "must be between 0 and 255");
        else
            this->unit_id = static_cast<std::uint8_t>(unit_id);
        const auto timeout_ms = parser.field<int>("timeout_ms", 1000);
        if (timeout_ms < 1 || timeout_ms > 60000)
            parser.field_err("timeout_ms", "must be between 1 and 60000");
        else
            this->timeout = x::telem::MILLISECOND * timeout_ms;
        const auto retries = parser.field<int>("retries", 1);
        if (retries < 0 || retries > 10)
            parser.field_err("retries", "must be between 0 and 10");
        else
            this->retries = static_cast<std::size_t>(retries);
    }

    /// @returns the JSON representation of the connection.
    [[nodiscard]] x::json::json to_json() const {
        return {
            {"transport", "tcp"},
            {"host", host},
            {"port", port},
            {"swap_bytes", swap_bytes},
            {"swap_words", swap_words},
            {"unit_id", unit_id},
            {"timeout_ms", static_cast<int>(timeout.milliseconds())},
            {"retries", retries},
        };
    }

    /// @returns the key of the server that devices on it share a connection to.
    [[nodiscard]] std::string endpoint() const {
        return host + ":" + std::to_string(port);
    }
};

/// @brief how a device plans and paces its requests.
struct RequestConfig {
    /// @brief the most registers one read covers.
    std::size_t max_registers = MODBUS_MAX_READ_REGISTERS;
    /// @brief the most unused addresses one read can span between two channels.
    std::size_t max_gap = MODBUS_MAX_READ_REGISTERS;
    /// @brief the function codes writes use.
    WriteCodes write_codes = WriteCodes::Multiple;
    /// @brief the wait between the end of one request on the connection and the start
    /// of the next.
    x::telem::TimeSpan delay;

    RequestConfig() = default;

    /// @brief parses the requests from their JSON representation. A missing object
    /// takes every default.
    explicit RequestConfig(x::json::Parser parser) {
        const auto max_registers = parser.field<int>(
            "max_registers",
            MODBUS_MAX_READ_REGISTERS
        );
        if (max_registers < 1 || max_registers > MODBUS_MAX_READ_REGISTERS)
            parser.field_err("max_registers", "must be between 1 and 125");
        else
            this->max_registers = static_cast<std::size_t>(max_registers);
        const auto max_gap = parser.field<int>("max_gap", MODBUS_MAX_READ_REGISTERS);
        if (max_gap < 0 || max_gap > 65535)
            parser.field_err("max_gap", "must be between 0 and 65535");
        else
            this->max_gap = static_cast<std::size_t>(max_gap);
        if (const auto codes = parser.field<std::string>("write_codes", "multiple");
            codes == "single")
            this->write_codes = WriteCodes::Single;
        else if (codes != "multiple")
            parser.field_err("write_codes", "must be multiple or single");
        const auto delay_ms = parser.field<int>("delay_ms", 0);
        if (delay_ms < 0 || delay_ms > 5000)
            parser.field_err("delay_ms", "must be between 0 and 5000");
        else
            this->delay = x::telem::MILLISECOND * delay_ms;
    }
};

/// @brief the properties of a Modbus server device.
struct Properties {
    /// @brief the endpoint and unit of the device.
    ConnectionConfig connection;
    /// @brief how the device plans and paces its requests.
    RequestConfig requests;

    Properties() = default;

    /// @brief parses the properties from their JSON representation.
    explicit Properties(const x::json::Parser &parser):
        connection(parser.child("connection")),
        requests(parser.optional_child("requests")) {}
};

/// @brief a connection to one Modbus TCP endpoint, shared by every device on it.
/// Requests take turns. A waiting write goes before waiting reads, but not twice in a
/// row, so reads keep their turns under steady writes. Safe for concurrent use.
class Connection {
public:
    /// @brief the clock of request, reply, and demotion times.
    using Clock = std::chrono::steady_clock;

private:
    /// @brief the state of one unit behind the endpoint.
    struct Unit {
        /// @brief the timed-out attempts since the unit last answered.
        std::size_t timeouts = 0;
        /// @brief requests to the unit fail at once until this time.
        Clock::time_point demoted_until;
        /// @brief when the unit last answered, an exception included.
        Clock::time_point last_reply;
    };

    modbus_t *ctx;
    /// @brief how long the requests of a demoted unit fail at once.
    x::telem::TimeSpan demotion;
    /// @brief guards busy, the waiting counts, write_went_ahead, and units.
    std::mutex mu;
    std::condition_variable released;
    /// @brief whether a request holds the turn.
    bool busy = false;
    std::size_t waiting_writes = 0;
    std::size_t waiting_reads = 0;
    /// @brief whether the last turn went to a write while a read waited.
    bool write_went_ahead = false;
    std::unordered_map<std::uint8_t, Unit> units;
    /// @brief whether the socket is open. Touched only while holding the turn.
    bool connected = false;
    /// @brief whether a late reply to a timed-out request can still arrive. Touched
    /// only while holding the turn.
    bool stale = false;
    /// @brief when the last request ended. Touched only while holding the turn.
    Clock::time_point last_request;

    /// @brief holds the connection for one request and its retries.
    class Turn {
        Connection &conn;

    public:
        /// @brief UNREACHABLE if the wait ran out before the turn came, and NIL if the
        /// request holds the turn.
        x::errors::Error err = x::errors::NIL;

        /// @param wait how long to wait for the turn. Waits without limit if empty.
        Turn(
            Connection &conn,
            const bool write,
            const std::optional<x::telem::TimeSpan> &wait
        ):
            conn(conn) {
            std::unique_lock lock(conn.mu);
            auto &waiting = write ? conn.waiting_writes : conn.waiting_reads;
            waiting++;
            const auto ready = [&] {
                if (conn.busy) return false;
                if (write) return conn.waiting_reads == 0 || !conn.write_went_ahead;
                return conn.waiting_writes == 0 || conn.write_went_ahead;
            };
            if (!wait.has_value())
                conn.released.wait(lock, ready);
            else if (!conn.released.wait_for(lock, wait->chrono(), ready))
                this->err = {
                    UNREACHABLE,
                    "other requests kept the connection busy for " + wait->to_string()
                };
            waiting--;
            if (this->err) {
                // One less waiting request can let another take the turn.
                lock.unlock();
                conn.released.notify_all();
                return;
            }
            conn.write_went_ahead = write && conn.waiting_reads > 0;
            conn.busy = true;
        }

        ~Turn() {
            if (this->err) return;
            {
                std::lock_guard lock(conn.mu);
                conn.busy = false;
            }
            conn.released.notify_all();
        }

        Turn(const Turn &) = delete;
        Turn &operator=(const Turn &) = delete;
    };

    /// @brief sends one attempt of a request.
    /// @returns 0 on success, or the error number.
    template<typename Request>
    int attempt(const ConnectionConfig &cfg, const Request &request) {
        const auto timeout = std::chrono::duration_cast<std::chrono::microseconds>(
            cfg.timeout.chrono()
        );
        modbus_set_response_timeout(
            this->ctx,
            static_cast<std::uint32_t>(timeout.count() / 1000000),
            static_cast<std::uint32_t>(timeout.count() % 1000000)
        );
        if (!this->connected) {
            if (modbus_connect(this->ctx) == -1) return errno;
            this->connected = true;
            this->stale = false;
        }
        if (this->stale) {
            modbus_flush(this->ctx);
            this->stale = false;
        }
        if (modbus_set_slave(this->ctx, cfg.unit_id) == -1) return errno;
        if (request(this->ctx) != -1) return 0;
        const int errnum = errno;
        if (errnum == ETIMEDOUT)
            this->stale = true;
        else if (classify(errnum).matches(UNREACHABLE)) {
            modbus_close(this->ctx);
            this->connected = false;
        }
        return errnum;
    }

    /// @returns UNREACHABLE if the unit is demoted, and NIL otherwise.
    x::errors::Error demoted(const std::uint8_t unit_id) {
        std::lock_guard lock(this->mu);
        if (Clock::now() >= this->units[unit_id].demoted_until) return x::errors::NIL;
        return {
            UNREACHABLE,
            "unit " + std::to_string(unit_id) + " timed out " +
                std::to_string(DEMOTE_AFTER) +
                " times in a row, so its requests fail for " +
                this->demotion.to_string()
        };
    }

    /// @brief records a reply from the unit.
    void replied(const std::uint8_t unit_id) {
        std::lock_guard lock(this->mu);
        auto &unit = this->units[unit_id];
        unit.timeouts = 0;
        unit.last_reply = Clock::now();
    }

    /// @brief records a timed-out attempt and demotes the unit after DEMOTE_AFTER in a
    /// row.
    /// @returns true if the unit is now demoted.
    bool timed_out(const std::uint8_t unit_id) {
        std::lock_guard lock(this->mu);
        auto &unit = this->units[unit_id];
        if (++unit.timeouts < DEMOTE_AFTER) return false;
        unit.timeouts = 0;
        unit.demoted_until = Clock::now() + this->demotion.chrono();
        return true;
    }

public:
    /// @brief takes ownership of a context created by modbus_new_tcp_pi.
    /// @param demotion how long the requests of a demoted unit fail at once.
    Connection(modbus_t *ctx, const x::telem::TimeSpan demotion):
        ctx(ctx), demotion(demotion) {}

    ~Connection() {
        modbus_close(this->ctx);
        modbus_free(this->ctx);
    }

    Connection(const Connection &) = delete;
    Connection &operator=(const Connection &) = delete;

    /// @brief runs a request for a unit, connecting first if needed. A temporary fault
    /// retries at once, up to cfg.retries times. After a timeout, the next request
    /// first discards any late reply. Any other lost connection closes the socket. A
    /// unit with DEMOTE_AFTER timed-out attempts in a row is demoted.
    /// @param cfg the unit, timeout, and retries of the request.
    /// @param delay the wait after the previous request on the connection.
    /// @param write whether the request goes before waiting reads.
    /// @param request sends the request on the context and returns -1 on failure.
    /// @param wait how long to wait for the turn. Waits without limit if empty.
    /// @returns NIL, UNREACHABLE if other requests kept the connection busy for wait,
    /// or the classified error of the last attempt.
    template<typename Request>
    x::errors::Error exec(
        const ConnectionConfig &cfg,
        const x::telem::TimeSpan delay,
        const bool write,
        const Request &request,
        const std::optional<x::telem::TimeSpan> &wait = std::nullopt
    ) {
        Turn turn(*this, write, wait);
        if (turn.err) return turn.err;
        if (auto err = this->demoted(cfg.unit_id)) return err;
        for (std::size_t i = 0;; i++) {
            if (delay > x::telem::TimeSpan(0))
                std::this_thread::sleep_until(this->last_request + delay.chrono());
            const int errnum = this->attempt(cfg, request);
            this->last_request = Clock::now();
            if (errnum == 0 || answered(errnum)) this->replied(cfg.unit_id);
            if (errnum == 0) return x::errors::NIL;
            if (errnum == ETIMEDOUT && this->timed_out(cfg.unit_id))
                return this->demoted(cfg.unit_id);
            auto err = classify(errnum);
            if (!err.matches(TEMPORARY_ERROR) || i >= cfg.retries) return err;
        }
    }

    /// @returns when the unit last answered a request, an exception included.
    Clock::time_point last_reply(const std::uint8_t unit_id) {
        std::lock_guard lock(this->mu);
        return this->units[unit_id].last_reply;
    }
};

/// @brief one unit on a shared connection. Safe for concurrent use.
class Device {
    std::shared_ptr<Connection> conn;
    ConnectionConfig cfg;
    RequestConfig requests;

    template<typename Request>
    x::errors::Error exec(const bool write, const Request &request) {
        return this->conn->exec(this->cfg, this->requests.delay, write, request);
    }

public:
    /// @brief creates a device that sends its requests on a shared connection.
    Device(
        std::shared_ptr<Connection> conn,
        ConnectionConfig cfg,
        RequestConfig requests
    ):
        conn(std::move(conn)), cfg(std::move(cfg)), requests(requests) {}

    /// @brief reads from the bit-address space of the device.
    /// @param type the table to read from.
    /// @param addr the address to start reading from.
    /// @param nb the number of bits to read.
    /// @param dest the destination buffer, at least nb bytes long.
    x::errors::Error
    read_bits(const BitType type, const int addr, const size_t nb, uint8_t *dest) {
        const int n = static_cast<int>(nb);
        return this->exec(false, [&](modbus_t *ctx) {
            if (type == Coil) return modbus_read_bits(ctx, addr, n, dest);
            return modbus_read_input_bits(ctx, addr, n, dest);
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
        return this->exec(false, [&](modbus_t *ctx) {
            if (type == HoldingRegister)
                return modbus_read_registers(ctx, addr, n, dest);
            return modbus_read_input_registers(ctx, addr, n, dest);
        });
    }

    /// @brief writes to the coils of the device, with function code 15, or 05 for each
    /// coil when the device writes with single codes.
    /// @param addr the address to start writing to.
    /// @param nb the number of bits to write.
    /// @param src the source buffer, at least nb bytes long.
    x::errors::Error write_bits(const int addr, const size_t nb, const uint8_t *src) {
        if (this->requests.write_codes == WriteCodes::Multiple)
            return this->exec(true, [&](modbus_t *ctx) {
                return modbus_write_bits(ctx, addr, static_cast<int>(nb), src);
            });
        for (std::size_t i = 0; i < nb; i++) {
            const int a = addr + static_cast<int>(i);
            if (auto err = this->exec(true, [&](modbus_t *ctx) {
                    return modbus_write_bit(ctx, a, src[i]);
                }))
                return err;
        }
        return x::errors::NIL;
    }

    /// @brief writes to the holding registers of the device, with function code 16, or
    /// 06 for each register when the device writes with single codes.
    /// @param addr the address to start writing to.
    /// @param nb the number of registers to write.
    /// @param src the source buffer, at least nb registers long.
    x::errors::Error
    write_registers(const int addr, const size_t nb, const uint16_t *src) {
        if (this->requests.write_codes == WriteCodes::Multiple)
            return this->exec(true, [&](modbus_t *ctx) {
                return modbus_write_registers(ctx, addr, static_cast<int>(nb), src);
            });
        for (std::size_t i = 0; i < nb; i++) {
            const int a = addr + static_cast<int>(i);
            if (auto err = this->exec(true, [&](modbus_t *ctx) {
                    return modbus_write_register(ctx, a, src[i]);
                }))
                return err;
        }
        return x::errors::NIL;
    }

    /// @brief checks that the unit answers by reading holding register 0.
    /// @param wait how long to wait for the turn on the shared connection.
    /// @returns NIL if the unit replied, an exception included, and the error of the
    /// request otherwise.
    x::errors::Error probe(const x::telem::TimeSpan wait = PROBE_WAIT) {
        const auto start = Connection::Clock::now();
        std::uint16_t value = 0;
        const auto err = this->conn->exec(
            this->cfg,
            this->requests.delay,
            false,
            [&](modbus_t *ctx) { return modbus_read_registers(ctx, 0, 1, &value); },
            wait
        );
        if (!err || this->last_reply() >= start) return x::errors::NIL;
        return err;
    }

    /// @returns when the unit last answered a request on the shared connection.
    [[nodiscard]] Connection::Clock::time_point last_reply() const {
        return this->conn->last_reply(this->cfg.unit_id);
    }
};

/// @brief shares one connection among the devices on each endpoint. Safe for concurrent
/// use.
class Manager {
    std::mutex mu;
    std::unordered_map<std::string, std::weak_ptr<Connection>> connections;
    /// @brief how long the requests of a demoted unit fail at once.
    x::telem::TimeSpan demotion;

public:
    /// @param demotion how long the requests of a demoted unit fail at once.
    explicit Manager(const x::telem::TimeSpan demotion = DEMOTION):
        demotion(demotion) {}

    /// @brief returns a device on the shared connection to its endpoint. The connection
    /// lives while any device on it does. Acquiring does not connect: the first request
    /// does.
    /// @param cfg the endpoint and unit of the device.
    /// @param requests how the device plans and paces its requests.
    std::pair<std::shared_ptr<Device>, x::errors::Error>
    acquire(const ConnectionConfig &cfg, const RequestConfig &requests = {}) {
        std::lock_guard lock(this->mu);
        std::erase_if(this->connections, [](const auto &entry) {
            return entry.second.expired();
        });
        auto &entry = this->connections[cfg.endpoint()];
        auto conn = entry.lock();
        if (conn == nullptr) {
            const auto port = std::to_string(cfg.port);
            auto *ctx = modbus_new_tcp_pi(cfg.host.c_str(), port.c_str());
            if (ctx == nullptr) return {nullptr, classify(errno)};
            modbus_enable_quirks(ctx, MODBUS_QUIRK_MAX_SLAVE);
            conn = std::make_shared<Connection>(ctx, this->demotion);
            entry = conn;
        }
        return {std::make_shared<Device>(conn, cfg, requests), x::errors::NIL};
    }
};
}
