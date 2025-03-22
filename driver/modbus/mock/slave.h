// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <memory>
#include <thread>
#include <atomic>
#include <mutex>
#include <unordered_map>
#include <sys/select.h>
#include <unistd.h>
#include <algorithm>

/// network headers
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>

/// external
#include "modbus/modbus.h"

/// internal
#include "x/cpp/xerrors/errors.h"

/// glog
#include "glog/logging.h"

namespace modbus::mock {

/// @brief Configuration for a mock Modbus slave
struct SlaveConfig {
    // Maps for storing configured values for each register type
    std::unordered_map<int, uint8_t> coils;
    std::unordered_map<int, uint8_t> discrete_inputs;
    std::unordered_map<int, uint16_t> holding_registers;
    std::unordered_map<int, uint16_t> input_registers;
    std::string host = "127.0.0.1";
    int port = 1502;
};

/// @brief A simple mock Modbus slave server using libmodbus
class Slave {
    modbus_t* ctx_;
    std::thread server_thread_;
    std::atomic<bool> running_;
    std::string ip_address_;
    int port_;
    int socket_;
    mutable std::mutex mutex_;
    SlaveConfig config_;
    modbus_mapping_t* mb_mapping_;  // Add as member

    // Helper function to find the highest address in a map
    template<typename T>
    int get_max_address(const std::unordered_map<int, T>& map) {
        int max_addr = -1;
        for (const auto& pair : map) {
            if (pair.first > max_addr) {
                max_addr = pair.first;
            }
        }
        return max_addr;
    }

    // Create mapping based on configured values
    modbus_mapping_t* create_mapping() {
        // Find the highest address for each type to determine mapping size
        const int max_coil = std::max(16, get_max_address(config_.coils));  // Minimum size of 16
        const int max_discrete = std::max(16, get_max_address(config_.discrete_inputs));
        const int max_holding = std::max(16, get_max_address(config_.holding_registers));
        const int max_input = std::max(16, get_max_address(config_.input_registers));

        // Add 1 to get the size (addresses are 0-based)
        const int nb_bits = max_coil + 1;
        const int nb_input_bits = max_discrete + 1;
        const int nb_registers = max_holding + 1;
        const int nb_input_registers = max_input + 1;

        LOG(INFO) << "Creating mapping with sizes:"
                  << " coils=" << nb_bits
                  << " discrete_inputs=" << nb_input_bits
                  << " holding_registers=" << nb_registers
                  << " input_registers=" << nb_input_registers;

        modbus_mapping_t* mb_mapping = modbus_mapping_new(nb_bits, nb_input_bits, 
                                                         nb_registers, nb_input_registers);
        if (mb_mapping == nullptr) {
            LOG(ERROR) << "modbus_mapping_new failed: " << modbus_strerror(errno);
            return nullptr;
        }

        // Log the configuration being applied
        for (const auto& pair : config_.coils) {
            if (pair.first < nb_bits) {
                mb_mapping->tab_bits[pair.first] = pair.second ? 1 : 0;
                LOG(INFO) << "Set coil[" << pair.first << "] = " << (pair.second ? 1 : 0);
            } else {
                LOG(WARNING) << "Coil address " << pair.first << " out of range";
            }
        }

        for (const auto& pair : config_.discrete_inputs)
            if (pair.first < nb_input_bits)
                mb_mapping->tab_input_bits[pair.first] = pair.second ? 1 : 0;

        for (const auto& pair : config_.holding_registers)
            if (pair.first < nb_registers)
                mb_mapping->tab_registers[pair.first] = pair.second;

        for (const auto& pair : config_.input_registers)
            if (pair.first < nb_input_registers)
                mb_mapping->tab_input_registers[pair.first] = pair.second;

        return mb_mapping;
    }

    void server_loop() {
        uint8_t query[MODBUS_TCP_MAX_ADU_LENGTH];
        int master_socket;
        fd_set refset;
        fd_set rdset;
        int fdmax;

        FD_ZERO(&refset);
        FD_SET(socket_, &refset);
        fdmax = socket_;

        while (running_) {
            rdset = refset;
            
            // Add timeout to allow checking running_ flag
            struct timeval timeout;
            timeout.tv_sec = 0;
            timeout.tv_usec = 100000;  // 100ms timeout
            
            int ready = select(fdmax + 1, &rdset, NULL, NULL, &timeout);
            if (ready == -1) {
                if (errno == EINTR) continue;
                LOG(ERROR) << "Select error: " << strerror(errno);
                break;
            }
            
            // Timeout or no events, continue to check running_ flag
            if (ready == 0) continue;

            for (master_socket = 0; master_socket <= fdmax; master_socket++) {
                if (!FD_ISSET(master_socket, &rdset)) {
                    continue;
                }

                if (master_socket == socket_) {
                    // Handle new connection
                    socklen_t addrlen;
                    struct sockaddr_in clientaddr;
                    int newfd;

                    addrlen = sizeof(clientaddr);
                    memset(&clientaddr, 0, sizeof(clientaddr));
                    newfd = accept(socket_, (struct sockaddr*)&clientaddr, &addrlen);
                    
                    if (newfd == -1) {
                        LOG(ERROR) << "Accept error: " << strerror(errno);
                        continue;
                    }

                    LOG(INFO) << "New connection from " << inet_ntoa(clientaddr.sin_addr)
                            << ":" << ntohs(clientaddr.sin_port) 
                            << " on socket " << newfd;

                    FD_SET(newfd, &refset);
                    if (newfd > fdmax) {
                        fdmax = newfd;
                    }
                } else {
                    // Handle existing connection
                    modbus_set_socket(ctx_, master_socket);
                    int rc = modbus_receive(ctx_, query);
                    
                    if (rc > 0) {
                        // Log the function code and data from the query
                        uint8_t function_code = query[7];  // Function code is at offset 7 in TCP ADU
                        LOG(INFO) << "Received Modbus request on socket " << master_socket
                                  << ", length: " << rc
                                  << ", function code: 0x" << std::hex << (int)function_code;

                        std::lock_guard lock(mutex_);
                        modbus_reply(ctx_, query, rc, mb_mapping_);
                        LOG(INFO) << "Replied to request on socket " << master_socket;
                        
                        // Log the response data
                        LOG(INFO) << "Response data:";
                        for (int i = 0; i < rc; i++) {
                            LOG(INFO) << "  byte[" << i << "] = 0x" << std::hex << (int)query[i];
                        }
                    } else if (rc == -1) {
                        LOG(INFO) << "Connection closed on socket " << master_socket;
                        close(master_socket);
                        FD_CLR(master_socket, &refset);
                        if (master_socket == fdmax) {
                            fdmax--;
                        }
                    }
                }
            }
        }
        
        LOG(INFO) << "Server loop exiting";
    }

public:
    /// @brief Create a new Modbus TCP slave with the given configuration
    /// @param config The slave configuration
    explicit Slave(const SlaveConfig& config) 
        : running_(false),
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
        if (mb_mapping_ != nullptr) {
            modbus_mapping_free(mb_mapping_);
        }
        if (ctx_ != nullptr) {
            modbus_free(ctx_);
        }
    }

    /// @brief Start the slave server in a background thread
    xerrors::Error start() {
        if (running_) {
            return xerrors::NIL;
        }

        socket_ = modbus_tcp_listen(ctx_, 1);
        if (socket_ == -1) {
            return xerrors::Error("Failed to listen on modbus socket: " + 
                                std::string(modbus_strerror(errno)));
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
        if (server_thread_.joinable()) {
            server_thread_.join();
        }
        
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


