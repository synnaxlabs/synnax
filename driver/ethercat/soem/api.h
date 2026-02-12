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
#include <string>

extern "C" {
#include "soem/soem.h"
}

namespace driver::ethercat::soem {

/// @brief abstract interface for SOEM EtherCAT operations, abstracting both ecx_*
/// function calls and ecx_contextt struct field access for testability.
class API {
public:
    virtual ~API() = default;

    /// @brief initializes EtherCAT on the specified network interface.
    virtual int init(const char *ifname) = 0;

    /// @brief closes the EtherCAT connection.
    virtual void close() = 0;

    /// @brief discovers and initializes all slaves on the network.
    virtual int config_init() = 0;

    /// @brief maps I/O buffer for process data exchange for a slave group.
    virtual int config_map_group(void *iomap, uint8_t group) = 0;

    /// @brief sends process data to slaves.
    virtual int send_processdata() = 0;

    /// @brief receives process data from slaves with timeout in microseconds.
    virtual int receive_processdata(int timeout) = 0;

    /// @brief requests a state transition for the specified slave.
    virtual int writestate(uint16_t slave) = 0;

    /// @brief polls slave state change with timeout in microseconds.
    virtual uint16_t statecheck(uint16_t slave, uint16_t reqstate, int timeout) = 0;

    /// @brief reads from slave object dictionary via SDO.
    virtual int SDOread(
        uint16_t slave,
        uint16_t index,
        uint8_t subindex,
        int ca,
        int *psize,
        void *p,
        int timeout
    ) = 0;

    /// @brief finds SII category offset.
    virtual int16_t siifind(uint16_t slave, uint16_t cat) = 0;

    /// @brief reads single byte from slave EEPROM via SII.
    virtual uint8_t siigetbyte(uint16_t slave, uint16_t address) = 0;

    /// @brief reads string from slave EEPROM.
    virtual void siistring(char *str, uint16_t slave, uint16_t sn) = 0;

    /// @brief reads object dictionary list from slave.
    virtual int readODlist(uint16_t slave, ec_ODlistt *od_list) = 0;

    /// @brief reads single object entry details from slave.
    virtual int readOEsingle(
        uint16_t item,
        uint8_t sub_index,
        ec_ODlistt *od_list,
        ec_OElistt *oe_list
    ) = 0;

    /// @brief returns the number of slaves found.
    virtual int slave_count() const = 0;

    /// @brief returns the EtherCAT state of a slave.
    virtual uint16_t slave_state(uint16_t pos) const = 0;

    /// @brief sets the EtherCAT state of a slave.
    virtual void set_slave_state(uint16_t pos, uint16_t state) = 0;

    /// @brief returns the manufacturer ID from slave EEPROM.
    virtual uint32_t slave_eep_man(uint16_t pos) const = 0;

    /// @brief returns the product ID from slave EEPROM.
    virtual uint32_t slave_eep_id(uint16_t pos) const = 0;

    /// @brief returns the revision from slave EEPROM.
    virtual uint32_t slave_eep_rev(uint16_t pos) const = 0;

    /// @brief returns the serial number from slave EEPROM.
    virtual uint32_t slave_eep_ser(uint16_t pos) const = 0;

    /// @brief returns the slave name.
    virtual std::string slave_name(uint16_t pos) const = 0;

    /// @brief returns the input bits for a slave.
    virtual uint16_t slave_Ibits(uint16_t pos) const = 0;

    /// @brief returns the output bits for a slave.
    virtual uint16_t slave_Obits(uint16_t pos) const = 0;

    /// @brief returns the group assignment of a slave.
    virtual uint8_t slave_group(uint16_t pos) const = 0;

    /// @brief sets the group assignment of a slave.
    virtual void set_slave_group(uint16_t pos, uint8_t group) = 0;

    /// @brief returns the output data pointer for a slave.
    virtual uint8_t *slave_outputs(uint16_t pos) const = 0;

    /// @brief returns the input data pointer for a slave.
    virtual uint8_t *slave_inputs(uint16_t pos) const = 0;

    /// @brief returns the mailbox protocol flags for a slave.
    virtual uint16_t slave_mbx_proto(uint16_t pos) const = 0;

    /// @brief returns the AL status code for a slave.
    virtual uint16_t slave_ALstatuscode(uint16_t pos) const = 0;

    /// @brief returns the input bytes for a group.
    virtual uint32_t group_Ibytes(uint8_t grp) const = 0;

    /// @brief returns the output bytes for a group.
    virtual uint32_t group_Obytes(uint8_t grp) const = 0;

    /// @brief returns the expected output working counter for a group.
    virtual uint16_t group_outputsWKC(uint8_t grp) const = 0;

    /// @brief returns the expected input working counter for a group.
    virtual uint16_t group_inputsWKC(uint8_t grp) const = 0;
};

/// @brief production implementation of soem::API wrapping ecx_contextt.
class ProdAPI final : public API {
    ecx_contextt context;

public:
    ProdAPI() { std::memset(&this->context, 0, sizeof(this->context)); }

    int init(const char *ifname) override { return ecx_init(&this->context, ifname); }

    void close() override { ecx_close(&this->context); }

    int config_init() override { return ecx_config_init(&this->context); }

    int config_map_group(void *iomap, uint8_t group) override {
        return ecx_config_map_group(&this->context, iomap, group);
    }

    int send_processdata() override { return ecx_send_processdata(&this->context); }

    int receive_processdata(int timeout) override {
        return ecx_receive_processdata(&this->context, timeout);
    }

    int writestate(uint16_t slave) override {
        return ecx_writestate(&this->context, slave);
    }

    uint16_t statecheck(uint16_t slave, uint16_t reqstate, int timeout) override {
        return ecx_statecheck(&this->context, slave, reqstate, timeout);
    }

    int SDOread(
        uint16_t slave,
        uint16_t index,
        uint8_t subindex,
        int ca,
        int *psize,
        void *p,
        int timeout
    ) override {
        return ecx_SDOread(
            &this->context,
            slave,
            index,
            subindex,
            ca,
            psize,
            p,
            timeout
        );
    }

    int16_t siifind(uint16_t slave, uint16_t cat) override {
        return ecx_siifind(&this->context, slave, cat);
    }

    uint8_t siigetbyte(uint16_t slave, uint16_t address) override {
        return ecx_siigetbyte(&this->context, slave, address);
    }

    void siistring(char *str, uint16_t slave, uint16_t sn) override {
        ecx_siistring(&this->context, str, slave, sn);
    }

    int readODlist(uint16_t slave, ec_ODlistt *od_list) override {
        return ecx_readODlist(&this->context, slave, od_list);
    }

    int readOEsingle(
        uint16_t item,
        uint8_t sub_index,
        ec_ODlistt *od_list,
        ec_OElistt *oe_list
    ) override {
        return ecx_readOEsingle(&this->context, item, sub_index, od_list, oe_list);
    }

    int slave_count() const override { return this->context.slavecount; }

    uint16_t slave_state(uint16_t pos) const override {
        return this->context.slavelist[pos].state;
    }

    void set_slave_state(uint16_t pos, uint16_t state) override {
        this->context.slavelist[pos].state = state;
    }

    uint32_t slave_eep_man(uint16_t pos) const override {
        return this->context.slavelist[pos].eep_man;
    }

    uint32_t slave_eep_id(uint16_t pos) const override {
        return this->context.slavelist[pos].eep_id;
    }

    uint32_t slave_eep_rev(uint16_t pos) const override {
        return this->context.slavelist[pos].eep_rev;
    }

    uint32_t slave_eep_ser(uint16_t pos) const override {
        return this->context.slavelist[pos].eep_ser;
    }

    std::string slave_name(uint16_t pos) const override {
        return this->context.slavelist[pos].name;
    }

    uint16_t slave_Ibits(uint16_t pos) const override {
        return this->context.slavelist[pos].Ibits;
    }

    uint16_t slave_Obits(uint16_t pos) const override {
        return this->context.slavelist[pos].Obits;
    }

    uint8_t slave_group(uint16_t pos) const override {
        return this->context.slavelist[pos].group;
    }

    void set_slave_group(uint16_t pos, uint8_t group) override {
        this->context.slavelist[pos].group = group;
    }

    uint8_t *slave_outputs(uint16_t pos) const override {
        return this->context.slavelist[pos].outputs;
    }

    uint8_t *slave_inputs(uint16_t pos) const override {
        return this->context.slavelist[pos].inputs;
    }

    uint16_t slave_mbx_proto(uint16_t pos) const override {
        return this->context.slavelist[pos].mbx_proto;
    }

    uint16_t slave_ALstatuscode(uint16_t pos) const override {
        return this->context.slavelist[pos].ALstatuscode;
    }

    uint32_t group_Ibytes(uint8_t grp) const override {
        return this->context.grouplist[grp].Ibytes;
    }

    uint32_t group_Obytes(uint8_t grp) const override {
        return this->context.grouplist[grp].Obytes;
    }

    uint16_t group_outputsWKC(uint8_t grp) const override {
        return this->context.grouplist[grp].outputsWKC;
    }

    uint16_t group_inputsWKC(uint8_t grp) const override {
        return this->context.grouplist[grp].inputsWKC;
    }
};

}
