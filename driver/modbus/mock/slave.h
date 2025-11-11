// Copyright 2025 Synnax Labs, Inc.
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
#include <memory>
#include <mutex>
#include <thread>
#include <unordered_map>

/// platform-specific headers
#ifdef _WIN32
#include <io.h>
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

#include "modbus/modbus.h"

#include "x/cpp/telem/telem.h"
#include "x/cpp/xerrors/errors.h"

#include "driver/modbus/util/util.h"

/// glog
#include "glog/logging.h"

namespace modbus::mock {
/// @brief Configuration for a mock Modbus slave
struct SlaveConfig {
    // Maps for storing configured values for each register type
    std::unordered_map<int, uint8_t> coils;
    std::unordered_map<int, uint8_t> discrete_inputs;
    std::unordered_map<int, telem::SampleValue> holding_registers;
    std::unordered_map<int, telem::SampleValue> input_registers;
    std::string host = "127.0.0.1";
    int port = 1502;
};

/// @brief A simple mock Modbus slave server using libmodbus
class Slave {
    modbus_t *ctx_;
    std::thread server_thread_;
    std::atomic<bool> running_;
    std::string ip_address_;
    int port_;
    int socket_;
    mutable std::mutex mutex_;
    SlaveConfig config_;
    modbus_mapping_t *mb_mapping_; // Add as member

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
            max_holding_register += (telem::DataType::infer(
                                         config_.holding_registers[max_holding_register]
                                     )
                                         .density() +
                                     1) /
                                    2;
        int max_input_register = std::max(16, get_max_address(config_.input_registers));
        if (config_.input_registers.empty())
            max_input_register = 16;
        else
            max_input_register += (telem::DataType::infer(
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
                auto dt = telem::DataType::infer(value);
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
                auto dt = telem::DataType::infer(value);
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

    void server_loop() {
        uint8_t query[MODBUS_TCP_MAX_ADU_LENGTH];
        fd_set ref_set, rd_set;

        FD_ZERO(&ref_set);
        FD_SET(socket_, &ref_set);
        int fd_max = socket_;

        while (running_) {
            rd_set = ref_set;

            // Add timeout to allow checking running_ flag
            struct timeval timeout;
            timeout.tv_sec = 0;
            timeout.tv_usec = 100000; // 100ms timeout

            const int ready = select(fd_max + 1, &rd_set, NULL, NULL, &timeout);
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
                    socklen_t addr_len;
                    struct sockaddr_in client_addr;

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

                    FD_SET(new_fd, &ref_set);
                    if (new_fd > fd_max) fd_max = new_fd;
                } else {
                    // Handle existing connection
                    modbus_set_socket(ctx_, master_socket);
                    const int rc = modbus_receive(ctx_, query);

                    if (rc > 0) {
                        // Log the function code and data from the query
                        const uint8_t function_code = query[7];
                        // Function code is at offset 7 in TCP ADU
                        VLOG(1) << "Received Modbus request on socket " << master_socket
                                << ", length: " << rc << ", function code: 0x"
                                << std::hex << static_cast<int>(function_code);

                        std::lock_guard lock(mutex_);
                        modbus_reply(ctx_, query, rc, mb_mapping_);
                        VLOG(1) << "Replied to request on socket " << master_socket;

                        VLOG(1) << "Response data:";
                        for (int i = 0; i < rc; i++) {
                            VLOG(1) << "  byte[" << i << "] = 0x" << std::hex
                                    << static_cast<int>(query[i]);
                        }
                    } else if (rc == -1) {
                        LOG(INFO) << "Connection closed on socket " << master_socket;
                        close(master_socket);
                        FD_CLR(master_socket, &ref_set);
                        if (master_socket == fd_max) fd_max--;
                    }
                }
            }
        }

        LOG(INFO) << "Server loop exiting";
    }

public:
    /// @brief Create a new Modbus TCP slave with the given configuration
    /// @param config The slave configuration
    explicit Slave(const SlaveConfig &config):
        running_(false),
        ip_address_(config.host),
        port_(config.port),
        socket_(-1),
        config_(config) {
        ctx_ = modbus_new_tcp(ip_address_.c_str(), port_);
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
    xerrors::Error start() {
        if (running_) { return xerrors::NIL; }

        socket_ = modbus_tcp_listen(ctx_, 1);
        if (socket_ == -1) {
            return xerrors::Error(
                "Failed to listen on modbus socket: " +
                std::string(modbus_strerror(errno))
            );
        }

        modbus_set_debug(ctx_, FALSE);
        running_ = true;

        server_thread_ = std::thread([this] { server_loop(); });
        return xerrors::NIL;
    }

    /// @brief Stop the slave server
    void stop() {
        if (!running_) return;

        running_ = false;
        if (server_thread_.joinable()) { server_thread_.join(); }

        if (socket_ != -1) {
            close(socket_);
            socket_ = -1;
        }
    }

    /// @brief Get the IP address this slave is bound to
    std::string ip_address() const { return ip_address_; }

    /// @brief Get the port this slave is listening on
    int port() const { return port_; }

    // Add getters that read directly from the mapping
    uint8_t get_coil(int addr) const {
        std::lock_guard lock(mutex_);
        return mb_mapping_->tab_bits[addr];
    }

    uint16_t get_holding_register(int addr) const {
        std::lock_guard lock(mutex_);
        return mb_mapping_->tab_registers[addr];
    }
};
} // namespace modbus::mock
