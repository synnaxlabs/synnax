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

/// external
#include "modbus/modbus.h"

/// internal
#include "x/cpp/xerrors/errors.h"


namespace modbus::mock {

/// @brief Configuration for a mock Modbus slave
struct SlaveConfig {
    // Maps for storing configured values for each register type
    std::unordered_map<int, uint8_t> coils;
    std::unordered_map<int, uint8_t> discrete_inputs;
    std::unordered_map<int, uint16_t> holding_registers;
    std::unordered_map<int, uint16_t> input_registers;
    
    // IP and port configuration
    std::string ip_address = "127.0.0.1";
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
    std::mutex mutex_;
    SlaveConfig config_;

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
        const int max_coil = get_max_address(config_.coils);
        const int max_discrete = get_max_address(config_.discrete_inputs);
        const int max_holding = get_max_address(config_.holding_registers);
        const int max_input = get_max_address(config_.input_registers);

        // Add 1 to get the size (addresses are 0-based)
        const int nb_bits = max_coil >= 0 ? max_coil + 1 : 0;
        const int nb_input_bits = max_discrete >= 0 ? max_discrete + 1 : 0;
        const int nb_registers = max_holding >= 0 ? max_holding + 1 : 0;
        const int nb_input_registers = max_input >= 0 ? max_input + 1 : 0;

        // Create the mapping
        modbus_mapping_t* mb_mapping = modbus_mapping_new(nb_bits, nb_input_bits, 
                                                         nb_registers, nb_input_registers);
        if (mb_mapping == nullptr) return nullptr;

        for (const auto& pair : config_.coils)
            if (pair.first < nb_bits)
                mb_mapping->tab_bits[pair.first] = pair.second ? 1 : 0;

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

public:
    /// @brief Create a new Modbus TCP slave with the given configuration
    /// @param config The slave configuration
    explicit Slave(const SlaveConfig& config) 
        : running_(false),
          ip_address_(config.ip_address),
          port_(config.port),
          socket_(-1),
          config_(config) {
        
        ctx_ = modbus_new_tcp(ip_address_.c_str(), port_);
        if (ctx_ == nullptr) {
            throw std::runtime_error("Failed to create modbus context");
        }
    }

    ~Slave() {
        stop();
        if (ctx_ != nullptr) {
            modbus_free(ctx_);
        }
    }

    /// @brief Update the slave configuration
    /// @param config The new configuration
    void update_config(const SlaveConfig& config) {
        std::lock_guard lock(mutex_);
        config_ = config;
    }

    /// @brief Start the slave server in a background thread
    xerrors::Error start() {
        if (running_) {
            return xerrors::NIL;  // Already running
        }

        socket_ = modbus_tcp_listen(ctx_, 1);
        if (socket_ == -1) {
            return xerrors::Error("Failed to listen on modbus socket: " + 
                                 std::string(modbus_strerror(errno)));
        }

        modbus_set_debug(ctx_, FALSE);
        running_ = true;

        server_thread_ = std::thread([this] {
            uint8_t query[MODBUS_TCP_MAX_ADU_LENGTH];

            while (running_) {
                fd_set refset;
                FD_ZERO(&refset);
                FD_SET(socket_, &refset);

                // Use a timeout to allow checking the running flag
                timeval tv = {0, 100000};  // 0 seconds, 100000 microseconds (100ms)

                int rc = select(socket_ + 1, &refset, nullptr, nullptr, &tv);
                if (rc == -1) {
                    if (errno == EINTR) {
                        continue;
                    }
                    break;
                }

                if (rc == 0) {
                    continue;  // Timeout, check running flag
                }

                if (FD_ISSET(socket_, &refset)) {
                    modbus_set_socket(ctx_, socket_);
                    rc = modbus_receive(ctx_, query);
                    if (rc > 0) {
                        std::lock_guard lock(mutex_);
                        // Create a fresh mapping with current values
                        modbus_mapping_t* mb_mapping = create_mapping();
                        if (mb_mapping != nullptr) {
                            modbus_reply(ctx_, query, rc, mb_mapping);
                            modbus_mapping_free(mb_mapping);
                        }
                    } else if (rc == -1 && errno != EMBBADCRC) {
                        // Connection closed or error
                        break;
                    }
                }
            }
        });

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
};

} // namespace modbus::mock


