// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstring>
#include <map>
#include <string>
#include <vector>

#include "driver/ethercat/soem/api.h"

namespace driver::ethercat::soem {

/// @brief configurable properties for a single mock slave.
struct MockSlaveInfo {
    uint16_t state = EC_STATE_INIT;
    uint32_t eep_man = 0;
    uint32_t eep_id = 0;
    uint32_t eep_rev = 0;
    uint32_t eep_ser = 0;
    std::string name;
    uint16_t Ibits = 0;
    uint16_t Obits = 0;
    uint8_t group = 0;
    size_t output_offset = 0;
    size_t input_offset = 0;
    uint16_t mbx_proto = 0;
    uint16_t ALstatuscode = 0;
};

/// @brief mock implementation of soem::API for unit testing.
class MockAPI final : public API {
public:
    std::vector<MockSlaveInfo> slave_infos;
    int init_return = 1;
    int config_init_return = 1;
    int config_map_return = 64;
    int send_return = 1;
    int receive_return = 3;
    int writestate_return = 1;
    std::map<uint16_t, uint16_t> statecheck_overrides;
    uint32_t ibytes = 0;
    uint32_t obytes = 0;
    uint16_t outputs_wkc = 1;
    uint16_t inputs_wkc = 1;
    int16_t siifind_return = 0;
    int readODlist_return = 0;
    bool close_called = false;
    bool init_called = false;
    int send_count = 0;
    int receive_count = 0;
    std::vector<std::pair<uint16_t, uint16_t>> state_change_requests;

private:
    uint8_t *iomap_ptr = nullptr;

public:
    int init(const char *) override {
        this->init_called = true;
        return this->init_return;
    }

    void close() override { this->close_called = true; }

    int config_init() override { return this->config_init_return; }

    int config_map_group(void *iomap, uint8_t) override {
        this->iomap_ptr = static_cast<uint8_t *>(iomap);
        return this->config_map_return;
    }

    int send_processdata() override {
        this->send_count++;
        return this->send_return;
    }

    int receive_processdata(int) override {
        this->receive_count++;
        return this->receive_return;
    }

    int writestate(uint16_t) override { return this->writestate_return; }

    uint16_t statecheck(uint16_t, uint16_t reqstate, int) override {
        auto it = this->statecheck_overrides.find(reqstate);
        if (it != this->statecheck_overrides.end()) return it->second;
        return reqstate;
    }

    int SDOread(uint16_t, uint16_t, uint8_t, int, int *, void *, int) override {
        return 0;
    }

    int16_t siifind(uint16_t, uint16_t) override { return this->siifind_return; }

    uint8_t siigetbyte(uint16_t, uint16_t) override { return 0; }

    void siistring(char *str, uint16_t, uint16_t) override { str[0] = '\0'; }

    int readODlist(uint16_t, ec_ODlistt *) override { return this->readODlist_return; }

    int readOEsingle(uint16_t, uint8_t, ec_ODlistt *, ec_OElistt *) override {
        return 0;
    }

    int slave_count() const override {
        return static_cast<int>(this->slave_infos.size());
    }

    uint16_t slave_state(uint16_t pos) const override {
        if (pos == 0 || pos > this->slave_infos.size()) return 0;
        return this->slave_infos[pos - 1].state;
    }

    void set_slave_state(uint16_t pos, uint16_t state) override {
        if (pos == 0 || pos > this->slave_infos.size()) return;
        this->slave_infos[pos - 1].state = state;
        this->state_change_requests.emplace_back(pos, state);
    }

    uint32_t slave_eep_man(uint16_t pos) const override {
        if (pos == 0 || pos > this->slave_infos.size()) return 0;
        return this->slave_infos[pos - 1].eep_man;
    }

    uint32_t slave_eep_id(uint16_t pos) const override {
        if (pos == 0 || pos > this->slave_infos.size()) return 0;
        return this->slave_infos[pos - 1].eep_id;
    }

    uint32_t slave_eep_rev(uint16_t pos) const override {
        if (pos == 0 || pos > this->slave_infos.size()) return 0;
        return this->slave_infos[pos - 1].eep_rev;
    }

    uint32_t slave_eep_ser(uint16_t pos) const override {
        if (pos == 0 || pos > this->slave_infos.size()) return 0;
        return this->slave_infos[pos - 1].eep_ser;
    }

    std::string slave_name(uint16_t pos) const override {
        if (pos == 0 || pos > this->slave_infos.size()) return "";
        return this->slave_infos[pos - 1].name;
    }

    uint16_t slave_Ibits(uint16_t pos) const override {
        if (pos == 0 || pos > this->slave_infos.size()) return 0;
        return this->slave_infos[pos - 1].Ibits;
    }

    uint16_t slave_Obits(uint16_t pos) const override {
        if (pos == 0 || pos > this->slave_infos.size()) return 0;
        return this->slave_infos[pos - 1].Obits;
    }

    uint8_t slave_group(uint16_t pos) const override {
        if (pos == 0 || pos > this->slave_infos.size()) return 0;
        return this->slave_infos[pos - 1].group;
    }

    void set_slave_group(uint16_t pos, uint8_t group) override {
        if (pos == 0 || pos > this->slave_infos.size()) return;
        this->slave_infos[pos - 1].group = group;
    }

    uint8_t *slave_outputs(uint16_t pos) const override {
        if (pos == 0 || pos > this->slave_infos.size()) return nullptr;
        if (this->iomap_ptr == nullptr) return nullptr;
        return this->iomap_ptr + this->slave_infos[pos - 1].output_offset;
    }

    uint8_t *slave_inputs(uint16_t pos) const override {
        if (pos == 0 || pos > this->slave_infos.size()) return nullptr;
        if (this->iomap_ptr == nullptr) return nullptr;
        return this->iomap_ptr + this->slave_infos[pos - 1].input_offset;
    }

    uint16_t slave_mbx_proto(uint16_t pos) const override {
        if (pos == 0 || pos > this->slave_infos.size()) return 0;
        return this->slave_infos[pos - 1].mbx_proto;
    }

    uint16_t slave_ALstatuscode(uint16_t pos) const override {
        if (pos == 0 || pos > this->slave_infos.size()) return 0;
        return this->slave_infos[pos - 1].ALstatuscode;
    }

    uint32_t group_Ibytes(uint8_t) const override { return this->ibytes; }

    uint32_t group_Obytes(uint8_t) const override { return this->obytes; }

    uint16_t group_outputsWKC(uint8_t) const override { return this->outputs_wkc; }

    uint16_t group_inputsWKC(uint8_t) const override { return this->inputs_wkc; }
};

}
