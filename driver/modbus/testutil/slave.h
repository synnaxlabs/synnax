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
#include <atomic>
#include <chrono>
#include <map>
#include <mutex>
#include <set>
#include <thread>
#include <unordered_map>
#include <vector>

/// platform-specific headers
#ifdef _WIN32
#include <winsock2.h>
#include <ws2tcpip.h>
#pragma comment(lib, "ws2_32.lib")
#else
#include <arpa/inet.h>
#include <netinet/in.h>
#include <sys/select.h>
#include <sys/socket.h>
#include <unistd.h>
#endif

#include "absl/log/log.h"
#include "modbus/modbus.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "driver/modbus/testutil/pseudo_terminal.h"
#include "driver/modbus/util/util.h"

namespace driver::modbus::testutil {
/// @brief Configuration for a mock Modbus slave
struct SlaveConfig {
    // Maps for storing configured values for each register type
    std::unordered_map<int, uint8_t> coils;
    std::unordered_map<int, uint8_t> discrete_inputs;
    std::unordered_map<int, x::telem::SampleValue> holding_registers;
    std::unordered_map<int, x::telem::SampleValue> input_registers;
    std::string host = "127.0.0.1";
    int port = 1502;
    /// @brief function codes the slave answers with an illegal function exception.
    std::set<int> reject_function_codes;
    /// @brief units the slave never answers, as if they were offline.
    std::set<int> silent_units;
    /// @brief units the slave answers with a gateway target exception (0B).
    std::set<int> gateway_failed_units;
    /// @brief whether the slave serves Modbus RTU on a pseudo-terminal instead of TCP.
    /// Not on Windows.
    bool serial = false;
};

/// @brief A simple mock Modbus slave server using libmodbus
class Slave {
    modbus_t *ctx_;
    std::thread server_thread_;
    std::atomic<bool> running_;
    std::string ip_address_;
    int port_;
    int socket_{-1};
    mutable std::mutex mutex_;
    SlaveConfig config_;
    modbus_mapping_t *mb_mapping_; // Add as member
    /// @brief the number of requests received, by function code. Guarded by mutex_.
    std::map<int, std::size_t> requests_;
    /// @brief the number of requests received, by unit. Guarded by mutex_.
    std::map<int, std::size_t> unit_requests_;
    /// @brief the function codes of the requests received, in order. Guarded by mutex_.
    std::vector<int> log_;
    /// @brief the one-time delay before the next reply to each unit. Guarded by mutex_.
    std::map<int, std::chrono::milliseconds> stalls_;
    /// @brief the units whose next reply goes out damaged. Guarded by mutex_.
    std::set<int> damaged_;
    /// @brief the number of client connections accepted. Guarded by mutex_.
    std::size_t connections_ = 0;
    /// @brief set to ask the server loop to close every client connection.
    std::atomic<bool> drop_requested_{false};
    /// @brief the line of a serial slave. Held open so that it outlives each client.
    PseudoTerminal terminal_;

    /// @brief closes a socket. A Winsock socket is not a CRT file descriptor, so
    /// close() aborts on Windows.
    static void close_socket(const int socket) {
#ifdef _WIN32
        closesocket(socket);
#else
        close(socket);
#endif
    }

    // Helper function to find the highest address in a map
    template<typename T>
    int get_max_address(const std::unordered_map<int, T> &map) {
        int max_addr = -1;
        for (const auto &pair: map) {
            if (pair.first > max_addr) { max_addr = pair.first; }
        }
        return max_addr;
    }

    // Create mapping based on configured values
    modbus_mapping_t *create_mapping() {
        const int max_coil_addr = get_max_address(config_.coils);
        const int max_coil = std::max(16, max_coil_addr);
        const int max_discrete_addr = get_max_address(config_.discrete_inputs);
        const int max_discrete = std::max(16, max_discrete_addr);
        int max_holding_register = get_max_address(config_.holding_registers);
        if (config_.holding_registers.empty())
            max_holding_register = 16;
        else
            max_holding_register += (x::telem::DataType::infer(
                                         config_.holding_registers[max_holding_register]
                                     )
                                         .density() +
                                     1) /
                                    2;
        int max_input_register = std::max(16, get_max_address(config_.input_registers));
        if (config_.input_registers.empty())
            max_input_register = 16;
        else
            max_input_register += (x::telem::DataType::infer(
                                       config_.input_registers[max_input_register]
                                   )
                                       .density() +
                                   1) /
                                  2;

        const int nb_bits = max_coil + 1;
        const int nb_input_bits = max_discrete + 1;
        const int nb_registers = max_holding_register + 1;
        const int nb_input_registers = max_input_register + 1;

        LOG(INFO) << "Creating mapping with sizes:" << " coils=" << nb_bits
                  << " discrete_inputs=" << nb_input_bits
                  << " holding_registers=" << nb_registers
                  << " input_registers=" << nb_input_registers;

        modbus_mapping_t *mb_mapping = modbus_mapping_new(
            nb_bits,
            nb_input_bits,
            nb_registers,
            nb_input_registers
        );
        if (mb_mapping == nullptr) {
            LOG(ERROR) << "modbus_mapping_new failed: " << modbus_strerror(errno);
            return nullptr;
        }

        // Log the configuration being applied
        for (const auto &pair: config_.coils) {
            if (pair.first < nb_bits) {
                mb_mapping->tab_bits[pair.first] = pair.second ? 1 : 0;
                LOG(INFO) << "Set coil[" << pair.first
                          << "] = " << (pair.second ? 1 : 0);
            } else
                LOG(WARNING) << "Coil address " << pair.first << " out of range";
        }

        for (const auto &[addr, value]: config_.discrete_inputs)
            if (addr < nb_input_bits) mb_mapping->tab_input_bits[addr] = value ? 1 : 0;

        for (const auto &[addr, value]: config_.holding_registers)
            if (addr < nb_registers) {
                auto dt = x::telem::DataType::infer(value);
                LOG(INFO) << "Holding register[" << addr
                          << "]: inferred type=" << dt.name()
                          << " density=" << dt.density()
                          << " dest_size=" << ((dt.density() + 1) / 2);
                std::vector<uint16_t> dest((dt.density() + 1) / 2);
                if (const auto err = util::format_register(
                        value,
                        dest.data(),
                        dt,
                        false,
                        false
                    ))
                    LOG(FATAL) << err;
                for (size_t i = 0; i < dest.size(); i++)
                    mb_mapping->tab_registers[addr + i] = dest[i];
            }

        for (const auto &[addr, value]: config_.input_registers)
            if (addr < nb_input_registers) {
                auto dt = x::telem::DataType::infer(value);
                LOG(INFO) << "Input register[" << addr
                          << "]: inferred type=" << dt.name()
                          << " density=" << dt.density()
                          << " dest_size=" << ((dt.density() + 1) / 2);
                std::vector<uint16_t> dest((dt.density() + 1) / 2);
                if (const auto err = util::format_register(
                        value,
                        dest.data(),
                        dt,
                        false,
                        false
                    ))
                    LOG(FATAL) << err;
                for (size_t i = 0; i < dest.size(); i++)
                    mb_mapping->tab_input_registers[addr + i] = dest[i];
            }
        return mb_mapping;
    }

    /// @brief counts a request and answers it on the context's socket as the
    /// configuration says.
    void handle(
        const uint8_t *query,
        const int length,
        const uint8_t unit,
        const uint8_t function_code
    ) {
        VLOG(1) << "Request length: " << length << ", function code: 0x" << std::hex
                << static_cast<int>(function_code);
        std::chrono::milliseconds stall{0};
        {
            std::lock_guard lock(mutex_);
            if (const auto it = stalls_.find(unit); it != stalls_.end()) {
                stall = it->second;
                stalls_.erase(it);
            }
        }
        std::this_thread::sleep_for(stall);
        std::lock_guard lock(mutex_);
        requests_[function_code]++;
        unit_requests_[unit]++;
        log_.push_back(function_code);
        if (config_.silent_units.contains(unit)) return;
        const auto reply = [&] {
            if (config_.gateway_failed_units.contains(unit))
                modbus_reply_exception(ctx_, query, MODBUS_EXCEPTION_GATEWAY_TARGET);
            else if (config_.reject_function_codes.contains(function_code))
                modbus_reply_exception(ctx_, query, MODBUS_EXCEPTION_ILLEGAL_FUNCTION);
            else
                modbus_reply(ctx_, query, length, mb_mapping_);
        };
        if (damaged_.erase(unit) == 0)
            reply();
        else
            terminal_.write_damaged([&](const int fd) {
                modbus_set_socket(ctx_, fd);
                reply();
                modbus_set_socket(ctx_, terminal_.fd());
            });
    }

    /// @brief serves RTU frames from the pseudo-terminal. libmodbus drops each RTU
    /// request for a unit other than its own, so the loop splits frames itself to serve
    /// every unit.
    void serial_loop() {
        uint8_t frame[MODBUS_RTU_MAX_ADU_LENGTH];
        while (running_) {
            const auto length = terminal_.read_frame(frame, sizeof(frame));
            // The shortest RTU frame holds a unit, a function code, and a CRC.
            if (length < 4) continue;
            this->handle(frame, static_cast<int>(length), frame[0], frame[1]);
        }
    }

    void server_loop() {
        uint8_t query[MODBUS_TCP_MAX_ADU_LENGTH];
        fd_set ref_set, rd_set;

        FD_ZERO(&ref_set);
        FD_SET(socket_, &ref_set);
        int fd_max = socket_;

        while (running_) {
            if (drop_requested_) {
                for (int fd = 0; fd <= fd_max; fd++)
                    if (fd != socket_ && FD_ISSET(fd, &ref_set)) {
                        close_socket(fd);
                        FD_CLR(fd, &ref_set);
                    }
                fd_max = socket_;
                drop_requested_ = false;
            }
            rd_set = ref_set;

            // Add timeout to allow checking running_ flag
            struct timeval timeout{};
            timeout.tv_sec = 0;
            timeout.tv_usec = 100000; // 100ms timeout

            const int ready = select(fd_max + 1, &rd_set, nullptr, nullptr, &timeout);
            if (ready == -1) {
                if (errno == EINTR) continue;
                LOG(ERROR) << "Select error: " << strerror(errno);
                break;
            }

            // Timeout or no events, continue to check running_ flag
            if (ready == 0) continue;

            for (int master_socket = 0; master_socket <= fd_max; master_socket++) {
                if (!FD_ISSET(master_socket, &rd_set)) { continue; }

                if (master_socket == socket_) {
                    // Handle new connection
                    socklen_t addr_len = 0;
                    struct sockaddr_in client_addr{};

                    addr_len = sizeof(client_addr);
                    memset(&client_addr, 0, sizeof(client_addr));
                    const int new_fd = accept(
                        socket_,
                        reinterpret_cast<struct sockaddr *>(&client_addr),
                        &addr_len
                    );

                    if (new_fd == -1) {
                        LOG(ERROR) << "Accept error: " << strerror(errno);
                        continue;
                    }

                    LOG(INFO) << "New connection from "
                              << inet_ntoa(client_addr.sin_addr) << ":"
                              << ntohs(client_addr.sin_port) << " on socket " << new_fd;

                    {
                        std::lock_guard lock(mutex_);
                        connections_++;
                    }
                    FD_SET(new_fd, &ref_set);
                    fd_max = std::max(new_fd, fd_max);
                } else {
                    // Handle existing connection
                    modbus_set_socket(ctx_, master_socket);
                    const int rc = modbus_receive(ctx_, query);

                    if (rc > 0) {
                        VLOG(1)
                            << "Received Modbus request on socket " << master_socket;
                        // The unit is at offset 6 and the function code at offset 7 of
                        // a TCP ADU.
                        this->handle(query, rc, query[6], query[7]);
                    } else if (rc == -1) {
                        LOG(INFO) << "Connection closed on socket " << master_socket;
                        close_socket(master_socket);
                        FD_CLR(master_socket, &ref_set);
                        if (master_socket == fd_max) fd_max--;
                    }
                }
            }
        }

        LOG(INFO) << "Server loop exiting";
    }

public:
    /// @brief Create a new Modbus TCP or RTU slave with the given configuration
    /// @param config The slave configuration
    explicit Slave(const SlaveConfig &config):
        running_(false), ip_address_(config.host), port_(config.port), config_(config) {
        // A serial slave reads and writes the pseudo-terminal it opens at start, so the
        // device name and line settings of its context go unused.
        ctx_ = config.serial ? modbus_new_rtu("pty", 19200, 'E', 8, 1)
                             : modbus_new_tcp(ip_address_.c_str(), port_);
        if (ctx_ == nullptr) {
            throw std::runtime_error("Failed to create modbus context");
        }

        // Create initial mapping using existing create_mapping function
        mb_mapping_ = create_mapping();
        if (mb_mapping_ == nullptr) {
            modbus_free(ctx_);
            throw std::runtime_error("Failed to create modbus mapping");
        }
    }

    ~Slave() {
        stop();
        if (mb_mapping_ != nullptr) { modbus_mapping_free(mb_mapping_); }
        if (ctx_ != nullptr) { modbus_free(ctx_); }
    }

    /// @brief Start the slave server in a background thread
    x::errors::Error start() {
        if (running_) { return x::errors::NIL; }

        if (config_.serial) {
            if (const auto err = terminal_.open()) return err;
            modbus_set_socket(ctx_, terminal_.fd());
        } else {
            // Concurrent tests connect faster than the loop accepts. macOS drops the
            // connects past the backlog, and libmodbus gives up before the retransmit.
            socket_ = modbus_tcp_listen(ctx_, SOMAXCONN);
            if (socket_ == -1) {
                return x::errors::Error(
                    "Failed to listen on modbus socket: " +
                    std::string(modbus_strerror(errno))
                );
            }
        }

        modbus_set_debug(ctx_, FALSE);
        running_ = true;

        server_thread_ = std::thread([this] {
            if (config_.serial)
                serial_loop();
            else
                server_loop();
        });
        return x::errors::NIL;
    }

    /// @brief Stop the slave server
    void stop() {
        if (!running_) return;

        this->drop_connections();
        running_ = false;
        if (server_thread_.joinable()) { server_thread_.join(); }

        if (socket_ != -1) {
            close_socket(socket_);
            socket_ = -1;
        }
        terminal_.close();
    }

    /// @brief Get the IP address this slave is bound to
    std::string ip_address() const { return ip_address_; }

    /// @brief Get the port this slave is listening on
    int port() const { return port_; }

    /// @returns the path a client opens to reach a serial slave, valid after start.
    std::string serial_port() const { return terminal_.path(); }

    // Add getters that read directly from the mapping
    uint8_t get_coil(int addr) const {
        std::lock_guard lock(mutex_);
        return mb_mapping_->tab_bits[addr];
    }

    uint16_t get_holding_register(int addr) const {
        std::lock_guard lock(mutex_);
        return mb_mapping_->tab_registers[addr];
    }

    /// @brief sets a coil as if the device changed it.
    void set_coil(const int addr, const uint8_t value) {
        std::lock_guard lock(mutex_);
        mb_mapping_->tab_bits[addr] = value;
    }

    /// @brief sets a holding register as if the device changed it.
    void set_holding_register(const int addr, const uint16_t value) {
        std::lock_guard lock(mutex_);
        mb_mapping_->tab_registers[addr] = value;
    }

    /// @returns the number of requests received with the function code.
    std::size_t requests(const int function_code) const {
        std::lock_guard lock(mutex_);
        const auto it = requests_.find(function_code);
        return it == requests_.end() ? 0 : it->second;
    }

    /// @returns the number of requests received for the unit.
    std::size_t unit_requests(const int unit) const {
        std::lock_guard lock(mutex_);
        const auto it = unit_requests_.find(unit);
        return it == unit_requests_.end() ? 0 : it->second;
    }

    /// @returns the function codes of the requests received, in order.
    std::vector<int> log() const {
        std::lock_guard lock(mutex_);
        return log_;
    }

    /// @returns the number of client connections accepted.
    std::size_t connections() const {
        std::lock_guard lock(mutex_);
        return connections_;
    }

    /// @brief delays the next reply to the unit, blocking the server for the delay.
    void stall_next(const int unit, const std::chrono::milliseconds delay) {
        std::lock_guard lock(mutex_);
        stalls_[unit] = delay;
    }

    /// @brief flips the last CRC byte of the next reply to the unit, as line noise
    /// would. Serial slaves only.
    void damage_next(const int unit) {
        std::lock_guard lock(mutex_);
        damaged_.insert(unit);
    }

    /// @brief sets whether the slave answers the unit.
    void set_silent(const int unit, const bool silent) {
        std::lock_guard lock(mutex_);
        if (silent)
            config_.silent_units.insert(unit);
        else
            config_.silent_units.erase(unit);
    }

    /// @brief closes every client connection, as a server restart or a network fault
    /// would, and blocks until the server loop has closed them. Keeps listening. A
    /// serial slave has no connections to close.
    void drop_connections() {
        if (config_.serial) return;
        drop_requested_ = true;
        while (drop_requested_ && running_)
            std::this_thread::sleep_for(std::chrono::milliseconds(5));
    }
};
}
