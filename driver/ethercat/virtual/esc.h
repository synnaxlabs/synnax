// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <atomic>
#include <cstdint>
#include <cstring>
#include <mutex>
#include <span>
#include <string>
#include <thread>
#include <vector>

#include "driver/ethercat/slave/slave.h"
#include "driver/ethercat/virtual/config.h"
#include "driver/ethercat/virtual/datagram.h"
#include "driver/ethercat/virtual/frame.h"
#include "driver/ethercat/virtual/object_dictionary.h"
#include "driver/ethercat/virtual/state_machine.h"
#include "x/cpp/xerrors/errors.h"

namespace ethercat::virtual_esc {

/// Error codes for Virtual ESC.
const xerrors::Error BASE_ERROR("virtual_esc");
const xerrors::Error SOCKET_ERROR = BASE_ERROR.sub("socket");
const xerrors::Error BIND_ERROR = BASE_ERROR.sub("bind");
const xerrors::Error RECEIVE_ERROR = BASE_ERROR.sub("receive");
const xerrors::Error SEND_ERROR = BASE_ERROR.sub("send");

/// @brief Virtual EtherCAT Slave Controller for testing.
///
/// VirtualESC emulates an EtherCAT slave device using raw sockets on a Linux
/// veth pair. It can be used to test EtherCAT master implementations without
/// requiring physical hardware.
///
/// Usage:
/// @code
///     VethPair veth("ectest");
///     veth.create();
///     VirtualESC esc(default_test_config());
///     esc.start(veth.slave_interface());
///     // Master can now connect to veth.master_interface()
///     // ...
///     esc.stop();
/// @endcode
class VirtualESC {
public:
    /// @brief Constructs a VirtualESC with the given configuration.
    explicit VirtualESC(Config config);

    ~VirtualESC();

    VirtualESC(const VirtualESC&) = delete;
    VirtualESC& operator=(const VirtualESC&) = delete;

    /// @brief Starts the virtual ESC on the specified network interface.
    /// @param interface The network interface name (e.g., "ectest1").
    /// @return An error if the ESC could not be started.
    [[nodiscard]] xerrors::Error start(const std::string& interface);

    /// @brief Stops the virtual ESC.
    void stop();

    /// @brief Returns true if the ESC is running.
    [[nodiscard]] bool is_running() const { return this->running.load(); }

    /// @brief Returns the current EtherCAT state.
    [[nodiscard]] slave::State current_state() const;

    /// @brief Returns true if the slave is in OPERATIONAL state.
    [[nodiscard]] bool is_operational() const;

    /// @brief Returns the output data at the given offset (data master sent to slave).
    template<typename T>
    T get_output(size_t offset) const {
        std::lock_guard lock(this->mu);
        T value{};
        if (offset + sizeof(T) <= this->output_data.size())
            std::memcpy(&value, this->output_data.data() + offset, sizeof(T));
        return value;
    }

    /// @brief Sets the input data at the given offset (data slave sends to master).
    template<typename T>
    void set_input(size_t offset, T value) {
        std::lock_guard lock(this->mu);
        if (offset + sizeof(T) <= this->input_data.size())
            std::memcpy(this->input_data.data() + offset, &value, sizeof(T));
    }

    /// @brief Returns the input data at the given offset (data slave sends to master).
    template<typename T>
    T get_input(size_t offset) const {
        std::lock_guard lock(this->mu);
        T value{};
        if (offset + sizeof(T) <= this->input_data.size())
            std::memcpy(&value, this->input_data.data() + offset, sizeof(T));
        return value;
    }

    /// @brief Returns a copy of the current output data buffer.
    [[nodiscard]] std::vector<uint8_t> get_output_data() const {
        std::lock_guard lock(this->mu);
        return this->output_data;
    }

    /// @brief Returns a copy of the current input data buffer.
    [[nodiscard]] std::vector<uint8_t> get_input_data() const {
        std::lock_guard lock(this->mu);
        return this->input_data;
    }

    /// @brief Sets the entire input data buffer.
    void set_input_data(const std::vector<uint8_t>& data) {
        std::lock_guard lock(this->mu);
        const size_t copy_size = std::min(data.size(), this->input_data.size());
        std::memcpy(this->input_data.data(), data.data(), copy_size);
    }

    /// @brief Returns the configured station address.
    [[nodiscard]] uint16_t station_address() const { return this->config.station_address; }

    /// @brief Returns the number of frames processed.
    [[nodiscard]] uint64_t frames_processed() const {
        return this->frame_count.load();
    }

private:
    Config config;
    ObjectDictionary od;
    StateMachine state_machine;
    std::vector<uint8_t> input_data;
    std::vector<uint8_t> output_data;
    std::vector<uint8_t> registers;
    std::vector<uint16_t> sii_eeprom;
    uint16_t sii_address;
    mutable std::mutex mu;

    std::thread worker;
    std::atomic<bool> running;
    std::atomic<uint64_t> frame_count;
    int raw_socket;
    std::string iface_name;

    void run();
    void process_frame(Frame& frame);
    void handle_datagram(Datagram& dgram);

    void handle_broadcast_read(Datagram& dgram);
    void handle_broadcast_write(Datagram& dgram);
    void handle_auto_increment_read(Datagram& dgram);
    void handle_auto_increment_write(Datagram& dgram);
    void handle_configured_address_read(Datagram& dgram);
    void handle_configured_address_write(Datagram& dgram);
    void handle_logical_read(Datagram& dgram);
    void handle_logical_write(Datagram& dgram);
    void handle_logical_read_write(Datagram& dgram);

    bool read_register(uint16_t addr, std::span<uint8_t> data);
    bool write_register(uint16_t addr, std::span<const uint8_t> data);

    void init_object_dictionary();
    void init_registers();
    void init_sii_eeprom();
};

}
