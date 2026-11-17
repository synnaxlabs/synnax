// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <array>
#include <cstddef>
#include <cstdint>
#include <cstring>
#include <deque>
#include <functional>
#include <map>
#include <memory>
#include <set>
#include <vector>

#include "gtest/gtest.h"
#include "soem/soem.h"

#include "soem_patched.h"

namespace {
/// The register addresses of the mailboxes of each slave.
constexpr uint16_t MAILBOX_OUT = 0x1000;
constexpr uint16_t MAILBOX_IN = 0x1100;
constexpr uint16_t MAILBOX_LENGTH = 128;
/// The byte that each input of a logical read gets.
constexpr uint8_t INPUT = 0xAB;
/// The distributed clock time that the reference clock gives.
constexpr int64_t DC_TIME = 123'456'789;

using Bytes = std::vector<uint8_t>;

/// @brief a slave on the Bus.
struct Slave {
    /// @brief the 32-bit registers that the slave reports, by address.
    std::map<uint16_t, uint32_t> registers;
    /// @brief returns the mailbox replies of the slave to a request.
    std::function<std::vector<Bytes>()> respond;
    /// @brief the mailbox replies that the master has not read.
    std::deque<Bytes> replies;

    void read(const uint16_t address, uint8_t *data, const size_t length) {
        std::memset(data, 0, length);
        if (address == ECT_REG_SM1STAT) {
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
    /// @brief the working counter of a logical read or write.
    int process_wkc = 0;
    /// @brief the frames, by the order in which they were sent from 0, that do not
    /// come back.
    std::set<int> lost;
    /// @brief the frames, by the order in which they were sent from 0, that come back
    /// only at release().
    std::set<int> late;
    /// @brief the working counter of the last datagram of each late frame that has not
    /// come back, by index.
    std::map<uint8_t, int> held;
    /// @brief the number of frames sent with ecx_outframe_red.
    int sent = 0;
    /// @brief the working counter of the last datagram of each frame that came back,
    /// by index.
    std::map<uint8_t, int> received;

    /// @brief brings back the late frames.
    void release() {
        this->received.merge(this->held);
        this->held.clear();
    }

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
            if (cmd == EC_CMD_FPRD && it != this->slaves.end()) {
                it->second.read(ado, data, length);
                wkc = 1;
            } else if (cmd == EC_CMD_FPWR && it != this->slaves.end()) {
                it->second.write(ado);
                wkc = 1;
            } else if (cmd == EC_CMD_LRW || cmd == EC_CMD_LRD) {
                std::memset(data, INPUT, length);
                wkc = this->process_wkc;
            } else if (cmd == EC_CMD_LWR) {
                wkc = this->process_wkc;
            } else if (cmd == EC_CMD_FRMW && it != this->slaves.end()) {
                std::memcpy(data, &DC_TIME, std::min(length, sizeof(DC_TIME)));
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

/// @brief returns the SDO abort reply for index:sub_index with code.
Bytes abort_reply(const uint16_t index, const uint8_t sub_index, const uint32_t code) {
    return {
        10,
        0,
        0,
        0,
        0,
        ECT_MBXT_COE,
        0x00,
        ECT_COES_SDOREQ << 4,
        ECT_SDO_ABORT,
        static_cast<uint8_t>(index & 0xFF),
        static_cast<uint8_t>(index >> 8),
        sub_index,
        static_cast<uint8_t>(code & 0xFF),
        static_cast<uint8_t>((code >> 8) & 0xFF),
        static_cast<uint8_t>((code >> 16) & 0xFF),
        static_cast<uint8_t>(code >> 24),
    };
}

/// The abort code of an object that does not exist.
constexpr uint32_t NO_OBJECT = 0x0602'0000;

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

int ecx_outframe_red(ecx_portt *port, const uint8 idx) {
    auto &bus = bus_of(port);
    const auto length = port->txbuflength[idx] - ETH_HEADERSIZE;
    std::memcpy(port->rxbuf[idx], &port->txbuf[idx][ETH_HEADERSIZE], length);
    const int wkc = bus.answer(port->rxbuf[idx]);
    const int order = bus.sent++;
    if (bus.late.contains(order))
        bus.held[idx] = wkc;
    else if (!bus.lost.contains(order))
        bus.received[idx] = wkc;
    return 0;
}

int ecx_waitinframe(ecx_portt *port, const uint8 idx, int) {
    auto &received = bus_of(port).received;
    const auto it = received.find(idx);
    if (it == received.end()) return EC_NOFRAME;
    const int wkc = it->second;
    received.erase(it);
    return wkc;
}

int ecx_srconfirm(ecx_portt *port, const uint8 idx, int) {
    const auto length = port->txbuflength[idx] - ETH_HEADERSIZE;
    std::memcpy(port->rxbuf[idx], &port->txbuf[idx][ETH_HEADERSIZE], length);
    return bus_of(port).answer(port->rxbuf[idx]);
}
}

namespace {
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

/// @brief it should report the abort of the object that an SDO read asks for.
TEST_F(SOEMTest, SDOReadReportsTheAbortOfItsObject) {
    this->slave.respond = [] {
        return std::vector<Bytes>{abort_reply(0x1C12, 2, NO_OBJECT)};
    };
    uint16_t value = 0;
    int size = sizeof(value);
    EXPECT_LE(
        ecx_SDOread(this->context.get(), 1, 0x1C12, 2, FALSE, &size, &value, 100'000),
        0
    );
    const auto err = this->pop_error();
    EXPECT_EQ(err.Etype, EC_ERR_TYPE_SDO_ERROR);
    EXPECT_EQ(err.AbortCode, NO_OBJECT);
}

/// @brief it should fail an SDO read with an unexpected frame error when the abort is
/// for another object, as an abort of an earlier request is.
TEST_F(SOEMTest, SDOReadFailsOnAnAbortOfAnotherObject) {
    this->slave.respond = [] {
        return std::vector<Bytes>{abort_reply(0x1C12, 1, NO_OBJECT)};
    };
    uint16_t value = 0;
    int size = sizeof(value);
    EXPECT_LE(
        ecx_SDOread(this->context.get(), 1, 0x1C12, 2, FALSE, &size, &value, 100'000),
        0
    );
    const auto err = this->pop_error();
    EXPECT_EQ(err.Etype, EC_ERR_TYPE_PACKET_ERROR);
    EXPECT_EQ(err.ErrorCode, 1);
}

/// @brief it should fail an expedited SDO write with an unexpected frame error when the
/// abort is for another object.
TEST_F(SOEMTest, ExpeditedSDOWriteFailsOnAnAbortOfAnotherObject) {
    this->slave.respond = [] {
        return std::vector<Bytes>{abort_reply(0x8040, 1, NO_OBJECT)};
    };
    uint16_t value = 3;
    EXPECT_LE(
        ecx_SDOwrite(
            this->context.get(),
            1,
            0x8040,
            2,
            FALSE,
            sizeof(value),
            &value,
            100'000
        ),
        0
    );
    const auto err = this->pop_error();
    EXPECT_EQ(err.Etype, EC_ERR_TYPE_PACKET_ERROR);
    EXPECT_EQ(err.ErrorCode, 1);
}

/// @brief it should fail a normal SDO write with an unexpected frame error when the
/// abort is for another object.
TEST_F(SOEMTest, NormalSDOWriteFailsOnAnAbortOfAnotherObject) {
    this->slave.respond = [] {
        return std::vector<Bytes>{abort_reply(0x8040, 1, NO_OBJECT)};
    };
    uint64_t value = 3;
    EXPECT_LE(
        ecx_SDOwrite(
            this->context.get(),
            1,
            0x8040,
            2,
            FALSE,
            sizeof(value),
            &value,
            100'000
        ),
        0
    );
    const auto err = this->pop_error();
    EXPECT_EQ(err.Etype, EC_ERR_TYPE_PACKET_ERROR);
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

/// A value that no AL status read writes.
constexpr uint16_t UNREAD = 0xFFFF;

class ProcessDataTest : public testing::Test {
protected:
    Bus bus;
    std::unique_ptr<ecx_contextt> context = std::make_unique<ecx_contextt>();
    std::vector<uint8_t> iomap = std::vector<uint8_t>(EC_MAXLRWDATA);
    std::array<uint16_t, EC_MAXSLAVE> alstat{};
    ec_inflightT inflight{};
    ec_groupt &group = this->context->grouplist[1];

    void SetUp() override {
        this->context->userdata = &this->bus;
        this->bus.process_wkc = 3;
        this->alstat.fill(UNREAD);
        this->set_inputs(8);
    }

    /// @brief maps bytes of inputs and no outputs in one segment of group 1.
    void set_inputs(const uint32_t bytes) {
        this->group.inputs = this->iomap.data();
        this->group.outputs = this->iomap.data();
        this->group.Ibytes = bytes;
        this->group.Obytes = 0;
        this->group.nsegments = 1;
        this->group.IOsegment[0] = bytes;
    }

    /// @brief adds a slave of group 1 in state with inputs bits of inputs, and returns
    /// its index.
    uint16_t add_slave(const uint16_t state, const uint16_t inputs = 8) {
        const auto slave = static_cast<uint16_t>(++this->context->slavecount);
        auto &rec = this->context->slavelist[slave];
        rec.configadr = EC_NODEOFFSET + slave;
        rec.group = 1;
        rec.Ibits = inputs;
        this->bus.slaves[rec.configadr].registers[ECT_REG_ALSTAT] = state;
        return slave;
    }

    /// @brief sends the process data of group 1 and keeps its frames in flight.
    void send() {
        EXPECT_GT(
            ecx_send_processdata_inflight(this->context.get(), 1, &this->inflight),
            0
        );
    }

    /// @brief receives the oldest send of group 1 in flight, returning the working
    /// counter.
    int receive() {
        return ecx_receive_processdata_inflight(
            this->context.get(),
            1,
            1000,
            &this->inflight,
            this->alstat.data()
        );
    }

    /// @brief sends and receives the process data of group 1, returning the working
    /// counter.
    int cycle() {
        this->send();
        return this->receive();
    }
};

/// @brief it should read the AL status of each slave with inputs in the process data
/// frame, and no status of a slave without inputs.
TEST_F(ProcessDataTest, ReadsTheALStatusOfEachSlaveWithInputsInTheFrame) {
    this->add_slave(EC_STATE_OPERATIONAL);
    this->add_slave(EC_STATE_SAFE_OP | EC_STATE_ERROR);
    this->add_slave(EC_STATE_PRE_OP);
    this->add_slave(EC_STATE_INIT, 0);
    this->cycle();
    EXPECT_EQ(this->bus.sent, 1);
    EXPECT_EQ(this->alstat[1], EC_STATE_OPERATIONAL);
    EXPECT_EQ(this->alstat[2], EC_STATE_SAFE_OP | EC_STATE_ERROR);
    EXPECT_EQ(this->alstat[3], EC_STATE_PRE_OP);
    EXPECT_EQ(this->alstat[4], UNREAD);
}

/// @brief it should return the working counter of the process data, not of the AL
/// status read at the end of the frame.
TEST_F(ProcessDataTest, ReturnsTheWorkingCounterOfTheProcessData) {
    this->add_slave(EC_STATE_OPERATIONAL);
    EXPECT_EQ(this->cycle(), 3);
    EXPECT_EQ(this->iomap[0], INPUT);
}

/// @brief it should give 0 to a slave that does not answer its AL status read.
TEST_F(ProcessDataTest, GivesZeroToASlaveThatDoesNotAnswer) {
    this->add_slave(EC_STATE_OPERATIONAL);
    const auto silent = this->add_slave(EC_STATE_OPERATIONAL);
    this->bus.slaves.erase(this->context->slavelist[silent].configadr);
    this->cycle();
    EXPECT_EQ(this->alstat[1], EC_STATE_OPERATIONAL);
    EXPECT_EQ(this->alstat[silent], 0);
}

/// @brief it should read the AL status after the distributed clock time, and still
/// read the time.
TEST_F(ProcessDataTest, ReadsTheALStatusAfterTheDCTime) {
    this->add_slave(EC_STATE_OPERATIONAL);
    this->add_slave(EC_STATE_SAFE_OP);
    this->group.hasdc = TRUE;
    this->group.DCnext = 1;
    EXPECT_EQ(this->cycle(), 3);
    EXPECT_EQ(this->context->DCtime, DC_TIME);
    EXPECT_EQ(this->alstat[1], EC_STATE_OPERATIONAL);
    EXPECT_EQ(this->alstat[2], EC_STATE_SAFE_OP);
}

/// @brief it should read the AL status in the logical read frame when a slave blocks
/// logical read-writes.
TEST_F(ProcessDataTest, ReadsTheALStatusInTheLogicalReadFrame) {
    this->add_slave(EC_STATE_PRE_OP);
    this->group.blockLRW = 1;
    EXPECT_EQ(this->cycle(), 3);
    EXPECT_EQ(this->bus.sent, 1);
    EXPECT_EQ(this->alstat[1], EC_STATE_PRE_OP);
}

/// @brief it should put the AL status reads that do not fit in the process data frame
/// in a frame of their own in the same send.
TEST_F(ProcessDataTest, PutsTheReadsThatDoNotFitInAFrameOfTheirOwn) {
    // Leaves room in the frame for one read.
    this->set_inputs(1470);
    this->add_slave(EC_STATE_OPERATIONAL);
    this->add_slave(EC_STATE_SAFE_OP);
    this->add_slave(EC_STATE_INIT);
    EXPECT_EQ(this->cycle(), 3);
    EXPECT_EQ(this->bus.sent, 2);
    EXPECT_EQ(this->alstat[1], EC_STATE_OPERATIONAL);
    EXPECT_EQ(this->alstat[2], EC_STATE_SAFE_OP);
    EXPECT_EQ(this->alstat[3], EC_STATE_INIT);
}

/// @brief it should give 0 to every slave while a frame of the send has not come back,
/// even a slave whose read is in a frame that came back.
TEST_F(ProcessDataTest, GivesZeroToEverySlaveWhileAFrameOfTheSendIsOut) {
    this->set_inputs(1470);
    this->add_slave(EC_STATE_OPERATIONAL);
    this->add_slave(EC_STATE_OPERATIONAL);
    this->add_slave(EC_STATE_OPERATIONAL);
    this->bus.lost = {1};
    EXPECT_EQ(this->cycle(), EC_NOFRAME);
    EXPECT_EQ(this->alstat[1], 0);
    EXPECT_EQ(this->alstat[2], 0);
    EXPECT_EQ(this->alstat[3], 0);
}

/// @brief it should keep a frame that comes back late in flight, and take it in a
/// later receive.
TEST_F(ProcessDataTest, TakesAFrameThatComesBackLate) {
    this->add_slave(EC_STATE_OPERATIONAL);
    this->bus.late = {0};
    EXPECT_EQ(this->cycle(), EC_NOFRAME);
    EXPECT_EQ(this->iomap[0], 0);
    this->bus.release();
    EXPECT_EQ(this->receive(), 3);
    EXPECT_EQ(this->iomap[0], INPUT);
    EXPECT_EQ(this->alstat[1], EC_STATE_OPERATIONAL);
}

/// @brief it should keep a frame that came back before the rest of its send, since
/// SOEM gives a frame once only.
TEST_F(ProcessDataTest, KeepsAFrameThatCameBackBeforeTheRestOfItsSend) {
    this->set_inputs(1470);
    this->add_slave(EC_STATE_OPERATIONAL);
    this->add_slave(EC_STATE_SAFE_OP);
    this->bus.late = {1};
    EXPECT_EQ(this->cycle(), EC_NOFRAME);
    this->bus.release();
    EXPECT_EQ(this->receive(), 3);
    EXPECT_EQ(this->iomap[0], INPUT);
    EXPECT_EQ(this->alstat[1], EC_STATE_OPERATIONAL);
    EXPECT_EQ(this->alstat[2], EC_STATE_SAFE_OP);
}

/// @brief it should count the working counter of a logical write twice, as a logical
/// read-write counts its outputs.
TEST_F(ProcessDataTest, CountsALogicalWriteTwice) {
    this->add_slave(EC_STATE_OPERATIONAL, 0);
    this->group.Ibytes = 0;
    this->group.Obytes = 8;
    this->group.blockLRW = 1;
    EXPECT_EQ(this->cycle(), 6);
}

/// @brief it should return the number of frames that the send put in flight.
TEST_F(ProcessDataTest, SendReturnsItsFrames) {
    this->set_inputs(1470);
    this->add_slave(EC_STATE_OPERATIONAL);
    this->add_slave(EC_STATE_OPERATIONAL);
    EXPECT_EQ(
        ecx_send_processdata_inflight(this->context.get(), 1, &this->inflight),
        2
    );
}

/// @brief it should take the sends in flight in the order sent.
TEST_F(ProcessDataTest, TakesTheSendsInTheOrderSent) {
    this->add_slave(EC_STATE_OPERATIONAL);
    this->send();
    this->bus.process_wkc = 2;
    this->send();
    EXPECT_EQ(this->receive(), 3);
    EXPECT_EQ(this->receive(), 2);
    EXPECT_EQ(this->receive(), EC_NOFRAME);
}

/// @brief it should free the frames of a send that it takes.
TEST_F(ProcessDataTest, FreesTheFramesOfASendThatItTakes) {
    this->set_inputs(1470);
    this->add_slave(EC_STATE_OPERATIONAL);
    this->add_slave(EC_STATE_OPERATIONAL);
    EXPECT_EQ(this->cycle(), 3);
    EXPECT_EQ(this->context->idxstack.pushed, 0);
    for (const auto bufstat: this->context->port.rxbufstat)
        EXPECT_EQ(bufstat, EC_BUF_EMPTY);
}

/// @brief it should free the frames of the oldest send on a drop, and keep the later
/// sends in flight.
TEST_F(ProcessDataTest, DropFreesTheOldestSend) {
    this->add_slave(EC_STATE_OPERATIONAL);
    this->bus.lost = {0};
    this->send();
    const auto dropped = this->context->idxstack.idx[0];
    this->bus.process_wkc = 2;
    this->send();
    EXPECT_EQ(this->receive(), EC_NOFRAME);
    ecx_drop_processdata_inflight(this->context.get(), &this->inflight);
    EXPECT_EQ(this->context->port.rxbufstat[dropped], EC_BUF_EMPTY);
    EXPECT_EQ(this->receive(), 2);
}

/// @brief it should return no frame and do nothing on a drop when no send is in
/// flight.
TEST_F(ProcessDataTest, ReturnsNoFrameWhenNoSendIsInFlight) {
    this->add_slave(EC_STATE_OPERATIONAL);
    ecx_drop_processdata_inflight(this->context.get(), &this->inflight);
    EXPECT_EQ(this->receive(), EC_NOFRAME);
    EXPECT_EQ(this->alstat[1], 0);
}
}
