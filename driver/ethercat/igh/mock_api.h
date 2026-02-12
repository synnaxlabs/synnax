// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <cstring>
#include <unordered_map>
#include <vector>

#include "driver/ethercat/igh/api.h"

namespace driver::ethercat::igh {

/// @brief mock implementation of the IgH API for unit testing without hardware.
class MockAPI final : public API {
    /// @brief sentinel pointer for the ec_master_t handle.
    ec_master_t *sentinel_master = reinterpret_cast<ec_master_t *>(0x1);
    /// @brief sentinel pointer for the output domain handle.
    ec_domain_t *sentinel_output_domain = reinterpret_cast<ec_domain_t *>(0x2);
    /// @brief sentinel pointer for the input domain handle.
    ec_domain_t *sentinel_input_domain = reinterpret_cast<ec_domain_t *>(0x3);
    /// @brief tracks the number of domains created for returning alternating sentinels.
    int domain_create_count = 0;
    /// @brief configurable return value for request_master.
    ec_master_t *request_master_result;
    /// @brief configurable return value for master() (get info).
    int master_info_result = 0;
    /// @brief configurable master info to populate.
    ec_master_info_t mock_master_info{};
    /// @brief configurable slave info entries.
    std::vector<ec_slave_info_t> mock_slaves;
    /// @brief configurable return value for master_create_domain.
    bool create_domain_fails_output = false;
    /// @brief configurable return value for master_create_domain (input).
    bool create_domain_fails_input = false;
    /// @brief configurable return value for master_activate.
    int activate_result = 0;
    /// @brief configurable return value for master_slave_config.
    bool slave_config_fails = false;
    /// @brief configurable return value for slave_config_pdos.
    int slave_config_pdos_result = 0;

    /// @brief running byte offset per domain for slave_config_reg_pdo_entry.
    size_t output_domain_offset = 0;
    /// @brief running byte offset per domain for slave_config_reg_pdo_entry.
    size_t input_domain_offset = 0;

    /// @brief configurable output domain size.
    size_t output_domain_sz = 0;
    /// @brief configurable input domain size.
    size_t input_domain_sz = 0;
    /// @brief output domain data buffer.
    std::vector<uint8_t> output_data_buf;
    /// @brief input domain data buffer.
    std::vector<uint8_t> input_data_buf;

    /// @brief configurable domain state for output domain.
    ec_domain_state_t output_domain_state_val{};
    /// @brief configurable domain state for input domain.
    ec_domain_state_t input_domain_state_val{};

    /// @brief configurable slave config states keyed by slave config pointer.
    std::unordered_map<uintptr_t, ec_slave_config_state_t> slave_config_states;
    /// @brief tracks slave config pointers per position.
    std::unordered_map<uint16_t, ec_slave_config_t *> created_slave_configs;
    /// @brief next sentinel address for slave config allocation.
    uintptr_t next_slave_config_addr = 0x100;

    /// @brief tracks whether release_master was called.
    bool release_master_called_val = false;
    /// @brief tracks whether master_deactivate was called.
    bool master_deactivate_called_val = false;

public:
    MockAPI(): API(MockTag{}) { this->request_master_result = this->sentinel_master; }

    void set_request_master_result(ec_master_t *result) {
        this->request_master_result = result;
    }

    void set_master_info_result(int result) { this->master_info_result = result; }

    void set_slave_count(unsigned int count) {
        this->mock_master_info.slave_count = count;
    }

    void add_slave(
        uint16_t position,
        uint32_t vendor_id,
        uint32_t product_code,
        uint32_t revision,
        uint32_t serial,
        const std::string &name
    ) {
        ec_slave_info_t info{};
        info.position = position;
        info.vendor_id = vendor_id;
        info.product_code = product_code;
        info.revision_number = revision;
        info.serial_number = serial;
        std::strncpy(info.name, name.c_str(), EC_MAX_STRING_LENGTH - 1);
        info.name[EC_MAX_STRING_LENGTH - 1] = '\0';

        if (this->mock_slaves.size() <= position)
            this->mock_slaves.resize(position + 1);
        this->mock_slaves[position] = info;
        this->mock_master_info.slave_count = static_cast<unsigned int>(
            this->mock_slaves.size()
        );
    }

    void set_create_domain_fails_output(bool fails) {
        this->create_domain_fails_output = fails;
    }

    void set_create_domain_fails_input(bool fails) {
        this->create_domain_fails_input = fails;
    }

    void set_activate_result(int result) { this->activate_result = result; }

    void set_slave_config_fails(bool fails) { this->slave_config_fails = fails; }

    void set_slave_config_pdos_result(int result) {
        this->slave_config_pdos_result = result;
    }

    void set_output_domain_size(size_t sz) {
        this->output_domain_sz = sz;
        this->output_data_buf.resize(sz, 0);
    }

    void set_input_domain_size(size_t sz) {
        this->input_domain_sz = sz;
        this->input_data_buf.resize(sz, 0);
    }

    void set_output_domain_state(ec_domain_state_t state) {
        this->output_domain_state_val = state;
    }

    void set_input_domain_state(ec_domain_state_t state) {
        this->input_domain_state_val = state;
    }

    void set_slave_config_state(uint16_t position, ec_slave_config_state_t state) {
        auto it = this->created_slave_configs.find(position);
        if (it != this->created_slave_configs.end())
            this->slave_config_states[reinterpret_cast<uintptr_t>(it->second)] = state;
    }

    void set_all_slaves_al_state(uint8_t al_state) {
        for (auto &[addr, state]: this->slave_config_states)
            state.al_state = al_state;
    }

    bool release_master_called() const { return this->release_master_called_val; }

    bool master_deactivate_called() const { return this->master_deactivate_called_val; }

    ec_domain_t *get_output_domain() const { return this->sentinel_output_domain; }

    ec_domain_t *get_input_domain() const { return this->sentinel_input_domain; }

    ec_master_t *request_master(unsigned int) const override {
        return this->request_master_result;
    }

    void release_master(ec_master_t *) const override {
        const_cast<MockAPI *>(this)->release_master_called_val = true;
    }

    int master_activate(ec_master_t *) const override { return this->activate_result; }

    int master_deactivate(ec_master_t *) const override {
        const_cast<MockAPI *>(this)->master_deactivate_called_val = true;
        return 0;
    }

    ec_domain_t *master_create_domain(ec_master_t *) const override {
        auto *self = const_cast<MockAPI *>(this);
        self->domain_create_count++;
        if (self->domain_create_count == 1) {
            if (this->create_domain_fails_output) return nullptr;
            return this->sentinel_output_domain;
        }
        if (this->create_domain_fails_input) return nullptr;
        return this->sentinel_input_domain;
    }

    size_t domain_size(const ec_domain_t *domain) const override {
        if (domain == this->sentinel_output_domain) return this->output_domain_sz;
        if (domain == this->sentinel_input_domain) return this->input_domain_sz;
        return 0;
    }

    uint8_t *domain_data(const ec_domain_t *domain) const override {
        auto *self = const_cast<MockAPI *>(this);
        if (domain == this->sentinel_output_domain) {
            if (self->output_data_buf.empty()) return nullptr;
            return self->output_data_buf.data();
        }
        if (domain == this->sentinel_input_domain) {
            if (self->input_data_buf.empty()) return nullptr;
            return self->input_data_buf.data();
        }
        return nullptr;
    }

    int domain_process(ec_domain_t *) const override { return 0; }

    int domain_queue(ec_domain_t *) const override { return 0; }

    int
    domain_state(const ec_domain_t *domain, ec_domain_state_t *state) const override {
        if (domain == this->sentinel_output_domain)
            *state = this->output_domain_state_val;
        else if (domain == this->sentinel_input_domain)
            *state = this->input_domain_state_val;
        return 0;
    }

    int master_send(ec_master_t *) const override { return 0; }

    int master_receive(ec_master_t *) const override { return 0; }

    int master(ec_master_t *, ec_master_info_t *info) const override {
        if (this->master_info_result < 0) return this->master_info_result;
        *info = this->mock_master_info;
        return this->master_info_result;
    }

    int master_get_slave(
        ec_master_t *,
        uint16_t slave_position,
        ec_slave_info_t *slave_info
    ) const override {
        if (slave_position >= this->mock_slaves.size()) return -1;
        *slave_info = this->mock_slaves[slave_position];
        return 0;
    }

    ec_slave_config_t *master_slave_config(
        ec_master_t *,
        uint16_t,
        uint16_t position,
        uint32_t,
        uint32_t
    ) const override {
        if (this->slave_config_fails) return nullptr;
        auto *self = const_cast<MockAPI *>(this);
        auto it = self->created_slave_configs.find(position);
        if (it != self->created_slave_configs.end()) return it->second;
        auto *sc = reinterpret_cast<ec_slave_config_t *>(self->next_slave_config_addr);
        self->next_slave_config_addr += 0x10;
        self->created_slave_configs[position] = sc;
        ec_slave_config_state_t initial_state{};
        initial_state.al_state = 0x01;
        initial_state.online = 1;
        self->slave_config_states[reinterpret_cast<uintptr_t>(sc)] = initial_state;
        return sc;
    }

    int slave_config_state(
        const ec_slave_config_t *sc,
        ec_slave_config_state_t *state
    ) const override {
        auto it = this->slave_config_states.find(reinterpret_cast<uintptr_t>(sc));
        if (it != this->slave_config_states.end()) {
            *state = it->second;
            return 0;
        }
        std::memset(state, 0, sizeof(*state));
        return 0;
    }

    int slave_config_pdos(
        ec_slave_config_t *,
        unsigned int,
        const ec_sync_info_t[]
    ) const override {
        return this->slave_config_pdos_result;
    }

    int slave_config_reg_pdo_entry(
        ec_slave_config_t *,
        uint16_t,
        uint8_t,
        ec_domain_t *domain,
        unsigned int *bit_position
    ) const override {
        auto *self = const_cast<MockAPI *>(this);
        *bit_position = 0;
        if (domain == this->sentinel_output_domain) {
            size_t offset = self->output_domain_offset;
            self->output_domain_offset += 2;
            return static_cast<int>(offset);
        }
        size_t offset = self->input_domain_offset;
        self->input_domain_offset += 2;
        return static_cast<int>(offset);
    }

    int master_get_sync_manager(
        ec_master_t *,
        uint16_t,
        uint8_t,
        ec_sync_info_t *
    ) const override {
        return -1;
    }

    int master_get_pdo(
        ec_master_t *,
        uint16_t,
        uint8_t,
        uint16_t,
        ec_pdo_info_t *
    ) const override {
        return -1;
    }

    int master_get_pdo_entry(
        ec_master_t *,
        uint16_t,
        uint8_t,
        uint16_t,
        uint16_t,
        ec_pdo_entry_info_t *
    ) const override {
        return -1;
    }
};

}
