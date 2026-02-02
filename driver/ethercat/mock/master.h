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
#include <span>
#include <unordered_map>
#include <vector>

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/master/master.h"

namespace ethercat::mock {
/// @brief key for PDO offset cache lookup.
struct PDOEntryKey {
    uint16_t slave_position;
    uint16_t index;
    uint8_t subindex;
    bool is_input;

    bool operator==(const PDOEntryKey &other) const {
        return slave_position == other.slave_position && index == other.index &&
               subindex == other.subindex && is_input == other.is_input;
    }
};

/// @brief hash function for PDOEntryKey.
struct PDOEntryKeyHash {
    size_t operator()(const PDOEntryKey &key) const {
        return std::hash<uint64_t>()(
            (static_cast<uint64_t>(key.slave_position) << 32) |
            (static_cast<uint64_t>(key.index) << 16) |
            (static_cast<uint64_t>(key.subindex) << 8) |
            static_cast<uint64_t>(key.is_input)
        );
    }
};

/// @brief configuration for a simulated slave device.
struct MockSlaveConfig {
    uint16_t position;
    uint32_t vendor_id;
    uint32_t product_code;
    uint32_t revision;
    uint32_t serial;
    std::string name;
    std::vector<PDOEntryInfo> input_pdos;
    std::vector<PDOEntryInfo> output_pdos;
    bool pdos_discovered;
    std::string pdo_discovery_error;

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
        name(std::move(name)),
        pdos_discovered(false) {}

    MockSlaveConfig(
        const uint16_t position,
        const uint32_t vendor_id,
        const uint32_t product_code,
        const uint32_t serial,
        std::string name
    ):
        position(position),
        vendor_id(vendor_id),
        product_code(product_code),
        revision(0),
        serial(serial),
        name(std::move(name)),
        pdos_discovered(false) {}

    MockSlaveConfig &with_input_pdos(std::vector<PDOEntryInfo> pdos) {
        this->input_pdos = std::move(pdos);
        this->pdos_discovered = true;
        return *this;
    }

    MockSlaveConfig &with_output_pdos(std::vector<PDOEntryInfo> pdos) {
        this->output_pdos = std::move(pdos);
        this->pdos_discovered = true;
        return *this;
    }

    MockSlaveConfig &with_pdo_discovery_error(std::string error) {
        this->pdo_discovery_error = std::move(error);
        return *this;
    }
};

/// @brief mock implementation of Master for testing without real hardware.
class Master final : public ethercat::master::Master {
    std::string iface_name;
    std::vector<SlaveInfo> slave_list;
    std::unordered_map<uint16_t, SlaveState> slave_states;
    std::unordered_map<PDOEntryKey, master::PDOOffset, PDOEntryKeyHash>
        pdo_offset_cache;
    bool initialized;
    bool activated;
    mutable std::mutex mu;

    xerrors::Error inject_init_err;
    xerrors::Error inject_activate_err;
    xerrors::Error inject_receive_err;
    xerrors::Error inject_send_err;

    std::unordered_map<uint16_t, SlaveState> state_transition_failures;

    std::vector<uint8_t> input_iomap;
    std::vector<uint8_t> output_iomap;
    size_t input_sz;
    size_t output_sz;
    std::vector<std::string> calls;
    size_t init_calls;
    std::vector<PDOEntry> registered_pdos;

public:
    explicit Master(std::string interface_name = "mock0"):
        iface_name(std::move(interface_name)),
        initialized(false),
        activated(false),
        input_sz(0),
        output_sz(0),
        init_calls(0) {}

    /// @brief adds a simulated slave to the mock master.
    void add_slave(const MockSlaveConfig &config) {
        SlaveInfo slave(
            config.position,
            config.vendor_id,
            config.product_code,
            config.revision,
            config.serial,
            config.name,
            SlaveState::INIT
        );
        slave.input_pdos = config.input_pdos;
        slave.output_pdos = config.output_pdos;
        slave.pdos_discovered = config.pdos_discovered;
        slave.pdo_discovery_error = config.pdo_discovery_error;
        this->slave_list.push_back(std::move(slave));
        this->slave_states[config.position] = SlaveState::INIT;
    }

    /// @brief injects an error to be returned by initialize().
    void inject_init_error(const xerrors::Error &err) { this->inject_init_err = err; }

    /// @brief injects an error to be returned by activate().
    void inject_activate_error(const xerrors::Error &err) {
        this->inject_activate_err = err;
    }

    /// @brief injects an error to be returned by receive().
    void inject_receive_error(const xerrors::Error &err) {
        this->inject_receive_err = err;
    }

    /// @brief injects an error to be returned by send().
    void inject_send_error(const xerrors::Error &err) { this->inject_send_err = err; }

    /// @brief clears all injected errors.
    void clear_injected_errors() {
        this->inject_init_err = xerrors::NIL;
        this->inject_activate_err = xerrors::NIL;
        this->inject_receive_err = xerrors::NIL;
        this->inject_send_err = xerrors::NIL;
    }

    /// @brief sets a slave to fail state transition to the given target state.
    void
    set_slave_transition_failure(const uint16_t position, const SlaveState target) {
        this->state_transition_failures[position] = target;
    }

    /// @brief directly sets the state of a specific slave.
    void set_slave_state(const uint16_t position, const SlaveState state) {
        std::lock_guard lock(this->mu);
        this->slave_states[position] = state;
        for (auto &slave: this->slave_list)
            if (slave.position == position) slave.state = state;
    }

    /// @brief returns the log of method calls for verification.
    const std::vector<std::string> &call_log() const { return this->calls; }

    /// @brief clears the method call log.
    void clear_call_log() { this->calls.clear(); }

    /// @brief checks if a specific method was called.
    bool was_called(const std::string &method) const {
        return std::find(this->calls.begin(), this->calls.end(), method) !=
               this->calls.end();
    }

    xerrors::Error initialize() override {
        std::lock_guard lock(this->mu);
        this->calls.push_back("initialize");
        this->init_calls++;
        if (this->inject_init_err) return this->inject_init_err;
        this->initialized = true;
        return xerrors::NIL;
    }

    xerrors::Error register_pdos(const std::vector<PDOEntry> &entries) override {
        std::lock_guard lock(this->mu);
        this->calls.push_back("register_pdos");
        this->registered_pdos = entries;
        return xerrors::NIL;
    }

    xerrors::Error activate() override {
        std::lock_guard lock(this->mu);
        this->calls.push_back("activate");
        if (this->inject_activate_err) return this->inject_activate_err;
        if (!this->initialized)
            return xerrors::Error(ACTIVATION_ERROR, "master not initialized");
        this->activated = true;
        this->input_sz = 0;
        this->output_sz = 0;
        if (!this->registered_pdos.empty()) {
            for (const auto &pdo: this->registered_pdos) {
                if (pdo.is_input)
                    this->input_sz += pdo.byte_length();
                else
                    this->output_sz += pdo.byte_length();
            }
        } else {
            for (const auto &slave: this->slave_list) {
                for (const auto &pdo: slave.input_pdos)
                    this->input_sz += pdo.byte_length();
                for (const auto &pdo: slave.output_pdos)
                    this->output_sz += pdo.byte_length();
            }
        }
        if (this->input_sz == 0) this->input_sz = this->slave_list.size() * 4;
        if (this->output_sz == 0) this->output_sz = this->slave_list.size() * 4;
        this->input_iomap.resize(this->input_sz, 0);
        this->output_iomap.resize(this->output_sz, 0);
        this->cache_pdo_offsets();
        for (auto &[pos, state]: this->slave_states) {
            auto it = this->state_transition_failures.find(pos);
            if (it != this->state_transition_failures.end() &&
                it->second == SlaveState::OP)
                state = SlaveState::SAFE_OP;
            else
                state = SlaveState::OP;
        }
        for (auto &slave: this->slave_list) {
            auto it = this->state_transition_failures.find(slave.position);
            if (it != this->state_transition_failures.end() &&
                it->second == SlaveState::OP)
                slave.state = SlaveState::SAFE_OP;
            else
                slave.state = SlaveState::OP;
        }
        return xerrors::NIL;
    }

    void deactivate() override {
        std::lock_guard lock(this->mu);
        this->calls.push_back("deactivate");
        this->activated = false;
        this->initialized = false;
        this->pdo_offset_cache.clear();
        this->registered_pdos.clear();
        this->input_sz = 0;
        this->output_sz = 0;
        for (auto &[pos, state]: this->slave_states)
            state = SlaveState::INIT;
        for (auto &slave: this->slave_list)
            slave.state = SlaveState::INIT;
    }

    xerrors::Error receive() override {
        std::lock_guard lock(this->mu);
        this->calls.push_back("receive");
        if (this->inject_receive_err) return this->inject_receive_err;
        return xerrors::NIL;
    }

    xerrors::Error send() override {
        std::lock_guard lock(this->mu);
        this->calls.push_back("send");
        if (this->inject_send_err) return this->inject_send_err;
        return xerrors::NIL;
    }

    std::span<const uint8_t> input_data() override {
        if (!this->activated) return {};
        return {this->input_iomap.data(), this->input_sz};
    }

    std::span<uint8_t> output_data() override {
        if (!this->activated) return {};
        return {this->output_iomap.data(), this->output_sz};
    }

    master::PDOOffset pdo_offset(const PDOEntry &entry) const override {
        std::lock_guard lock(this->mu);
        PDOEntryKey key{
            entry.slave_position,
            entry.index,
            entry.subindex,
            entry.is_input
        };
        auto it = this->pdo_offset_cache.find(key);
        if (it != this->pdo_offset_cache.end()) return it->second;
        return {};
    }

    std::vector<SlaveInfo> slaves() const override {
        std::lock_guard lock(this->mu);
        return this->slave_list;
    }

    SlaveState slave_state(const uint16_t position) const override {
        std::lock_guard lock(this->mu);
        const auto it = this->slave_states.find(position);
        if (it == this->slave_states.end()) return SlaveState::UNKNOWN;
        return it->second;
    }

    bool all_slaves_operational() const override {
        std::lock_guard lock(this->mu);
        for (const auto &[pos, state]: this->slave_states)
            if (state != SlaveState::OP) return false;
        return !this->slave_states.empty();
    }

    std::string interface_name() const override { return this->iface_name; }

    /// @brief returns whether the master has been initialized.
    bool is_initialized() const {
        std::lock_guard lock(this->mu);
        return this->initialized;
    }

    /// @brief returns whether the master has been activated.
    bool is_activated() const {
        std::lock_guard lock(this->mu);
        return this->activated;
    }

    /// @brief returns the number of slaves configured.
    size_t slave_count() const {
        std::lock_guard lock(this->mu);
        return this->slave_list.size();
    }

    /// @brief checks if any slaves have the given state.
    bool has_slave_in_state(const SlaveState state) const {
        std::lock_guard lock(this->mu);
        for (const auto &[pos, s]: this->slave_states)
            if (s == state) return true;
        return false;
    }

    /// @brief returns the count of slaves in the given state.
    size_t slaves_in_state(const SlaveState state) const {
        std::lock_guard lock(this->mu);
        size_t count = 0;
        for (const auto &[pos, s]: this->slave_states)
            if (s == state) ++count;
        return count;
    }

    /// @brief returns the number of times initialize() was called.
    size_t init_call_count() const {
        std::lock_guard lock(this->mu);
        return this->init_calls;
    }

    /// @brief resets the initialize call counter.
    void reset_init_call_count() {
        std::lock_guard lock(this->mu);
        this->init_calls = 0;
    }

    /// @brief sets a value in the input region for testing.
    template<typename T>
    void set_input(const size_t offset, const T value) {
        if (offset + sizeof(T) <= this->input_sz)
            std::memcpy(this->input_iomap.data() + offset, &value, sizeof(T));
    }

    /// @brief gets a value from the output region for verification.
    template<typename T>
    T get_output(const size_t offset) const {
        T value{};
        if (offset + sizeof(T) <= this->output_sz)
            std::memcpy(&value, this->output_iomap.data() + offset, sizeof(T));
        return value;
    }

private:
    void cache_pdo_offsets() {
        this->pdo_offset_cache.clear();
        size_t input_byte_offset = 0;
        size_t output_byte_offset = 0;
        if (!this->registered_pdos.empty()) {
            for (const auto &pdo: this->registered_pdos) {
                PDOEntryKey key{
                    pdo.slave_position,
                    pdo.index,
                    pdo.subindex,
                    pdo.is_input
                };
                if (pdo.is_input) {
                    this->pdo_offset_cache[key] = master::PDOOffset{
                        input_byte_offset,
                        0
                    };
                    input_byte_offset += pdo.byte_length();
                } else {
                    this->pdo_offset_cache[key] = master::PDOOffset{
                        output_byte_offset,
                        0
                    };
                    output_byte_offset += pdo.byte_length();
                }
            }
        } else {
            for (const auto &slave: this->slave_list) {
                for (const auto &pdo: slave.input_pdos) {
                    PDOEntryKey key{slave.position, pdo.index, pdo.subindex, true};
                    this->pdo_offset_cache[key] = master::PDOOffset{
                        input_byte_offset,
                        0
                    };
                    input_byte_offset += pdo.byte_length();
                }
                for (const auto &pdo: slave.output_pdos) {
                    PDOEntryKey key{slave.position, pdo.index, pdo.subindex, false};
                    this->pdo_offset_cache[key] = master::PDOOffset{
                        output_byte_offset,
                        0
                    };
                    output_byte_offset += pdo.byte_length();
                }
            }
        }
    }
};

/// @brief mock implementation of master::Manager for testing.
class Manager final : public master::Manager {
    std::vector<master::Info> infos;
    std::unordered_map<std::string, std::shared_ptr<Master>> masters;

public:
    /// @brief configures a master to be returned by enumerate() and create().
    void configure(const std::string &key, std::shared_ptr<Master> m) {
        this->infos.push_back({key, "Mock " + key});
        this->masters[key] = std::move(m);
    }

    [[nodiscard]] std::vector<master::Info> enumerate() override { return this->infos; }

    [[nodiscard]] std::pair<std::shared_ptr<master::Master>, xerrors::Error>
    create(const std::string &key) override {
        auto it = this->masters.find(key);
        if (it != this->masters.end()) return {it->second, xerrors::NIL};
        return {
            nullptr,
            xerrors::Error(
                MASTER_INIT_ERROR,
                "no mock master configured for key: " + key
            )
        };
    }
};

}
