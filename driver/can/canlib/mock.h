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
#include <cstring>
#include <deque>
#include <map>
#include <mutex>
#include <string>
#include <vector>

#include "driver/can/canlib/api.h"

namespace driver::can::canlib {
/// @brief a simulated CANlib. Tests queue the messages reads return and inspect how
/// the backend configured each handle and what it wrote.
class MockAPI final : public API {
public:
    /// @brief a channel the mock reports to scans.
    struct Channel {
        std::string description;
        std::uint64_t serial = 0;
        unsigned int number = 0;
    };

    /// @brief a message ReadWait returns.
    struct Message {
        long id = 0;
        std::array<std::uint8_t, 64> data{};
        unsigned int dlc = 0;
        unsigned int flags = 0;
        unsigned long time = 0;
    };

    /// @brief what the backend did to one handle.
    struct Handle {
        int channel = 0;
        int flags = 0;
        long freq = 0;
        long freq_brs = 0;
        unsigned int driver = 0;
        unsigned long timer_scale = 0;
        bool txecho = true;
        bool on = false;
        bool closed = false;
    };

    /// @brief a message canWrite sent.
    struct Written {
        canHandle hnd = 0;
        long id = 0;
        std::vector<std::uint8_t> data;
        unsigned int flags = 0;
    };

    std::mutex mu;
    std::vector<Channel> channels;
    std::map<canHandle, Handle> handles;
    std::deque<Message> reads;
    std::vector<Written> written;
    /// @brief when negative, the status OpenChannel returns.
    canStatus open_status = canOK;
    canStatus write_status = canOK;
    canStatus bus_params_status = canOK;
    /// @brief the timeout of the last ReadWait in milliseconds.
    unsigned long last_timeout = 0;

    canHandle OpenChannel(const int channel, const int flags) override {
        std::lock_guard lock(this->mu);
        if (this->open_status < 0) return this->open_status;
        const auto hnd = static_cast<canHandle>(this->handles.size());
        this->handles[hnd] = {.channel = channel, .flags = flags};
        return hnd;
    }

    canStatus Close(const canHandle hnd) override {
        std::lock_guard lock(this->mu);
        this->handles.at(hnd).closed = true;
        return canOK;
    }

    canStatus BusOn(const canHandle hnd) override {
        std::lock_guard lock(this->mu);
        this->handles.at(hnd).on = true;
        return canOK;
    }

    canStatus BusOff(const canHandle hnd) override {
        std::lock_guard lock(this->mu);
        this->handles.at(hnd).on = false;
        return canOK;
    }

    canStatus SetBusParams(
        const canHandle hnd,
        const long freq,
        unsigned int,
        unsigned int,
        unsigned int,
        unsigned int,
        unsigned int
    ) override {
        std::lock_guard lock(this->mu);
        this->handles.at(hnd).freq = freq;
        return this->bus_params_status;
    }

    canStatus SetBusParamsFd(
        const canHandle hnd,
        const long freq_brs,
        unsigned int,
        unsigned int,
        unsigned int
    ) override {
        std::lock_guard lock(this->mu);
        this->handles.at(hnd).freq_brs = freq_brs;
        return canOK;
    }

    canStatus
    SetBusOutputControl(const canHandle hnd, const unsigned int type) override {
        std::lock_guard lock(this->mu);
        this->handles.at(hnd).driver = type;
        return canOK;
    }

    canStatus IoCtl(
        const canHandle hnd,
        const unsigned int func,
        void *buf,
        unsigned int
    ) override {
        std::lock_guard lock(this->mu);
        auto &h = this->handles.at(hnd);
        if (func == canIOCTL_SET_TIMER_SCALE)
            h.timer_scale = *static_cast<unsigned long *>(buf);
        if (func == canIOCTL_SET_LOCAL_TXECHO)
            h.txecho = *static_cast<unsigned char *>(buf) != 0;
        return canOK;
    }

    canStatus ReadWait(
        canHandle,
        long *id,
        void *msg,
        unsigned int *dlc,
        unsigned int *flag,
        unsigned long *time,
        const unsigned long timeout
    ) override {
        std::lock_guard lock(this->mu);
        this->last_timeout = timeout;
        if (this->reads.empty()) return canERR_NOMSG;
        const auto m = this->reads.front();
        this->reads.pop_front();
        *id = m.id;
        std::memcpy(msg, m.data.data(), m.data.size());
        *dlc = m.dlc;
        *flag = m.flags;
        *time = m.time;
        return canOK;
    }

    canStatus Write(
        const canHandle hnd,
        const long id,
        void *msg,
        const unsigned int dlc,
        const unsigned int flag
    ) override {
        std::lock_guard lock(this->mu);
        if (this->write_status != canOK) return this->write_status;
        const auto *bytes = static_cast<std::uint8_t *>(msg);
        this->written.push_back({
            .hnd = hnd,
            .id = id,
            .data = {bytes, bytes + dlc},
            .flags = flag,
        });
        return canOK;
    }

    canStatus GetNumberOfChannels(int *count) override {
        std::lock_guard lock(this->mu);
        *count = static_cast<int>(this->channels.size());
        return canOK;
    }

    canStatus GetChannelData(
        const int channel,
        const int item,
        void *buffer,
        const std::size_t size
    ) override {
        std::lock_guard lock(this->mu);
        const auto &c = this->channels.at(channel);
        if (item == canCHANNELDATA_DEVDESCR_ASCII)
            std::strncpy(static_cast<char *>(buffer), c.description.c_str(), size - 1);
        if (item == canCHANNELDATA_CARD_SERIAL_NO)
            std::memcpy(buffer, &c.serial, sizeof(c.serial));
        if (item == canCHANNELDATA_CHAN_NO_ON_CARD)
            std::memcpy(buffer, &c.number, sizeof(c.number));
        return canOK;
    }

    canStatus
    GetErrorText(const canStatus err, char *buf, const unsigned int size) override {
        const auto text = "mock status " + std::to_string(err);
        std::strncpy(buf, text.c_str(), size - 1);
        return canOK;
    }
};
}
