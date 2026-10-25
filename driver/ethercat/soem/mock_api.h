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
#include <map>
#include <optional>
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
    bool lost = false;
    /// @brief the state readstate() reports, or nullopt to report the current state.
    std::optional<uint16_t> bus_state;
    /// @brief the state last requested and not yet reached.
    std::optional<uint16_t> requested;
};

/// @brief mock implementation of soem::API for unit testing.
class MockAPI final : public API {
public:
    std::vector<MockSlaveInfo> slave_infos;
    x::errors::Error check_socket_err = x::errors::NIL;
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
    /// @brief SII strings returned by siistring, keyed by string index.
    std::map<uint16_t, std::string> sii_strings;
    int readODlist_return = 0;
    bool close_called = false;
    bool init_called = false;
    int send_count = 0;
    int receive_count = 0;
    std::vector<std::pair<uint16_t, uint16_t>> state_change_requests;
    std::vector<uint8_t> processdata_groups;
    std::optional<uint8_t> mapped_group;
    /// @brief CoE objects returned by SDOread, keyed by index and sub-index.
    std::map<std::pair<uint16_t, uint8_t>, uint32_t> sdo_objects;
    /// @brief entry names and CoE data types returned by readOEsingle, keyed by index
    /// and sub-index.
    std::map<std::pair<uint16_t, uint8_t>, std::pair<std::string, uint16_t>> oe_entries;
    /// @brief object names returned by readODdescription, keyed by index.
    std::map<uint16_t, std::string> od_names;
    int readstate_count = 0;
    /// @brief the state reconfig_slave returns.
    int reconfig_return = EC_STATE_SAFE_OP;
    std::vector<uint16_t> reconfigured_slaves;
    /// @brief the value recover_slave returns.
    int recover_return = 1;
    std::vector<uint16_t> recovered_slaves;
    /// @brief slaves passed to writestate, in call order.
    std::vector<uint16_t> written_slaves;
    /// @brief the value count_bus_slaves returns, or nullopt for the slave count.
    std::optional<int> bus_slaves;
    /// @brief whether the mapped group carries the reference clock time.
    bool hasdc = false;
    int configdc_count = 0;
    /// @brief the value dc_time returns.
    int64_t dc_time_value = 0;

private:
    uint8_t *iomap_ptr = nullptr;

public:
    x::errors::Error check_socket(const char *) override {
        return this->check_socket_err;
    }

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

    int writestate(uint16_t slave) override {
        this->written_slaves.push_back(slave);
        return this->writestate_return;
    }

    int readstate() override {
        this->readstate_count++;
        uint16_t lowest = 0xFF;
        for (size_t i = 0; i < this->slave_infos.size(); ++i) {
            auto &info = this->slave_infos[i];
            if (info.bus_state)
                info.state = *info.bus_state;
            else if (info.requested) {
                const uint16_t target = *info.requested & 0x0F;
                info.state = this->reach(static_cast<uint16_t>(i + 1), target);
                if (info.state == target) info.requested.reset();
            }
            lowest = std::min<uint16_t>(lowest, info.state & 0x0F);
        }
        return lowest;
    }

    int count_bus_slaves() override {
        return this->bus_slaves.value_or(static_cast<int>(this->slave_infos.size()));
    }

    bool configdc() override {
        this->configdc_count++;
        return this->hasdc;
    }

    [[nodiscard]] int64_t dc_time() const override { return this->dc_time_value; }

    int reconfig_slave(uint16_t slave, int) override {
        this->reconfigured_slaves.push_back(slave);
        return this->reconfig_return;
    }

    int recover_slave(uint16_t slave, int) override {
        this->recovered_slaves.push_back(slave);
        return this->recover_return;
    }

    uint16_t statecheck(uint16_t slave, uint16_t reqstate, int) override {
        const uint16_t state = this->reach(slave, reqstate);
        if (slave > 0 && slave <= this->slave_infos.size()) {
            auto &info = this->slave_infos[slave - 1];
            info.state = state;
            if (state == reqstate) info.requested.reset();
        }
        return state;
    }

    /// @brief returns the state a slave reports on one poll while it moves to reqstate.
    uint16_t reach(const uint16_t slave, const uint16_t reqstate) {
        const auto it = this->statecheck_overrides.find(reqstate);
        if (auto &pending = this->statecheck_pending_polls[reqstate]; pending > 0) {
            pending--;
            return this->slave_state(slave);
        }
        return it != this->statecheck_overrides.end() ? it->second : reqstate;
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

    void siistring(char *str, uint16_t, uint16_t sn) override {
        const auto it = this->sii_strings.find(sn);
        const std::string value = it == this->sii_strings.end() ? "" : it->second;
        std::strncpy(str, value.c_str(), EC_MAXNAME);
        str[EC_MAXNAME] = '\0';
    }

    int readODlist(uint16_t, ec_ODlistt *) override { return this->readODlist_return; }

    int readODdescription(uint16_t item, ec_ODlistt *od_list) override {
        const auto it = this->od_names.find(od_list->Index[item]);
        if (it == this->od_names.end()) return 0;
        std::strncpy(od_list->Name[item], it->second.c_str(), EC_MAXNAME);
        od_list->Name[item][EC_MAXNAME] = '\0';
        return 1;
    }

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

    [[nodiscard]] int slave_count() const override {
        return static_cast<int>(this->slave_infos.size());
    }

    [[nodiscard]] uint16_t slave_state(uint16_t pos) const override {
        if (pos == 0 || pos > this->slave_infos.size()) return 0;
        return this->slave_infos[pos - 1].state;
    }

    void set_slave_state(uint16_t pos, uint16_t state) override {
        if (pos == 0 || pos > this->slave_infos.size()) return;
        this->state_change_requests.emplace_back(pos, state);
        this->slave_infos[pos - 1].requested = state;
    }

    [[nodiscard]] uint32_t slave_eep_man(uint16_t pos) const override {
        if (pos == 0 || pos > this->slave_infos.size()) return 0;
        return this->slave_infos[pos - 1].eep_man;
    }

    [[nodiscard]] uint32_t slave_eep_id(uint16_t pos) const override {
        if (pos == 0 || pos > this->slave_infos.size()) return 0;
        return this->slave_infos[pos - 1].eep_id;
    }

    [[nodiscard]] uint32_t slave_eep_rev(uint16_t pos) const override {
        if (pos == 0 || pos > this->slave_infos.size()) return 0;
        return this->slave_infos[pos - 1].eep_rev;
    }

    [[nodiscard]] uint32_t slave_eep_ser(uint16_t pos) const override {
        if (pos == 0 || pos > this->slave_infos.size()) return 0;
        return this->slave_infos[pos - 1].eep_ser;
    }

    [[nodiscard]] std::string slave_name(uint16_t pos) const override {
        if (pos == 0 || pos > this->slave_infos.size()) return "";
        return this->slave_infos[pos - 1].name;
    }

    [[nodiscard]] uint16_t slave_Ibits(uint16_t pos) const override {
        if (pos == 0 || pos > this->slave_infos.size()) return 0;
        return this->slave_infos[pos - 1].Ibits;
    }

    [[nodiscard]] uint16_t slave_Obits(uint16_t pos) const override {
        if (pos == 0 || pos > this->slave_infos.size()) return 0;
        return this->slave_infos[pos - 1].Obits;
    }

    [[nodiscard]] uint8_t slave_group(uint16_t pos) const override {
        if (pos == 0 || pos > this->slave_infos.size()) return 0;
        return this->slave_infos[pos - 1].group;
    }

    void set_slave_group(uint16_t pos, uint8_t group) override {
        if (pos == 0 || pos > this->slave_infos.size()) return;
        this->slave_infos[pos - 1].group = group;
    }

    [[nodiscard]] uint8_t *slave_outputs(uint16_t pos) const override {
        if (pos == 0 || pos > this->slave_infos.size()) return nullptr;
        if (this->iomap_ptr == nullptr) return nullptr;
        return this->iomap_ptr + this->slave_infos[pos - 1].output_offset;
    }

    [[nodiscard]] uint8_t *slave_inputs(uint16_t pos) const override {
        if (pos == 0 || pos > this->slave_infos.size()) return nullptr;
        if (this->iomap_ptr == nullptr) return nullptr;
        return this->iomap_ptr + this->slave_infos[pos - 1].input_offset;
    }

    [[nodiscard]] uint16_t slave_mbx_proto(uint16_t pos) const override {
        if (pos == 0 || pos > this->slave_infos.size()) return 0;
        return this->slave_infos[pos - 1].mbx_proto;
    }

    [[nodiscard]] uint16_t slave_ALstatuscode(uint16_t pos) const override {
        if (pos == 0 || pos > this->slave_infos.size()) return 0;
        return this->slave_infos[pos - 1].ALstatuscode;
    }

    [[nodiscard]] bool slave_lost(uint16_t pos) const override {
        if (pos == 0 || pos > this->slave_infos.size()) return false;
        return this->slave_infos[pos - 1].lost;
    }

    void set_slave_lost(uint16_t pos, bool lost) override {
        if (pos == 0 || pos > this->slave_infos.size()) return;
        this->slave_infos[pos - 1].lost = lost;
    }

    [[nodiscard]] uint32_t group_Ibytes(uint8_t group) const override {
        return group == this->mapped_group ? this->ibytes : 0;
    }

    [[nodiscard]] uint32_t group_Obytes(uint8_t group) const override {
        return group == this->mapped_group ? this->obytes : 0;
    }

    [[nodiscard]] uint16_t group_outputsWKC(uint8_t group) const override {
        return group == this->mapped_group ? this->outputs_wkc : 0;
    }

    [[nodiscard]] uint16_t group_inputsWKC(uint8_t group) const override {
        return group == this->mapped_group ? this->inputs_wkc : 0;
    }

    [[nodiscard]] bool group_hasdc(uint8_t group) const override {
        return group == this->mapped_group && this->hasdc;
    }
};

}
