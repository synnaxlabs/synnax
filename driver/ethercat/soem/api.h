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

#include "x/cpp/errors/errors.h"

extern "C" {
#include "soem/soem.h"
}

namespace driver::ethercat::soem {

/// @brief abstract interface for SOEM EtherCAT operations, abstracting both ecx_*
/// function calls and ecx_contextt struct field access for testability.
class API {
public:
    virtual ~API() = default;

    /// @brief checks that the process can open a raw EtherCAT socket.
    /// @param ifname the interface named in the returned error.
    /// @returns errors::PERMISSION_DENIED when the process lacks permission, or
    /// errors::INTERFACE_ERROR when the socket fails to open for another reason.
    virtual x::errors::Error check_socket(const char *ifname) = 0;

    /// @brief initializes EtherCAT on the specified network interface.
    virtual int init(const char *ifname) = 0;

    /// @brief closes the EtherCAT connection.
    virtual void close() = 0;

    /// @brief discovers and initializes all slaves on the network.
    virtual int config_init() = 0;

    /// @brief maps I/O buffer for process data exchange for a slave group.
    virtual int config_map_group(void *iomap, uint8_t group) = 0;

    /// @brief sends the process data of a slave group.
    virtual int send_processdata(uint8_t group) = 0;

    /// @brief receives the process data of a slave group with a timeout in
    /// microseconds.
    virtual int receive_processdata(uint8_t group, int timeout) = 0;

    /// @brief requests a state transition for the specified slave.
    virtual int writestate(uint16_t slave) = 0;

    /// @brief polls slave state change with timeout in microseconds.
    virtual uint16_t statecheck(uint16_t slave, uint16_t reqstate, int timeout) = 0;

    /// @brief reads the AL state and status code of every slave into the slave list.
    virtual int readstate() = 0;

    /// @brief returns the number of slaves that answer a broadcast read, or a value
    /// below 1 when none answers.
    virtual int count_bus_slaves() = 0;

    /// @brief takes a slave through INIT and PRE-OP back to SAFE-OP with its stored
    /// configuration, returning the state it reached.
    virtual int reconfig_slave(uint16_t slave, int timeout) = 0;

    /// @brief restores the address of a slave that lost power, returning a positive
    /// value when the same slave answered at its position.
    virtual int recover_slave(uint16_t slave, int timeout) = 0;

    /// @brief measures propagation delays and sets up the distributed clock of every
    /// slave that has one, returning true when at least one does.
    virtual bool configdc() = 0;

    /// @brief returns the reference clock system time from the latest process data
    /// frame, in nanoseconds since 2000-01-01.
    [[nodiscard]] virtual int64_t dc_time() const = 0;

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

    /// @brief reads the description, including the name, of object
    /// od_list->Index[item] into od_list.
    virtual int readODdescription(uint16_t item, ec_ODlistt *od_list) = 0;

    /// @brief reads single object entry details from slave.
    virtual int readOEsingle(
        uint16_t item,
        uint8_t sub_index,
        ec_ODlistt *od_list,
        ec_OElistt *oe_list
    ) = 0;

    /// @brief returns the number of slaves found.
    [[nodiscard]] virtual int slave_count() const = 0;

    /// @brief returns the EtherCAT state of a slave.
    [[nodiscard]] virtual uint16_t slave_state(uint16_t pos) const = 0;

    /// @brief sets the EtherCAT state of a slave.
    virtual void set_slave_state(uint16_t pos, uint16_t state) = 0;

    /// @brief returns the manufacturer ID from slave EEPROM.
    [[nodiscard]] virtual uint32_t slave_eep_man(uint16_t pos) const = 0;

    /// @brief returns the product ID from slave EEPROM.
    [[nodiscard]] virtual uint32_t slave_eep_id(uint16_t pos) const = 0;

    /// @brief returns the revision from slave EEPROM.
    [[nodiscard]] virtual uint32_t slave_eep_rev(uint16_t pos) const = 0;

    /// @brief returns the serial number from slave EEPROM.
    [[nodiscard]] virtual uint32_t slave_eep_ser(uint16_t pos) const = 0;

    /// @brief returns the slave name.
    [[nodiscard]] virtual std::string slave_name(uint16_t pos) const = 0;

    /// @brief returns the input bits for a slave.
    [[nodiscard]] virtual uint16_t slave_Ibits(uint16_t pos) const = 0;

    /// @brief returns the output bits for a slave.
    [[nodiscard]] virtual uint16_t slave_Obits(uint16_t pos) const = 0;

    /// @brief returns the group assignment of a slave.
    [[nodiscard]] virtual uint8_t slave_group(uint16_t pos) const = 0;

    /// @brief sets the group assignment of a slave.
    virtual void set_slave_group(uint16_t pos, uint8_t group) = 0;

    /// @brief returns the output data pointer for a slave.
    [[nodiscard]] virtual uint8_t *slave_outputs(uint16_t pos) const = 0;

    /// @brief returns the input data pointer for a slave.
    [[nodiscard]] virtual uint8_t *slave_inputs(uint16_t pos) const = 0;

    /// @brief returns the mailbox protocol flags for a slave.
    [[nodiscard]] virtual uint16_t slave_mbx_proto(uint16_t pos) const = 0;

    /// @brief returns the AL status code for a slave.
    [[nodiscard]] virtual uint16_t slave_ALstatuscode(uint16_t pos) const = 0;

    /// @brief returns whether a slave is marked as no longer responding.
    [[nodiscard]] virtual bool slave_lost(uint16_t pos) const = 0;

    /// @brief marks whether a slave no longer responds.
    virtual void set_slave_lost(uint16_t pos, bool lost) = 0;

    /// @brief returns the input bytes for a group.
    [[nodiscard]] virtual uint32_t group_Ibytes(uint8_t grp) const = 0;

    /// @brief returns the output bytes for a group.
    [[nodiscard]] virtual uint32_t group_Obytes(uint8_t grp) const = 0;

    /// @brief returns the expected output working counter for a group.
    [[nodiscard]] virtual uint16_t group_outputsWKC(uint8_t grp) const = 0;

    /// @brief returns the expected input working counter for a group.
    [[nodiscard]] virtual uint16_t group_inputsWKC(uint8_t grp) const = 0;

    /// @brief returns true when the group's process data frames carry the reference
    /// clock time.
    [[nodiscard]] virtual bool group_hasdc(uint8_t grp) const = 0;
};

/// @brief production implementation of soem::API wrapping ecx_contextt.
class ProdAPI final : public API {
    ecx_contextt context{};

public:
    ProdAPI() { std::memset(&this->context, 0, sizeof(this->context)); }

    x::errors::Error check_socket(const char *ifname) override;

    int init(const char *ifname) override { return ecx_init(&this->context, ifname); }

    void close() override { ecx_close(&this->context); }

    int config_init() override { return ecx_config_init(&this->context); }

    int config_map_group(void *iomap, uint8_t group) override {
        return ecx_config_map_group(&this->context, iomap, group);
    }

    int send_processdata(uint8_t group) override {
        return ecx_send_processdata_group(&this->context, group);
    }

    int receive_processdata(uint8_t group, int timeout) override {
        return ecx_receive_processdata_group(&this->context, group, timeout);
    }

    int writestate(uint16_t slave) override {
        return ecx_writestate(&this->context, slave);
    }

    uint16_t statecheck(uint16_t slave, uint16_t reqstate, int timeout) override {
        return ecx_statecheck(&this->context, slave, reqstate, timeout);
    }

    int readstate() override { return ecx_readstate(&this->context); }

    int count_bus_slaves() override {
        uint16_t type = 0;
        return ecx_BRD(
            &this->context.port,
            0x0000,
            ECT_REG_TYPE,
            sizeof(type),
            &type,
            EC_TIMEOUTSAFE
        );
    }

    bool configdc() override { return ecx_configdc(&this->context); }

    [[nodiscard]] int64_t dc_time() const override { return this->context.DCtime; }

    int reconfig_slave(uint16_t slave, int timeout) override {
        return ecx_reconfig_slave(&this->context, slave, timeout);
    }

    int recover_slave(uint16_t slave, int timeout) override {
        return ecx_recover_slave(&this->context, slave, timeout);
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

    int readODdescription(uint16_t item, ec_ODlistt *od_list) override {
        return ecx_readODdescription(&this->context, item, od_list);
    }

    int readOEsingle(
        uint16_t item,
        uint8_t sub_index,
        ec_ODlistt *od_list,
        ec_OElistt *oe_list
    ) override {
        return ecx_readOEsingle(&this->context, item, sub_index, od_list, oe_list);
    }

    [[nodiscard]] int slave_count() const override { return this->context.slavecount; }

    [[nodiscard]] uint16_t slave_state(uint16_t pos) const override {
        return this->context.slavelist[pos].state;
    }

    void set_slave_state(uint16_t pos, uint16_t state) override {
        this->context.slavelist[pos].state = state;
    }

    [[nodiscard]] uint32_t slave_eep_man(uint16_t pos) const override {
        return this->context.slavelist[pos].eep_man;
    }

    [[nodiscard]] uint32_t slave_eep_id(uint16_t pos) const override {
        return this->context.slavelist[pos].eep_id;
    }

    [[nodiscard]] uint32_t slave_eep_rev(uint16_t pos) const override {
        return this->context.slavelist[pos].eep_rev;
    }

    [[nodiscard]] uint32_t slave_eep_ser(uint16_t pos) const override {
        return this->context.slavelist[pos].eep_ser;
    }

    [[nodiscard]] std::string slave_name(uint16_t pos) const override {
        return this->context.slavelist[pos].name;
    }

    [[nodiscard]] uint16_t slave_Ibits(uint16_t pos) const override {
        return this->context.slavelist[pos].Ibits;
    }

    [[nodiscard]] uint16_t slave_Obits(uint16_t pos) const override {
        return this->context.slavelist[pos].Obits;
    }

    [[nodiscard]] uint8_t slave_group(uint16_t pos) const override {
        return this->context.slavelist[pos].group;
    }

    void set_slave_group(uint16_t pos, uint8_t group) override {
        this->context.slavelist[pos].group = group;
    }

    [[nodiscard]] uint8_t *slave_outputs(uint16_t pos) const override {
        return this->context.slavelist[pos].outputs;
    }

    [[nodiscard]] uint8_t *slave_inputs(uint16_t pos) const override {
        return this->context.slavelist[pos].inputs;
    }

    [[nodiscard]] uint16_t slave_mbx_proto(uint16_t pos) const override {
        return this->context.slavelist[pos].mbx_proto;
    }

    [[nodiscard]] uint16_t slave_ALstatuscode(uint16_t pos) const override {
        return this->context.slavelist[pos].ALstatuscode;
    }

    [[nodiscard]] bool slave_lost(uint16_t pos) const override {
        return this->context.slavelist[pos].islost;
    }

    void set_slave_lost(uint16_t pos, bool lost) override {
        this->context.slavelist[pos].islost = lost;
    }

    [[nodiscard]] uint32_t group_Ibytes(uint8_t grp) const override {
        return this->context.grouplist[grp].Ibytes;
    }

    [[nodiscard]] uint32_t group_Obytes(uint8_t grp) const override {
        return this->context.grouplist[grp].Obytes;
    }

    [[nodiscard]] uint16_t group_outputsWKC(uint8_t grp) const override {
        return this->context.grouplist[grp].outputsWKC;
    }

    [[nodiscard]] uint16_t group_inputsWKC(uint8_t grp) const override {
        return this->context.grouplist[grp].inputsWKC;
    }

    [[nodiscard]] bool group_hasdc(uint8_t grp) const override {
        return this->context.grouplist[grp].hasdc;
    }
};

}
