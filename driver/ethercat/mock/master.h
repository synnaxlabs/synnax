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
#include <cstring>
#include <mutex>
#include <unordered_map>
#include <vector>

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/master/master.h"

namespace ethercat::mock {
/// Mock implementation of Domain for testing without real EtherCAT hardware.
///
/// Provides a simulated process data buffer that can be pre-populated with
/// test values or inspected after write operations.
class MockDomain final : public Domain {
    std::vector<uint8_t> data_;
    size_t input_size_;
    size_t output_size_;
    std::vector<std::pair<PDOEntry, size_t>> registered_pdos_;

public:
    MockDomain(): input_size_(0), output_size_(0) {}

    std::pair<size_t, xerrors::Error> register_pdo(const PDOEntry &entry) override {
        const size_t byte_len = entry.byte_length();
        size_t offset;
        if (entry.is_input) {
            offset = input_size_;
            input_size_ += byte_len;
        } else {
            offset = output_size_;
            output_size_ += byte_len;
        }
        registered_pdos_.emplace_back(entry, offset);
        data_.resize(input_size_ + output_size_, 0);
        return {offset, xerrors::NIL};
    }

    uint8_t *data() override { return data_.data(); }

    size_t size() const override { return data_.size(); }

    size_t input_size() const override { return input_size_; }

    size_t output_size() const override { return output_size_; }

    /// Sets a value in the input region of the domain buffer for testing.
    /// @tparam T The type of value to set.
    /// @param offset Byte offset into the input region.
    /// @param value The value to write.
    template<typename T>
    void set_input(const size_t offset, const T value) {
        if (offset + sizeof(T) <= input_size_)
            std::memcpy(data_.data() + offset, &value, sizeof(T));
    }

    /// Gets a value from the output region of the domain buffer for verification.
    /// @tparam T The type of value to get.
    /// @param offset Byte offset into the output region.
    /// @returns The value at the specified offset.
    template<typename T>
    T get_output(const size_t offset) const {
        T value{};
        if (offset + sizeof(T) <= output_size_)
            std::memcpy(&value, data_.data() + input_size_ + offset, sizeof(T));
        return value;
    }

    /// Returns the list of registered PDO entries with their offsets.
    const std::vector<std::pair<PDOEntry, size_t>> &registered_pdos() const {
        return registered_pdos_;
    }
};

/// Configuration for a simulated slave device.
struct MockSlaveConfig {
    uint16_t position;
    uint32_t vendor_id;
    uint32_t product_code;
    uint32_t revision;
    uint32_t serial;
    std::string name;

    MockSlaveConfig(
        const uint16_t position,
        const uint32_t vendor_id,
        const uint32_t product_code,
        std::string name
    ):
        position(position),
        vendor_id(vendor_id),
        product_code(product_code),
        revision(0),
        serial(0),
        name(std::move(name)) {}
};

/// Mock implementation of Master for testing without real EtherCAT hardware.
///
/// Simulates the EtherCAT master lifecycle and cyclic operations. Can be configured
/// with virtual slaves and inject errors for testing error handling paths.
class MockMaster final : public Master {
    std::string interface_name_;
    std::vector<SlaveInfo> slaves_;
    std::unordered_map<uint16_t, SlaveState> slave_states_;
    bool initialized_;
    bool activated_;
    mutable std::mutex mutex_;

    xerrors::Error inject_init_error_;
    xerrors::Error inject_activate_error_;
    xerrors::Error inject_receive_error_;
    xerrors::Error inject_send_error_;

    std::unique_ptr<MockDomain> active_domain_;
    std::vector<std::string> call_log_;

public:
    explicit MockMaster(std::string interface_name = "mock0"):
        interface_name_(std::move(interface_name)),
        initialized_(false),
        activated_(false) {}

    /// Adds a simulated slave to the mock master.
    /// Must be called before initialize().
    void add_slave(const MockSlaveConfig &config) {
        slaves_.emplace_back(
            config.position,
            config.vendor_id,
            config.product_code,
            config.revision,
            config.serial,
            config.name,
            SlaveState::INIT
        );
        slave_states_[config.position] = SlaveState::INIT;
    }

    /// Injects an error to be returned by initialize().
    void inject_init_error(const xerrors::Error &err) { inject_init_error_ = err; }

    /// Injects an error to be returned by activate().
    void inject_activate_error(const xerrors::Error &err) {
        inject_activate_error_ = err;
    }

    /// Injects an error to be returned by receive().
    void inject_receive_error(const xerrors::Error &err) { inject_receive_error_ = err; }

    /// Injects an error to be returned by send().
    void inject_send_error(const xerrors::Error &err) { inject_send_error_ = err; }

    /// Returns the log of method calls for verification.
    const std::vector<std::string> &call_log() const { return call_log_; }

    /// Clears the method call log.
    void clear_call_log() { call_log_.clear(); }

    /// Checks if a specific method was called.
    bool was_called(const std::string &method) const {
        return std::find(call_log_.begin(), call_log_.end(), method) != call_log_.end();
    }

    xerrors::Error initialize() override {
        std::lock_guard lock(mutex_);
        call_log_.push_back("initialize");
        if (inject_init_error_) return inject_init_error_;
        initialized_ = true;
        return xerrors::NIL;
    }

    std::unique_ptr<Domain> create_domain() override {
        std::lock_guard lock(mutex_);
        call_log_.push_back("create_domain");
        return std::make_unique<MockDomain>();
    }

    xerrors::Error activate() override {
        std::lock_guard lock(mutex_);
        call_log_.push_back("activate");
        if (inject_activate_error_) return inject_activate_error_;
        if (!initialized_)
            return xerrors::Error(ACTIVATION_ERROR, "master not initialized");
        activated_ = true;
        active_domain_ = std::make_unique<MockDomain>();
        for (auto &[pos, state]: slave_states_)
            state = SlaveState::OP;
        for (auto &slave: slaves_)
            slave.state = SlaveState::OP;
        return xerrors::NIL;
    }

    void deactivate() override {
        std::lock_guard lock(mutex_);
        call_log_.push_back("deactivate");
        activated_ = false;
        for (auto &[pos, state]: slave_states_)
            state = SlaveState::INIT;
        for (auto &slave: slaves_)
            slave.state = SlaveState::INIT;
    }

    xerrors::Error receive() override {
        std::lock_guard lock(mutex_);
        call_log_.push_back("receive");
        if (inject_receive_error_) return inject_receive_error_;
        return xerrors::NIL;
    }

    xerrors::Error process(Domain &) override {
        std::lock_guard lock(mutex_);
        call_log_.push_back("process");
        return xerrors::NIL;
    }

    xerrors::Error queue(Domain &) override {
        std::lock_guard lock(mutex_);
        call_log_.push_back("queue");
        return xerrors::NIL;
    }

    xerrors::Error send() override {
        std::lock_guard lock(mutex_);
        call_log_.push_back("send");
        if (inject_send_error_) return inject_send_error_;
        return xerrors::NIL;
    }

    std::vector<SlaveInfo> slaves() const override {
        std::lock_guard lock(mutex_);
        return slaves_;
    }

    SlaveState slave_state(const uint16_t position) const override {
        std::lock_guard lock(mutex_);
        const auto it = slave_states_.find(position);
        if (it == slave_states_.end()) return SlaveState::UNKNOWN;
        return it->second;
    }

    bool all_slaves_operational() const override {
        std::lock_guard lock(mutex_);
        for (const auto &[pos, state]: slave_states_)
            if (state != SlaveState::OP) return false;
        return !slave_states_.empty();
    }

    std::string interface_name() const override { return interface_name_; }

    SlaveDataOffsets slave_data_offsets(const uint16_t position) const override {
        std::lock_guard lock(mutex_);
        if (!activated_) return SlaveDataOffsets{};

        // For mock, calculate offsets based on slave position order.
        // Inputs come first in our mock layout, then outputs.
        size_t input_offset = 0;
        size_t output_offset = 0;

        // Find cumulative offsets for slaves before this position
        for (const auto &slave : slaves_) {
            if (slave.position == position) {
                // For mock, each slave has a default 4 bytes input + 4 bytes output
                const size_t slave_input_size = 4;
                const size_t slave_output_size = 4;
                return SlaveDataOffsets{
                    input_offset,
                    slave_input_size,
                    output_offset,
                    slave_output_size
                };
            }
            input_offset += 4;   // Default 4 bytes per slave input
            output_offset += 4;  // Default 4 bytes per slave output
        }
        return SlaveDataOffsets{};
    }

    Domain *active_domain() const override { return active_domain_.get(); }

    /// Returns whether the master has been initialized.
    bool is_initialized() const {
        std::lock_guard lock(mutex_);
        return initialized_;
    }

    /// Returns whether the master has been activated.
    bool is_activated() const {
        std::lock_guard lock(mutex_);
        return activated_;
    }
};
}
