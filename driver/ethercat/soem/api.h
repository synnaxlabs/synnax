// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <array>
#include <cstdint>
#include <cstring>
#include <functional>
#include <optional>
#include <span>
#include <string>
#include <unordered_map>
#include <vector>

#include "x/cpp/errors/errors.h"

extern "C" {
#include "soem/soem.h"
}

namespace driver::ethercat::soem {

/// @brief the bytes of one AL status read in a frame: a 10-byte datagram header, the
/// 2-byte status, and a 2-byte working counter.
constexpr size_t STATE_READ_SIZE = 14;

/// @brief the most slaves whose AL status one frame of API::send_state_read() reads,
/// after the EtherCAT header and before the 4-byte frame check sequence.
constexpr size_t MAX_STATE_READS = (EC_MAXECATFRAME - ETH_HEADERSIZE - EC_ELENGTHSIZE -
                                    4) /
                                   STATE_READ_SIZE;

/// @brief the SOEM calls that the master makes, so that tests can run the master
/// without a bus.
class API {
public:
    virtual ~API() = default;

    /// @brief the SOEM context. The master reads and writes its slave table, group
    /// table, and DC time directly, and each call below updates it as its ecx_*
    /// function does.
    ecx_contextt context{};

    /// @brief checks that the process can open a raw EtherCAT socket.
    /// @param ifname the interface named in the returned error.
    /// @returns errors::PERMISSION_DENIED when the process lacks permission, or
    /// errors::INTERFACE_ERROR when the socket fails to open for another reason.
    virtual x::errors::Error check_socket(const char *ifname) = 0;

    /// @brief returns true when the DewesoftX "EthercatDAQ Filter" is bound to the
    /// adapter of ifname. The filter drops EtherCAT replies. Always false outside
    /// Windows.
    virtual bool dewesoft_filter_bound(const char *ifname) = 0;

    /// @brief returns the name Windows shows for the adapter of ifname, such as
    /// "Ethernet 2", or std::nullopt when the name cannot be read. Always std::nullopt
    /// outside Windows.
    virtual std::optional<std::string> adapter_name(const char *ifname) = 0;

    /// @brief returns true when the adapter of ifname is on USB. Always false on macOS.
    virtual bool on_usb(const char *ifname) = 0;

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

    /// @brief requests INIT for a slave and clears its sync managers and FMMUs, as
    /// config_init does for every slave. Returns 1 when the slave takes each write.
    virtual int reset_slave(uint16_t slave, int timeout) = 0;

    /// @brief takes a slave through INIT and PRE-OP back to SAFE-OP with its stored
    /// configuration, returning the state it reached.
    virtual int reconfig_slave(uint16_t slave, int timeout) = 0;

    /// @brief restores the address of a slave that lost power, returning a positive
    /// value when the same slave answered at its position.
    virtual int recover_slave(uint16_t slave, int timeout) = 0;

    /// @brief measures propagation delays and sets up the distributed clock of every
    /// slave that has one, returning true when at least one does.
    virtual bool configdc() = 0;

    /// @brief makes every slave latch its local time (0x0918) as one frame passes it,
    /// with a timeout in microseconds, returning the number of slaves that latched.
    virtual int latch_receive_times(int timeout) = 0;

    /// @brief sends a frame that carries only the reference clock time to the slaves of
    /// a group and waits up to timeout microseconds for it. Returns the working
    /// counter, or a value below 1 when the frame does not come back.
    virtual int sync_dc(uint8_t group, int timeout) = 0;

    /// @brief sends a frame that reads the AL status of each of slaves, at most
    /// MAX_STATE_READS of them, and returns the index of the frame. The frame passes
    /// each slave after the frames sent before it.
    virtual uint8_t send_state_read(std::span<const uint16_t> slaves) = 0;

    /// @brief waits up to timeout microseconds for the frame at idx that
    /// send_state_read() sent, and frees it. When it comes back, writes the AL status
    /// of each of its slaves into states, in order, giving 0 to a slave that did not
    /// answer, and returns true.
    virtual bool
    receive_state_read(uint8_t idx, std::span<uint16_t> states, int timeout) = 0;

    /// @brief writes data to a register of a slave with a timeout in microseconds,
    /// returning the working counter, which is 1 when the slave took the write.
    virtual int write_register(
        uint16_t slave,
        uint16_t address,
        std::vector<uint8_t> data,
        int timeout
    ) = 0;

    /// @brief reads a register of a slave into data with a timeout in microseconds,
    /// returning the working counter, which is 1 when the slave answered.
    virtual int read_register(
        uint16_t slave,
        uint16_t address,
        std::span<uint8_t> data,
        int timeout
    ) = 0;

    /// @brief makes SOEM call config in PRE-OP each time it maps the slave on the way
    /// to SAFE-OP, including reconfiguration after a fault. Passing nullptr removes it.
    virtual void
    set_PO2SOconfig(uint16_t slave, std::function<void(uint16_t)> config) = 0;

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

    /// @brief writes to a slave object dictionary via SDO, returning a positive working
    /// counter on success.
    virtual int SDOwrite(
        uint16_t slave,
        uint16_t index,
        uint8_t subindex,
        bool ca,
        int psize,
        const void *p,
        int timeout
    ) = 0;

    /// @brief moves the oldest entry of the SOEM error list into err, returning false
    /// when the list is empty.
    virtual bool poperror(ec_errort &err) = 0;

    /// @brief finds SII category offset.
    virtual int16_t siifind(uint16_t slave, uint16_t cat) = 0;

    /// @brief reads single byte from slave EEPROM via SII.
    virtual uint8_t siigetbyte(uint16_t slave, uint16_t address) = 0;

    /// @brief reads string from slave EEPROM.
    virtual void siistring(char *str, uint16_t slave, uint16_t sn) = 0;

    /// @brief gives control of the slave EEPROM back to the slave. siigetbyte takes
    /// control and keeps it.
    virtual void eeprom2pdi(uint16_t slave) = 0;

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
};

/// @brief production implementation of soem::API wrapping ecx_contextt.
class ProdAPI final : public API {
    std::unordered_map<uint16_t, std::function<void(uint16_t)>> po2so_configs;

    static int run_PO2SOconfig(ecx_contextt *context, const uint16 slave) {
        static_cast<ProdAPI *>(context->userdata)->po2so_configs.at(slave)(slave);
        return 1;
    }

public:
    x::errors::Error check_socket(const char *ifname) override;

    bool dewesoft_filter_bound(const char *ifname) override;

    std::optional<std::string> adapter_name(const char *ifname) override;

    bool on_usb(const char *ifname) override;

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

    int sync_dc(uint8_t group, int timeout) override {
        const auto ref = this->context.grouplist[group].DCnext;
        int64_t time = 0;
        const int wkc = ecx_FRMW(
            &this->context.port,
            this->context.slavelist[ref].configadr,
            ECT_REG_DCSYSTIME,
            sizeof(time),
            &time,
            timeout
        );
        if (wkc > 0) this->context.DCtime = etohll(time);
        return wkc;
    }

    uint8_t send_state_read(const std::span<const uint16_t> slaves) override {
        auto &port = this->context.port;
        const uint8_t idx = ecx_getindex(&port);
        uint16_t status = 0;
        for (size_t i = 0; i < slaves.size(); ++i) {
            const uint16_t adp = this->context.slavelist[slaves[i]].configadr;
            if (i == 0)
                ecx_setupdatagram(
                    &port,
                    &port.txbuf[idx],
                    EC_CMD_FPRD,
                    idx,
                    adp,
                    ECT_REG_ALSTAT,
                    sizeof(status),
                    &status
                );
            else
                ecx_adddatagram(
                    &port,
                    &port.txbuf[idx],
                    EC_CMD_FPRD,
                    idx,
                    i + 1 < slaves.size(),
                    adp,
                    ECT_REG_ALSTAT,
                    sizeof(status),
                    &status
                );
        }
        ecx_outframe_red(&port, idx);
        return idx;
    }

    bool receive_state_read(
        const uint8_t idx,
        const std::span<uint16_t> states,
        const int timeout
    ) override {
        auto &port = this->context.port;
        const bool received = ecx_waitinframe(&port, idx, timeout) > EC_NOFRAME;
        const uint8_t *frame = port.rxbuf[idx];
        for (size_t i = 0; received && i < states.size(); ++i) {
            // The receive buffer starts at the EtherCAT header.
            const size_t at = EC_HEADERSIZE + i * STATE_READ_SIZE;
            uint16_t status = 0;
            uint16_t wkc = 0;
            std::memcpy(&status, &frame[at], sizeof(status));
            std::memcpy(&wkc, &frame[at + sizeof(status)], sizeof(wkc));
            states[i] = etohs(wkc) == 1 ? etohs(status) : 0;
        }
        ecx_setbufstat(&port, idx, EC_BUF_EMPTY);
        return received;
    }

    int latch_receive_times(const int timeout) override {
        int32_t zero = 0;
        return ecx_BWR(
            &this->context.port,
            0x0000,
            ECT_REG_DCTIME0,
            sizeof(zero),
            &zero,
            timeout
        );
    }

    int write_register(
        uint16_t slave,
        uint16_t address,
        std::vector<uint8_t> data,
        int timeout
    ) override {
        return ecx_FPWR(
            &this->context.port,
            this->context.slavelist[slave].configadr,
            address,
            static_cast<uint16_t>(data.size()),
            data.data(),
            timeout
        );
    }

    int read_register(
        uint16_t slave,
        uint16_t address,
        std::span<uint8_t> data,
        int timeout
    ) override {
        return ecx_FPRD(
            &this->context.port,
            this->context.slavelist[slave].configadr,
            address,
            static_cast<uint16_t>(data.size()),
            data.data(),
            timeout
        );
    }

    int reset_slave(uint16_t slave, int timeout) override {
        const uint16_t address = this->context.slavelist[slave].configadr;
        std::array<uint8_t, 4 * sizeof(ec_smt)> sync_managers{};
        std::array<uint8_t, 3 * sizeof(ec_fmmut)> fmmus{};
        return ecx_FPWRw(
                   &this->context.port,
                   address,
                   ECT_REG_ALCTL,
                   htoes(EC_STATE_INIT),
                   timeout
               ) == 1 &&
               ecx_FPWR(
                   &this->context.port,
                   address,
                   ECT_REG_SM0,
                   sync_managers.size(),
                   sync_managers.data(),
                   timeout
               ) == 1 &&
               ecx_FPWR(
                   &this->context.port,
                   address,
                   ECT_REG_FMMU0,
                   fmmus.size(),
                   fmmus.data(),
                   timeout
               ) == 1;
    }

    int reconfig_slave(uint16_t slave, int timeout) override {
        return ecx_reconfig_slave(&this->context, slave, timeout);
    }

    int recover_slave(uint16_t slave, int timeout) override {
        return ecx_recover_slave(&this->context, slave, timeout);
    }

    void
    set_PO2SOconfig(uint16_t slave, std::function<void(uint16_t)> config) override {
        this->context.userdata = this;
        if (config == nullptr) {
            this->po2so_configs.erase(slave);
            this->context.slavelist[slave].PO2SOconfig = nullptr;
            return;
        }
        this->po2so_configs[slave] = std::move(config);
        this->context.slavelist[slave].PO2SOconfig = &ProdAPI::run_PO2SOconfig;
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

    int SDOwrite(
        uint16_t slave,
        uint16_t index,
        uint8_t subindex,
        bool ca,
        int psize,
        const void *p,
        int timeout
    ) override {
        return ecx_SDOwrite(
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

    bool poperror(ec_errort &err) override {
        return ecx_poperror(&this->context, &err) != 0;
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

    void eeprom2pdi(uint16_t slave) override { ecx_eeprom2pdi(&this->context, slave); }

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
};

}
