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
    /// @brief per requested state, the number of statecheck polls that report the
    /// slave's current state before it reaches the requested one.
    std::map<uint16_t, int> statecheck_pending_polls;
    uint32_t ibytes = 0;
    uint32_t obytes = 0;
    uint16_t outputs_wkc = 1;
    uint16_t inputs_wkc = 1;
    /// @brief the SII address of each category, keyed by category.
    std::map<uint16_t, int16_t> sii_categories;
    /// @brief SII bytes returned by siigetbyte, keyed by address.
    std::map<uint16_t, uint8_t> sii_bytes;
    int readODlist_return = 0;
    bool close_called = false;
    bool init_called = false;
    int send_count = 0;
    int receive_count = 0;
    std::vector<std::pair<uint16_t, uint16_t>> state_change_requests;
    std::vector<uint8_t> processdata_groups;
    int mapped_group = -1;
    /// @brief CoE objects returned by SDOread, keyed by index and sub-index.
    std::map<std::pair<uint16_t, uint8_t>, uint32_t> sdo_objects;
    /// @brief entry names and CoE data types returned by readOEsingle, keyed by index
    /// and sub-index.
    std::map<std::pair<uint16_t, uint8_t>, std::pair<std::string, uint16_t>> oe_entries;

private:
    uint8_t *iomap_ptr = nullptr;

public:
    int init(const char *) override {
        this->init_called = true;
        return this->init_return;
    }

    void close() override { this->close_called = true; }

    int config_init() override { return this->config_init_return; }

    int config_map_group(void *iomap, uint8_t group) override {
        this->iomap_ptr = static_cast<uint8_t *>(iomap);
        this->mapped_group = group;
        return this->config_map_return;
    }

    int send_processdata(uint8_t group) override {
        this->send_count++;
        this->processdata_groups.push_back(group);
        return this->send_return;
    }

    int receive_processdata(uint8_t group, int) override {
        this->receive_count++;
        this->processdata_groups.push_back(group);
        return this->receive_return;
    }

    int writestate(uint16_t) override { return this->writestate_return; }

    uint16_t statecheck(uint16_t slave, uint16_t reqstate, int) override {
        auto it = this->statecheck_overrides.find(reqstate);
        uint16_t state = it != this->statecheck_overrides.end() ? it->second : reqstate;
        if (auto &pending = this->statecheck_pending_polls[reqstate]; pending > 0) {
            pending--;
            state = this->slave_state(slave);
        }
        if (slave > 0 && slave <= this->slave_infos.size())
            this->slave_infos[slave - 1].state = state;
        return state;
    }

    int SDOread(
        uint16_t,
        uint16_t index,
        uint8_t sub_index,
        int,
        int *psize,
        void *p,
        int
    ) override {
        const auto it = this->sdo_objects.find({index, sub_index});
        if (it == this->sdo_objects.end()) return 0;
        std::memcpy(p, &it->second, *psize);
        return 1;
    }

    int16_t siifind(uint16_t, uint16_t category) override {
        const auto it = this->sii_categories.find(category);
        return it == this->sii_categories.end() ? 0 : it->second;
    }

    uint8_t siigetbyte(uint16_t, uint16_t address) override {
        const auto it = this->sii_bytes.find(address);
        return it == this->sii_bytes.end() ? 0 : it->second;
    }

    void siistring(char *str, uint16_t, uint16_t) override { str[0] = '\0'; }

    int readODlist(uint16_t, ec_ODlistt *) override { return this->readODlist_return; }

    int readOEsingle(
        uint16_t item,
        uint8_t sub_index,
        ec_ODlistt *od_list,
        ec_OElistt *oe_list
    ) override {
        const auto it = this->oe_entries.find({od_list->Index[item], sub_index});
        if (it == this->oe_entries.end()) return 0;
        oe_list->Entries++;
        std::strncpy(oe_list->Name[sub_index], it->second.first.c_str(), EC_MAXNAME);
        oe_list->DataType[sub_index] = it->second.second;
        return 1;
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

    uint32_t group_Ibytes(uint8_t group) const override {
        return group == this->mapped_group ? this->ibytes : 0;
    }

    uint32_t group_Obytes(uint8_t group) const override {
        return group == this->mapped_group ? this->obytes : 0;
    }

    uint16_t group_outputsWKC(uint8_t group) const override {
        return group == this->mapped_group ? this->outputs_wkc : 0;
    }

    uint16_t group_inputsWKC(uint8_t group) const override {
        return group == this->mapped_group ? this->inputs_wkc : 0;
    }
};

}
