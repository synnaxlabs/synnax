// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstddef>
#include <cstdint>
#include <cstring>
#include <map>
#include <memory>

#include "gtest/gtest.h"
#include "soem/soem.h"

namespace {
/// @brief a bus that answers the AL status reads of SOEM, keyed by configured address.
struct Bus {
    /// @brief the AL status of each slave on the bus.
    std::map<uint16_t, ec_alstatust> slaves;
    /// @brief true when every frame is lost.
    bool frames_lost = false;

    /// @brief answers the frame in buf, returning the work counter of its last
    /// datagram.
    int answer(uint8_t *buf) const {
        size_t pos = sizeof(uint16_t);
        int wkc = 0;
        while (true) {
            const uint8_t cmd = buf[pos];
            uint16_t adp = 0;
            uint16_t dlength = 0;
            std::memcpy(&adp, &buf[pos + 2], sizeof(adp));
            std::memcpy(&dlength, &buf[pos + 6], sizeof(dlength));
            const size_t length = dlength & 0x07FF;
            uint8_t *data = &buf[pos + 10];
            wkc = 0;
            if (cmd == EC_CMD_BRD) {
                uint16_t state = 0;
                for (const auto &[_, status]: this->slaves)
                    state |= status.alstatus;
                std::memcpy(data, &state, std::min(length, sizeof(state)));
                wkc = static_cast<int>(this->slaves.size());
            } else if (
                const auto it = this->slaves.find(adp);
                cmd == EC_CMD_FPRD && it != this->slaves.end()
            ) {
                std::memcpy(data, &it->second, std::min(length, sizeof(ec_alstatust)));
                wkc = 1;
            }
            const auto counter = static_cast<uint16_t>(wkc);
            std::memcpy(&data[length], &counter, sizeof(counter));
            if ((dlength & EC_DATAGRAMFOLLOWS) == 0) return wkc;
            pos += 10 + length + EC_WKCSIZE;
        }
    }
};

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
    const auto &bus = bus_of(port);
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
    this->bus.slaves = {
        {0x1001, {.alstatus = OP}},
        {0x1002, {.alstatus = SAFE_OP_ERROR, .alstatuscode = 0x001B}},
    };
    EXPECT_EQ(ecx_readstate(this->context.get()), SAFE_OP);
    EXPECT_EQ(this->context->slavelist[1].state, OP);
    EXPECT_EQ(this->context->slavelist[2].state, SAFE_OP_ERROR);
    EXPECT_EQ(this->context->slavelist[2].ALstatuscode, 0x001B);
}

/// @brief it should read a slave that does not answer in a frame that comes back as in
/// no state.
TEST_F(ReadStateTest, ReadsNoStateForASlaveThatDoesNotAnswer) {
    this->bus.slaves = {{0x1001, {.alstatus = OP}}};
    EXPECT_EQ(ecx_readstate(this->context.get()), NONE);
    EXPECT_EQ(this->context->slavelist[1].state, OP);
    EXPECT_EQ(this->context->slavelist[2].state, NONE);
}
}
