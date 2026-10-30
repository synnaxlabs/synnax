// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <array>
#include <cctype>
#include <cerrno>
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
#include <vector>

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
        case EBUSY:
            return {UNREACHABLE, "serial port is in use by another connection"};
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

/// @brief the links a Modbus device can connect over.
enum class Transport {
    /// @brief Modbus TCP to a server or a gateway.
    TCP,
    /// @brief Modbus RTU on a serial port.
    RTU,
};

/// @brief the baud rates libmodbus sets on Linux, macOS, and Windows. On POSIX,
/// libmodbus opens any other rate at 9600 without an error.
const std::array<int, 9> BAUD_RATES = {
    1200,
    2400,
    4800,
    9600,
    19200,
    38400,
    57600,
    115200,
    230400,
};

/// @brief the line settings of a serial port.
struct Serial {
    /// @brief the data bits of each character. Modbus RTU requires 8.
    static constexpr int DATA_BITS = 8;

    /// @brief the serial port, such as /dev/ttyUSB0 or COM3.
    std::string port;
    /// @brief the bits per second on the line.
    std::uint32_t baud_rate = 19200;
    /// @brief 'N' for no parity, 'E' for even, or 'O' for odd.
    char parity = 'E';
    /// @brief the stop bits of each character, 1 or 2.
    std::uint8_t stop_bits = 1;

    /// @returns the parity as JSON stores it: none, even, or odd.
    [[nodiscard]] std::string parity_name() const {
        if (this->parity == 'N') return "none";
        if (this->parity == 'O') return "odd";
        return "even";
    }

    /// @returns the line settings in their usual short form, such as "19200 8E1".
    [[nodiscard]] std::string line() const {
        return std::to_string(this->baud_rate) + " " + std::to_string(DATA_BITS) +
               this->parity + std::to_string(this->stop_bits);
    }

    /// @returns the silence of 3.5 characters that must separate two frames: 1.75 ms
    /// above 19200 baud, as the Modbus serial line specification fixes it.
    [[nodiscard]] x::telem::TimeSpan frame_gap() const {
        if (this->baud_rate > 19200) return x::telem::MICROSECOND * 1750;
        const auto bits = 1 + DATA_BITS + (this->parity == 'N' ? 0 : 1) +
                          this->stop_bits;
        return x::telem::TimeSpan(
            static_cast<std::int64_t>(3.5 * bits * 1e9 / this->baud_rate)
        );
    }
};

/// @returns the path libmodbus opens for a serial port. Windows opens COM10 and up only
/// through the \\.\ device namespace, so every COM port gets that prefix.
inline std::string serial_path(const std::string &port) {
    if (port.size() < 4 || port.size() > 6) return port;
    const auto upper = [&](const std::size_t i) {
        return static_cast<char>(std::toupper(static_cast<unsigned char>(port[i])));
    };
    if (upper(0) != 'C' || upper(1) != 'O' || upper(2) != 'M') return port;
    for (std::size_t i = 3; i < port.size(); i++)
        if (std::isdigit(static_cast<unsigned char>(port[i])) == 0) return port;
    return R"(\\.\)" + port;
}

/// @returns the name that every spelling of a serial port shares: on POSIX, the path
/// with its links resolved, and on Windows, the device path in upper case.
std::string canonical_port(const std::string &port);

/// @returns the name that every spelling of a serial port path shares, without
/// following links: on Windows, the device path in upper case, and on POSIX, the path.
std::string port_name(const std::string &port);

/// @brief clears the error that last_error reads. Call it before a libmodbus request.
void clear_error();

/// @returns the error number of a failed libmodbus request. A failed Windows socket or
/// port call leaves errno as it was, so there it reads as a lost connection.
int last_error();

/// @brief an exclusive hold on a serial port. Take it before the port's line settings
/// change, so that a refused connection cannot disturb the one using the port. Windows
/// already opens a COM port for one handle only, so there it holds nothing.
class PortLock {
    int fd = -1;

public:
    PortLock() = default;
    ~PortLock() { this->unlock(); }
    PortLock(const PortLock &) = delete;
    PortLock &operator=(const PortLock &) = delete;

    /// @brief holds the port until unlock.
    /// @returns 0, EBUSY if another connection holds the port, or the error of opening
    /// the port.
    int lock(const std::string &port);

    /// @brief lets other connections hold the port. Does nothing if not held.
    void unlock();
};

/// @brief the link to a Modbus server and the unit a device addresses on it.
struct ConnectionConfig {
    /// @brief the link the device connects over.
    Transport transport = Transport::TCP;
    /// @brief the hostname, IPv4 address, or IPv6 address of a TCP server.
    std::string host;
    /// @brief the TCP port of a TCP server.
    std::uint16_t port = 0;
    /// @brief the serial port and line settings of an RTU device.
    Serial serial;
    /// @brief whether to swap the byte order within each 16-bit word.
    bool swap_bytes = false;
    /// @brief whether to swap the word order of 32-bit and larger values.
    bool swap_words = false;
    /// @brief the unit the device addresses. 255 addresses a TCP server itself.
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

    /// @brief an RTU device on a serial port, addressing unit 1.
    explicit ConnectionConfig(Serial serial):
        transport(Transport::RTU), serial(std::move(serial)), unit_id(1) {}

    /// @brief parses the connection from its JSON representation. Absent optional
    /// fields take their defaults, and an absent transport is TCP.
    explicit ConnectionConfig(x::json::Parser parser):
        swap_bytes(parser.field<bool>("swap_bytes")),
        swap_words(parser.field<bool>("swap_words")) {
        if (const auto transport = parser.field<std::string>("transport", "tcp");
            transport == "tcp") {
            this->host = parser.field<std::string>("host");
            this->port = parser.field<std::uint16_t>("port");
        } else if (transport == "rtu") {
            this->transport = Transport::RTU;
            this->parse_serial(parser);
        } else
            parser.field_err("transport", "must be tcp or rtu");
        const bool rtu = this->transport == Transport::RTU;
        const auto unit_id = parser.field<int>("unit_id", rtu ? 1 : MODBUS_TCP_SLAVE);
        if (rtu && (unit_id < 1 || unit_id > 247))
            parser.field_err("unit_id", "must be between 1 and 247");
        else if (unit_id < 0 || unit_id > 255)
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
        x::json::json j = {
            {"swap_bytes", swap_bytes},
            {"swap_words", swap_words},
            {"unit_id", unit_id},
            {"timeout_ms", static_cast<int>(timeout.milliseconds())},
            {"retries", retries},
        };
        if (this->transport == Transport::TCP) {
            j["transport"] = "tcp";
            j["host"] = host;
            j["port"] = port;
            return j;
        }
        j["transport"] = "rtu";
        j["serial_port"] = serial.port;
        j["baud_rate"] = serial.baud_rate;
        j["parity"] = serial.parity_name();
        j["stop_bits"] = serial.stop_bits;
        return j;
    }

    /// @returns the host and port for TCP, or the file that the serial port resolves to
    /// now for RTU.
    [[nodiscard]] std::string endpoint() const {
        if (this->transport == Transport::RTU) return canonical_port(this->serial.port);
        return host + ":" + std::to_string(port);
    }

private:
    void parse_serial(x::json::Parser &parser) {
        this->serial.port = parser.field<std::string>("serial_port");
        if (parser.has("serial_port") && this->serial.port.empty())
            parser.field_err("serial_port", "must not be empty");
        const auto baud_rate = parser.field<int>("baud_rate", 19200);
        if (std::ranges::find(BAUD_RATES, baud_rate) == BAUD_RATES.end()) {
            std::string rates;
            for (const int rate: BAUD_RATES)
                rates += (rates.empty() ? "" : ", ") + std::to_string(rate);
            parser.field_err("baud_rate", "must be one of " + rates);
        } else {
            this->serial.baud_rate = static_cast<std::uint32_t>(baud_rate);
        }
        if (const auto parity = parser.field<std::string>("parity", "even");
            parity == "none")
            this->serial.parity = 'N';
        else if (parity == "odd")
            this->serial.parity = 'O';
        else if (parity != "even")
            parser.field_err("parity", "must be none, even, or odd");
        const auto stop_bits = parser.field<int>("stop_bits", 1);
        if (stop_bits != 1 && stop_bits != 2)
            parser.field_err("stop_bits", "must be 1 or 2");
        else
            this->serial.stop_bits = static_cast<std::uint8_t>(stop_bits);
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
    /// @brief whether the server takes function code 22, so a register bit write needs
    /// one request instead of a read and a write.
    bool mask_write_supported = false;
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
        this->mask_write_supported = parser.field<bool>("mask_write_supported", false);
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

/// @brief the request counters of one unit on a shared connection.
struct Stats {
    /// @brief the requests tried, retries included.
    std::uint64_t requests = 0;
    /// @brief the requests tried again after a temporary fault.
    std::uint64_t retries = 0;
    /// @brief the requests the unit did not answer in time.
    std::uint64_t timeouts = 0;
    /// @brief the requests answered with an exception, by the unit or by a gateway in
    /// front of it.
    std::uint64_t exceptions = 0;
    /// @brief the time from sending the last answered request to its reply.
    x::telem::TimeSpan last_response;
    /// @brief the longest time from sending a request to its reply.
    x::telem::TimeSpan max_response;

    /// @returns the JSON representation of the counters, with times in nanoseconds.
    [[nodiscard]] x::json::json to_json() const {
        return {
            {"requests", requests},
            {"retries", retries},
            {"timeouts", timeouts},
            {"exceptions", exceptions},
            {"last_response", last_response.nanoseconds()},
            {"max_response", max_response.nanoseconds()},
        };
    }
};

/// @brief a connection to one Modbus TCP endpoint or serial port, shared by every
/// device on it. Requests take turns. A waiting write goes before waiting reads, but
/// not twice in a row, so reads keep their turns under steady writes. Safe for
/// concurrent use.
class Connection {
public:
    /// @brief the clock of request, reply, and demotion times.
    using Clock = std::chrono::steady_clock;

    /// @brief the link of the connection.
    const Transport transport;
    /// @brief the line settings of a serial connection. Unused for TCP.
    const Serial serial;
    /// @brief the host and port of a TCP connection. Empty for RTU.
    const std::string address;

private:
    /// @brief the state of one unit behind the endpoint.
    struct Unit {
        /// @brief the timed-out attempts since the unit last answered.
        std::size_t timeouts = 0;
        /// @brief requests to the unit fail at once until this time.
        Clock::time_point demoted_until;
        /// @brief when the unit last answered, an exception included.
        Clock::time_point last_reply;
        /// @brief the request counters of the unit.
        Stats stats;
    };

    modbus_t *ctx;
    /// @brief how long the requests of a demoted unit fail at once.
    x::telem::TimeSpan demotion;
    /// @brief the shortest silence between two requests.
    const x::telem::TimeSpan frame_gap;
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
    /// @brief holds the serial port while connected. Touched only while holding the
    /// turn.
    PortLock port_lock;
    /// @brief whether the socket or port is open. Touched only while holding the turn.
    bool connected = false;
    /// @brief the file the serial port resolved to when it opened. Touched only while
    /// holding the turn.
    std::string opened;
    /// @brief whether a late reply or the rest of a garbled frame can still arrive.
    /// Touched only while holding the turn.
    bool stale = false;
    /// @brief when the last request ended. Touched only while holding the turn.
    Clock::time_point last_request;
    /// @brief when the last attempt sent its request. Touched only while holding the
    /// turn.
    Clock::time_point sent;

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

    /// @brief closes the socket or port. Call only while holding the turn.
    void disconnect() {
        modbus_close(this->ctx);
        this->port_lock.unlock();
        this->connected = false;
    }

    /// @brief the result of one attempt of a request.
    struct Attempt {
        /// @brief 0 on success, or the error number.
        int errnum = 0;
        /// @brief whether the request went out. A failed connect sends nothing, so its
        /// timeout does not count against the unit.
        bool sent = false;
    };

    /// @brief sends one attempt of a request.
    template<typename Request>
    Attempt attempt(const ConnectionConfig &cfg, const Request &request) {
        const auto timeout = std::chrono::duration_cast<std::chrono::microseconds>(
            cfg.timeout.chrono()
        );
        modbus_set_response_timeout(
            this->ctx,
            static_cast<std::uint32_t>(timeout.count() / 1000000),
            static_cast<std::uint32_t>(timeout.count() % 1000000)
        );
        if (this->connected && this->transport == Transport::RTU &&
            canonical_port(this->serial.port) != this->opened)
            this->disconnect();
        if (!this->connected) {
            if (this->transport == Transport::RTU) {
                if (const int errnum = this->port_lock.lock(this->serial.port))
                    return {errnum};
                this->opened = canonical_port(this->serial.port);
            }
            if (modbus_connect(this->ctx) == -1) {
                const int errnum = errno;
                this->port_lock.unlock();
                return {errnum};
            }
            this->connected = true;
            this->stale = false;
        }
        if (this->stale) {
            modbus_flush(this->ctx);
            this->stale = false;
        }
        if (modbus_set_slave(this->ctx, cfg.unit_id) == -1) return {errno};
        this->sent = Clock::now();
        clear_error();
        if (request(this->ctx) != -1) return {0, true};
        const int errnum = last_error();
        if (errnum == ETIMEDOUT || errnum == EMBBADCRC)
            this->stale = true;
        else if (classify(errnum).matches(UNREACHABLE))
            this->disconnect();
        return {errnum, true};
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

    /// @brief adds an attempt to the counters of the unit. Any reply times the
    /// response, a gateway exception included. Only a reply from the unit itself
    /// records that the unit answered.
    void record(const std::uint8_t unit_id, const bool retry, const Attempt &result) {
        const int errnum = result.errnum;
        std::lock_guard lock(this->mu);
        auto &unit = this->units[unit_id];
        auto &stats = unit.stats;
        stats.requests++;
        if (retry) stats.retries++;
        if (result.sent && errnum == ETIMEDOUT) stats.timeouts++;
        // libmodbus numbers the exception codes 01 to 0B in order.
        const bool exception = errnum >= EMBXILFUN && errnum <= EMBXGTAR;
        if (exception) stats.exceptions++;
        if (errnum != 0 && !exception) return;
        stats.last_response = x::telem::TimeSpan(this->last_request - this->sent);
        stats.max_response = std::max(stats.max_response, stats.last_response);
        if (errnum != 0 && !answered(errnum)) return;
        unit.timeouts = 0;
        unit.last_reply = this->last_request;
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

    /// @brief runs a request for a unit while holding the turn.
    template<typename Request>
    x::errors::Error send(
        const ConnectionConfig &cfg,
        const x::telem::TimeSpan delay,
        const Request &request
    ) {
        if (auto err = this->demoted(cfg.unit_id)) return err;
        const auto gap = std::max(delay, this->frame_gap);
        for (std::size_t i = 0;; i++) {
            if (gap > x::telem::TimeSpan(0))
                std::this_thread::sleep_until(this->last_request + gap.chrono());
            const auto result = this->attempt(cfg, request);
            this->last_request = Clock::now();
            this->record(cfg.unit_id, i > 0, result);
            if (result.errnum == 0) return x::errors::NIL;
            if (result.sent && result.errnum == ETIMEDOUT &&
                this->timed_out(cfg.unit_id))
                return this->demoted(cfg.unit_id);
            auto err = classify(result.errnum);
            if (!err.matches(TEMPORARY_ERROR) || i >= cfg.retries) return err;
        }
    }

public:
    /// @brief takes ownership of a context created by modbus_new_tcp_pi.
    /// @param address the host and port of the context.
    /// @param demotion how long the requests of a demoted unit fail at once.
    Connection(modbus_t *ctx, std::string address, const x::telem::TimeSpan demotion):
        transport(Transport::TCP),
        address(std::move(address)),
        ctx(ctx),
        demotion(demotion),
        frame_gap(0) {}

    /// @brief takes ownership of a context created by modbus_new_rtu with the line
    /// settings.
    /// @param demotion how long the requests of a demoted unit fail at once.
    Connection(modbus_t *ctx, Serial serial, const x::telem::TimeSpan demotion):
        transport(Transport::RTU),
        serial(std::move(serial)),
        ctx(ctx),
        demotion(demotion),
        frame_gap(this->serial.frame_gap()) {}

    ~Connection() {
        modbus_close(this->ctx);
        modbus_free(this->ctx);
    }

    Connection(const Connection &) = delete;
    Connection &operator=(const Connection &) = delete;

    /// @brief runs a request for a unit, connecting first if needed. A temporary fault
    /// retries at once, up to cfg.retries times. After a timeout or a bad CRC, the next
    /// request first discards any late bytes. Any other lost connection closes the
    /// socket or port. A unit with DEMOTE_AFTER timed-out attempts in a row is demoted.
    /// A serial port whose link moved to another file reopens first.
    /// @param cfg the unit, timeout, and retries of the request.
    /// @param delay the wait after the previous request on the connection. A serial
    /// connection waits at least its frame gap.
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
        return this->sequence(
            cfg,
            delay,
            write,
            [&](const auto &send) { return send(request); },
            wait
        );
    }

    /// @brief runs several requests for a unit in one turn, so that no other request on
    /// the connection comes between them. Each request runs as exec runs it.
    /// @param run calls its argument with each request, which returns the request's
    /// error, and returns NIL or the error to stop at.
    /// @param wait how long to wait for the turn. Waits without limit if empty.
    /// @returns the error of run, or UNREACHABLE if other requests kept the connection
    /// busy for wait.
    template<typename Run>
    x::errors::Error sequence(
        const ConnectionConfig &cfg,
        const x::telem::TimeSpan delay,
        const bool write,
        const Run &run,
        const std::optional<x::telem::TimeSpan> &wait = std::nullopt
    ) {
        Turn turn(*this, write, wait);
        if (turn.err) return turn.err;
        return run([&](const auto &request) {
            return this->send(cfg, delay, request);
        });
    }

    /// @returns when the unit last answered a request, an exception included.
    Clock::time_point last_reply(const std::uint8_t unit_id) {
        std::lock_guard lock(this->mu);
        return this->units[unit_id].last_reply;
    }

    /// @returns the request counters of the unit since the connection was created.
    Stats stats(const std::uint8_t unit_id) {
        std::lock_guard lock(this->mu);
        return this->units[unit_id].stats;
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

    template<typename Run>
    x::errors::Error sequence(const bool write, const Run &run) {
        return this->conn->sequence(this->cfg, this->requests.delay, write, run);
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

    /// @brief sets the bits of a holding register that and_mask clears to those of
    /// or_mask, and keeps the rest. With mask writes this is one function code 22
    /// request. Otherwise the device reads the register and writes it back in one turn
    /// on the connection, and a change the server makes between the two is lost.
    /// @param addr the address of the register.
    /// @param and_mask the bits to keep.
    /// @param or_mask the new values of the bits that and_mask clears.
    x::errors::Error mask_write_register(
        const int addr,
        const std::uint16_t and_mask,
        const std::uint16_t or_mask
    ) {
        if (this->requests.mask_write_supported)
            return this->exec(true, [&](modbus_t *ctx) {
                return modbus_mask_write_register(ctx, addr, and_mask, or_mask);
            });
        return this->sequence(true, [&](const auto &send) {
            std::uint16_t value = 0;
            if (auto err = send([&](modbus_t *ctx) {
                    return modbus_read_registers(ctx, addr, 1, &value);
                }))
                return err;
            value = (value & and_mask) | (or_mask & ~and_mask);
            return send([&](modbus_t *ctx) {
                if (this->requests.write_codes == WriteCodes::Single)
                    return modbus_write_register(ctx, addr, value);
                return modbus_write_registers(ctx, addr, 1, &value);
            });
        });
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

    /// @returns the request counters of the unit on the shared connection.
    [[nodiscard]] Stats stats() const { return this->conn->stats(this->cfg.unit_id); }

    /// @returns the endpoint and unit of the device.
    [[nodiscard]] const ConnectionConfig &config() const { return this->cfg; }
};

/// @brief shares one connection among the devices on each endpoint: a TCP host and
/// port, or a serial port path. Devices that name one serial port by different paths
/// get separate connections, and the port lock lets one of them use the port at a time.
/// Safe for concurrent use.
class Manager {
    std::mutex mu;
    std::vector<std::weak_ptr<Connection>> connections;
    /// @brief how long the requests of a demoted unit fail at once.
    x::telem::TimeSpan demotion;

    /// @returns a new connection, or nullptr with errno set.
    [[nodiscard]] std::shared_ptr<Connection>
    connect(const ConnectionConfig &cfg) const {
        if (cfg.transport == Transport::RTU) {
            const auto &serial = cfg.serial;
            auto *ctx = modbus_new_rtu(
                serial_path(serial.port).c_str(),
                static_cast<int>(serial.baud_rate),
                serial.parity,
                Serial::DATA_BITS,
                serial.stop_bits
            );
            if (ctx == nullptr) return nullptr;
            return std::make_shared<Connection>(ctx, serial, this->demotion);
        }
        const auto port = std::to_string(cfg.port);
        auto *ctx = modbus_new_tcp_pi(cfg.host.c_str(), port.c_str());
        if (ctx == nullptr) return nullptr;
        modbus_enable_quirks(ctx, MODBUS_QUIRK_MAX_SLAVE);
        return std::make_shared<Connection>(ctx, cfg.endpoint(), this->demotion);
    }

    /// @returns the open connection to the TCP endpoint or serial port path of cfg, or
    /// nullptr if none.
    [[nodiscard]] std::shared_ptr<Connection> find(const ConnectionConfig &cfg) const {
        for (const auto &weak: this->connections) {
            auto conn = weak.lock();
            if (conn == nullptr || conn->transport != cfg.transport) continue;
            if (cfg.transport == Transport::RTU
                    ? port_name(conn->serial.port) == port_name(cfg.serial.port)
                    : conn->address == cfg.endpoint())
                return conn;
        }
        return nullptr;
    }

public:
    /// @param demotion how long the requests of a demoted unit fail at once.
    explicit Manager(const x::telem::TimeSpan demotion = DEMOTION):
        demotion(demotion) {}

    /// @brief returns a device on the shared connection to its endpoint. The connection
    /// lives while any device on it does. Acquiring does not connect: the first request
    /// does.
    /// @param cfg the endpoint and unit of the device.
    /// @param requests how the device plans and paces its requests.
    /// @returns CRITICAL_ERROR if the serial port is already open with other line
    /// settings.
    std::pair<std::shared_ptr<Device>, x::errors::Error>
    acquire(const ConnectionConfig &cfg, const RequestConfig &requests = {}) {
        std::lock_guard lock(this->mu);
        std::erase_if(this->connections, [](const auto &weak) {
            return weak.expired();
        });
        auto conn = this->find(cfg);
        if (conn == nullptr) {
            conn = connect(cfg);
            if (conn == nullptr) return {nullptr, classify(errno)};
            this->connections.push_back(conn);
        } else if (
            cfg.transport == Transport::RTU && conn->serial.line() != cfg.serial.line()
        )
            return {
                nullptr,
                {CRITICAL_ERROR,
                 "serial port " + cfg.serial.port + " is open at " +
                     conn->serial.line() + ", but this device uses " +
                     cfg.serial.line()}
            };
        return {std::make_shared<Device>(conn, cfg, requests), x::errors::NIL};
    }
};
}
