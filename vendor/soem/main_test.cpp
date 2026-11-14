// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cstddef>
#include <cstdint>
#include <cstring>
#include <deque>
#include <functional>
#include <map>
#include <memory>
#include <vector>

#include "gtest/gtest.h"
#include "soem/soem.h"

namespace {
/// The register addresses of the mailboxes of each slave.
constexpr uint16_t MAILBOX_OUT = 0x1000;
constexpr uint16_t MAILBOX_IN = 0x1100;
constexpr uint16_t MAILBOX_LENGTH = 128;

using Bytes = std::vector<uint8_t>;

/// @brief a slave on the Bus.
struct Slave {
    /// @brief the AL status that the slave reports.
    ec_alstatust al{};
    /// @brief the 32-bit registers that the slave reports, by address.
    std::map<uint16_t, uint32_t> registers;
    /// @brief returns the mailbox replies of the slave to a request.
    std::function<std::vector<Bytes>()> respond;
    /// @brief the mailbox replies that the master has not read.
    std::deque<Bytes> replies;

    void read(const uint16_t address, uint8_t *data, const size_t length) {
        std::memset(data, 0, length);
        if (address == ECT_REG_ALSTAT) {
            std::memcpy(data, &this->al, std::min(length, sizeof(this->al)));
        } else if (address == ECT_REG_SM1STAT) {
            data[0] = this->replies.empty() ? 0x00 : 0x08;
        } else if (address == MAILBOX_IN && !this->replies.empty()) {
            const auto &reply = this->replies.front();
            std::memcpy(data, reply.data(), std::min(length, reply.size()));
            this->replies.pop_front();
        } else if (
            const auto it = this->registers.find(address); it != this->registers.end()
        ) {
            std::memcpy(data, &it->second, std::min(length, sizeof(uint32_t)));
        }
    }

    void write(const uint16_t address) {
        if (address != MAILBOX_OUT || !this->respond) return;
        for (auto &reply: this->respond())
            this->replies.push_back(std::move(reply));
    }
};

/// @brief a bus that answers the register reads and mailbox requests of SOEM, keyed by
/// configured address.
struct Bus {
    /// @brief the slaves on the bus.
    std::map<uint16_t, Slave> slaves;
    /// @brief true when every frame is lost.
    bool frames_lost = false;

    /// @brief answers the frame in buf, returning the work counter of its last
    /// datagram.
    int answer(uint8_t *buf) {
        size_t pos = sizeof(uint16_t);
        while (true) {
            const uint8_t cmd = buf[pos];
            uint16_t adp = 0;
            uint16_t ado = 0;
            uint16_t dlength = 0;
            std::memcpy(&adp, &buf[pos + 2], sizeof(adp));
            std::memcpy(&ado, &buf[pos + 4], sizeof(ado));
            std::memcpy(&dlength, &buf[pos + 6], sizeof(dlength));
            const size_t length = dlength & 0x07FF;
            uint8_t *data = &buf[pos + 10];
            int wkc = 0;
            const auto it = this->slaves.find(adp);
            if (cmd == EC_CMD_BRD) {
                uint16_t state = 0;
                for (const auto &[_, slave]: this->slaves)
                    state |= slave.al.alstatus;
                std::memcpy(data, &state, std::min(length, sizeof(state)));
                wkc = static_cast<int>(this->slaves.size());
            } else if (cmd == EC_CMD_FPRD && it != this->slaves.end()) {
                it->second.read(ado, data, length);
                wkc = 1;
            } else if (cmd == EC_CMD_FPWR && it != this->slaves.end()) {
                it->second.write(ado);
                wkc = 1;
            }
            const auto counter = static_cast<uint16_t>(wkc);
            std::memcpy(&data[length], &counter, sizeof(counter));
            if ((dlength & EC_DATAGRAMFOLLOWS) == 0) return wkc;
            pos += 10 + length + EC_WKCSIZE;
        }
    }
};

/// @brief returns the expedited SDO upload reply that holds the 16-bit value of
/// index:sub_index.
Bytes upload_reply(
    const uint16_t index,
    const uint8_t sub_index,
    const uint16_t value
) {
    return {
        10,
        0,
        0,
        0,
        0,
        ECT_MBXT_COE,
        0x00,
        ECT_COES_SDORES << 4,
        0x4B,
        static_cast<uint8_t>(index & 0xFF),
        static_cast<uint8_t>(index >> 8),
        sub_index,
        static_cast<uint8_t>(value & 0xFF),
        static_cast<uint8_t>(value >> 8),
        0,
        0,
    };
}

Bus &bus_of(ecx_portt *port) {
    auto *context = reinterpret_cast<ecx_contextt *>(
        reinterpret_cast<char *>(port) - offsetof(ecx_contextt, port)
    );
    return *static_cast<Bus *>(context->userdata);
}
}

/// The NIC driver of SOEM, over the Bus in the userdata of the context.
extern "C" {
const uint16 priMAC[3] = EC_PRIMARY_MAC_ARRAY;
const uint16 secMAC[3] = EC_SECONDARY_MAC_ARRAY;

void ec_setupheader(void *) {}

int ecx_setupnic(ecx_portt *, const char *, int) {
    return 1;
}

int ecx_closenic(ecx_portt *) {
    return 0;
}

void ecx_setbufstat(ecx_portt *port, const uint8 idx, const int bufstat) {
    port->rxbufstat[idx] = bufstat;
}

uint8 ecx_getindex(ecx_portt *port) {
    for (uint8 idx = 0; idx < EC_MAXBUF; ++idx)
        if (port->rxbufstat[idx] == EC_BUF_EMPTY) {
            port->rxbufstat[idx] = EC_BUF_ALLOC;
            return idx;
        }
    return 0;
}

int ecx_outframe(ecx_portt *, uint8, int) {
    return 0;
}

int ecx_outframe_red(ecx_portt *, uint8) {
    return 0;
}

int ecx_waitinframe(ecx_portt *, uint8, int) {
    return EC_NOFRAME;
}

int ecx_srconfirm(ecx_portt *port, const uint8 idx, int) {
    auto &bus = bus_of(port);
    if (bus.frames_lost) return EC_NOFRAME;
    const auto length = port->txbuflength[idx] - ETH_HEADERSIZE;
    std::memcpy(port->rxbuf[idx], &port->txbuf[idx][ETH_HEADERSIZE], length);
    return bus.answer(port->rxbuf[idx]);
}
}

namespace {
constexpr uint16_t NONE = EC_STATE_NONE;
constexpr uint16_t SAFE_OP = EC_STATE_SAFE_OP;
constexpr uint16_t SAFE_OP_ERROR = EC_STATE_SAFE_OP + EC_STATE_ERROR;
constexpr uint16_t OP = EC_STATE_OPERATIONAL;

class ReadStateTest : public testing::Test {
protected:
    Bus bus;
    std::unique_ptr<ecx_contextt> context = std::make_unique<ecx_contextt>();

    void SetUp() override {
        this->context->userdata = &this->bus;
        this->context->slavecount = 2;
        for (uint16_t slave = 1; slave <= 2; ++slave) {
            this->context->slavelist[slave].configadr = 0x1000 + slave;
            this->context->slavelist[slave].state = OP;
        }
    }
};

/// @brief it should keep the state of each slave when the state read frame is lost.
TEST_F(ReadStateTest, KeepsEachStateWhenTheFrameIsLost) {
    this->bus.frames_lost = true;
    EXPECT_EQ(ecx_readstate(this->context.get()), OP);
    EXPECT_EQ(this->context->slavelist[1].state, OP);
    EXPECT_EQ(this->context->slavelist[2].state, OP);
}

/// @brief it should read the state and AL status code of each slave that answers.
TEST_F(ReadStateTest, ReadsTheStateOfEachSlave) {
    this->bus.slaves[0x1001].al = {.alstatus = OP};
    this->bus.slaves[0x1002].al = {.alstatus = SAFE_OP_ERROR, .alstatuscode = 0x001B};
    EXPECT_EQ(ecx_readstate(this->context.get()), SAFE_OP);
    EXPECT_EQ(this->context->slavelist[1].state, OP);
    EXPECT_EQ(this->context->slavelist[2].state, SAFE_OP_ERROR);
    EXPECT_EQ(this->context->slavelist[2].ALstatuscode, 0x001B);
}

/// @brief it should read a slave that does not answer in a frame that comes back as in
/// no state.
TEST_F(ReadStateTest, ReadsNoStateForASlaveThatDoesNotAnswer) {
    this->bus.slaves[0x1001].al = {.alstatus = OP};
    EXPECT_EQ(ecx_readstate(this->context.get()), NONE);
    EXPECT_EQ(this->context->slavelist[1].state, OP);
    EXPECT_EQ(this->context->slavelist[2].state, NONE);
}

class SOEMTest : public testing::Test {
protected:
    Bus bus;
    std::unique_ptr<ecx_contextt> context = std::make_unique<ecx_contextt>();
    Slave &slave = this->bus.slaves[0x1001];

    void SetUp() override {
        this->context->userdata = &this->bus;
        this->context->slavecount = 1;
        auto &rec = this->context->slavelist[1];
        rec.configadr = 0x1001;
        rec.state = EC_STATE_PRE_OP;
        rec.mbx_wo = MAILBOX_OUT;
        rec.mbx_ro = MAILBOX_IN;
        rec.mbx_l = MAILBOX_LENGTH;
        rec.mbx_rl = MAILBOX_LENGTH;
        ecx_initmbxpool(this->context.get());
    }

    /// @brief returns the next error in the SOEM error list.
    ec_errort pop_error() {
        ec_errort err{};
        EXPECT_TRUE(ecx_poperror(this->context.get(), &err));
        return err;
    }
};

/// @brief it should read the value in the SDO reply for the sub-index it asks for.
TEST_F(SOEMTest, SDOReadReturnsTheValueOfTheSubIndex) {
    this->slave.respond = [] {
        return std::vector<Bytes>{upload_reply(0x1C12, 2, 0x1601)};
    };
    uint16_t value = 0;
    int size = sizeof(value);
    EXPECT_GT(
        ecx_SDOread(this->context.get(), 1, 0x1C12, 2, FALSE, &size, &value, 100'000),
        0
    );
    EXPECT_EQ(value, 0x1601);
}

/// @brief it should fail an SDO read with an unexpected frame error when the reply is
/// for another sub-index of the object.
TEST_F(SOEMTest, SDOReadFailsOnAReplyForAnotherSubIndex) {
    this->slave.respond = [] {
        return std::vector<Bytes>{upload_reply(0x1C12, 1, 0x1600)};
    };
    uint16_t value = 0;
    int size = sizeof(value);
    EXPECT_LE(
        ecx_SDOread(this->context.get(), 1, 0x1C12, 2, FALSE, &size, &value, 100'000),
        0
    );
    const auto err = this->pop_error();
    EXPECT_EQ(err.Etype, EC_ERR_TYPE_PACKET_ERROR);
    EXPECT_EQ(err.Index, 0x1C12);
    EXPECT_EQ(err.SubIdx, 2);
    EXPECT_EQ(err.ErrorCode, 1);
}

/// @brief it should return the EEPROM word that the slave reads.
TEST_F(SOEMTest, EEPROMReadReturnsTheWord) {
    this->slave.registers[ECT_REG_EEPDAT] = 0x0000'0002;
    EXPECT_EQ(ecx_readeeprom2(this->context.get(), 1, EC_TIMEOUTEEP), 0x0000'0002);
}

/// @brief it should return 0 and push a packet error when the slave refuses an EEPROM
/// read, instead of the word of the read before.
TEST_F(SOEMTest, EEPROMReadFailsWhenTheSlaveRefusesIt) {
    this->slave.registers[ECT_REG_EEPSTAT] = EC_ESTAT_NACK;
    this->slave.registers[ECT_REG_EEPDAT] = 0x0000'0002;
    EXPECT_EQ(ecx_readeeprom2(this->context.get(), 1, EC_TIMEOUTEEP), 0);
    const auto err = this->pop_error();
    EXPECT_EQ(err.Etype, EC_ERR_TYPE_PACKET_ERROR);
    EXPECT_EQ(err.Slave, 1);
    EXPECT_EQ(err.ErrorCode, 30);
}
}
