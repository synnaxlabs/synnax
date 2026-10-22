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
#include <array>
#include <chrono>
#include <condition_variable>
#include <cstring>
#include <deque>
#include <map>
#include <mutex>
#include <string>
#include <vector>

#include "driver/can/canlib/api.h"

namespace driver::can::canlib {
/// @brief a model of CANlib on Windows with Kvaser cards, written from the CANlib
/// documentation and Kvaser's linuxcan source. Channel numbers change only when
/// canEnumHardwareEx runs, and a replugged card takes new numbers after the cards
/// already present. Each handle has its own receive queue, a read blocks until a
/// message arrives or the timeout elapses, and calls on the handle of a removed card
/// fail with canERR_HARDWARE, which linuxcan returns for a removed device. Safe for
/// concurrent use.
class MockAPI final : public API {
public:
    /// @brief a Kvaser card with one or more channels.
    struct Card {
        std::string description;
        /// @brief the EAN as BCD digits, LSB first, as canCHANNELDATA_CARD_UPC_NO
        /// returns it.
        std::uint64_t ean = 0;
        std::uint64_t serial = 0;
        unsigned int channels = 1;
        bool is_virtual = false;
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
        int card = 0;
        unsigned int channel = 0;
        int flags = 0;
        long freq = 0;
        long freq_brs = 0;
        unsigned int driver = 0;
        unsigned long timer_scale = 0;
        bool txecho = true;
        bool on = false;
        bool closed = false;
        std::deque<Message> reads;
    };

    /// @brief a message canWrite sent.
    struct Written {
        canHandle hnd = 0;
        long id = 0;
        std::vector<std::uint8_t> data;
        unsigned int flags = 0;
    };

    /// @brief the status Write returns while the card is present.
    canStatus write_status = canOK;
    /// @brief the status SetBusParams returns while the card is present.
    canStatus bus_params_status = canOK;

    /// @brief plugs in a card.
    /// @returns the card's id.
    int attach(const Card &card) {
        std::lock_guard lock(this->mu);
        const auto id = static_cast<int>(this->cards.size());
        this->cards.push_back({.card = card});
        this->present.push_back(id);
        return id;
    }

    /// @brief removes a card. Reads waiting on its handles fail.
    void unplug(const int card) {
        std::lock_guard lock(this->mu);
        std::erase(this->present, card);
        this->cv.notify_all();
    }

    /// @brief plugs a removed card back in after the cards already present.
    void plug(const int card) {
        std::lock_guard lock(this->mu);
        this->present.push_back(card);
    }

    /// @brief makes another program hold a channel exclusively, or let it go.
    void hold(const int card, const unsigned int channel, const bool held) {
        std::lock_guard lock(this->mu);
        this->cards.at(card).held[channel] = held;
    }

    /// @brief puts a message on a channel's bus, which every handle on the bus
    /// receives.
    void deliver(const int card, const unsigned int channel, const Message &msg) {
        std::lock_guard lock(this->mu);
        for (auto &[hnd, h]: this->handles)
            if (h.card == card && h.channel == channel && h.on && !h.closed)
                h.reads.push_back(msg);
        this->cv.notify_all();
    }

    /// @returns a copy of a handle's state.
    Handle handle(const canHandle hnd) {
        std::lock_guard lock(this->mu);
        return this->handles.at(hnd);
    }

    /// @returns the number of handles the backend opened.
    std::size_t opened() {
        std::lock_guard lock(this->mu);
        return this->handles.size();
    }

    /// @returns the messages canWrite sent.
    std::vector<Written> written() {
        std::lock_guard lock(this->mu);
        return this->sent;
    }

    /// @returns the timeout of the last ReadWait in milliseconds.
    unsigned long last_timeout() {
        std::lock_guard lock(this->mu);
        return this->timeout;
    }

    canHandle OpenChannel(const int channel, const int flags) override {
        std::lock_guard lock(this->mu);
        if (channel < 0 || channel >= static_cast<int>(this->numbers.size()))
            return canERR_NOTFOUND;
        const auto [card, number] = this->numbers[channel];
        if (!this->attached(card)) return canERR_NOTFOUND;
        auto &c = this->cards.at(card);
        if (c.card.is_virtual && (flags & canOPEN_ACCEPT_VIRTUAL) == 0)
            return canERR_NOTFOUND;
        if (c.held[number]) return canERR_NOCHANNELS;
        const auto hnd = static_cast<canHandle>(this->handles.size());
        this->handles[hnd] = {.card = card, .channel = number, .flags = flags};
        return hnd;
    }

    canStatus Close(const canHandle hnd) override {
        std::lock_guard lock(this->mu);
        auto &h = this->handles.at(hnd);
        h.on = false;
        h.closed = true;
        return canOK;
    }

    canStatus BusOn(const canHandle hnd) override {
        std::lock_guard lock(this->mu);
        auto &h = this->handles.at(hnd);
        if (!this->attached(h.card)) return canERR_HARDWARE;
        h.on = true;
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
        auto &h = this->handles.at(hnd);
        if (!this->attached(h.card)) return canERR_HARDWARE;
        h.freq = freq;
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
        auto &h = this->handles.at(hnd);
        if (!this->attached(h.card)) return canERR_HARDWARE;
        h.freq_brs = freq_brs;
        return canOK;
    }

    canStatus
    SetBusOutputControl(const canHandle hnd, const unsigned int type) override {
        std::lock_guard lock(this->mu);
        auto &h = this->handles.at(hnd);
        if (!this->attached(h.card)) return canERR_HARDWARE;
        h.driver = type;
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
        if (!this->attached(h.card)) return canERR_HARDWARE;
        if (func == canIOCTL_SET_TIMER_SCALE)
            h.timer_scale = *static_cast<unsigned long *>(buf);
        if (func == canIOCTL_SET_LOCAL_TXECHO)
            h.txecho = *static_cast<unsigned char *>(buf) != 0;
        return canOK;
    }

    canStatus ReadWait(
        const canHandle hnd,
        long *id,
        void *msg,
        unsigned int *dlc,
        unsigned int *flag,
        unsigned long *time,
        const unsigned long timeout
    ) override {
        std::unique_lock lock(this->mu);
        this->timeout = timeout;
        auto &h = this->handles.at(hnd);
        this->cv.wait_for(lock, std::chrono::milliseconds(timeout), [&] {
            return !h.reads.empty() || !this->attached(h.card);
        });
        if (!this->attached(h.card)) return canERR_HARDWARE;
        if (h.reads.empty()) return canERR_NOMSG;
        const auto m = h.reads.front();
        h.reads.pop_front();
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
        if (!this->attached(this->handles.at(hnd).card)) return canERR_HARDWARE;
        if (this->write_status != canOK) return this->write_status;
        const auto *bytes = static_cast<std::uint8_t *>(msg);
        this->sent.push_back({
            .hnd = hnd,
            .id = id,
            .data = {bytes, bytes + dlc},
            .flags = flag,
        });
        return canOK;
    }

    canStatus EnumHardwareEx(int *count) override {
        std::lock_guard lock(this->mu);
        this->numbers.clear();
        for (const auto card: this->present)
            for (unsigned int n = 0; n < this->cards.at(card).card.channels; n++)
                this->numbers.emplace_back(card, n);
        *count = static_cast<int>(this->numbers.size());
        return canOK;
    }

    canStatus GetChannelData(
        const int channel,
        const int item,
        void *buffer,
        const std::size_t size
    ) override {
        std::lock_guard lock(this->mu);
        if (channel < 0 || channel >= static_cast<int>(this->numbers.size()))
            return canERR_NOTFOUND;
        const auto [card, number] = this->numbers[channel];
        const auto &c = this->cards.at(card).card;
        if (item == canCHANNELDATA_DEVDESCR_ASCII)
            std::strncpy(static_cast<char *>(buffer), c.description.c_str(), size - 1);
        if (item == canCHANNELDATA_CARD_SERIAL_NO)
            std::memcpy(buffer, &c.serial, sizeof(c.serial));
        if (item == canCHANNELDATA_CARD_UPC_NO)
            std::memcpy(buffer, &c.ean, sizeof(c.ean));
        if (item == canCHANNELDATA_CHAN_NO_ON_CARD)
            std::memcpy(buffer, &number, sizeof(number));
        if (item == canCHANNELDATA_CHANNEL_CAP) {
            const std::uint32_t cap = c.is_virtual ? canCHANNEL_CAP_VIRTUAL : 0;
            std::memcpy(buffer, &cap, sizeof(cap));
        }
        return canOK;
    }

    canStatus
    GetErrorText(const canStatus err, char *buf, const unsigned int size) override {
        const auto text = "mock status " + std::to_string(err);
        std::strncpy(buf, text.c_str(), size - 1);
        return canOK;
    }

private:
    struct State {
        Card card;
        std::map<unsigned int, bool> held;
    };

    std::mutex mu;
    std::condition_variable cv;
    std::vector<State> cards;
    /// @brief the ids of the cards plugged in, in plug order.
    std::vector<int> present;
    /// @brief the card and channel on the card of each channel number, as of the last
    /// enumeration.
    std::vector<std::pair<int, unsigned int>> numbers;
    std::map<canHandle, Handle> handles;
    std::vector<Written> sent;
    unsigned long timeout = 0;

    bool attached(const int card) const {
        return std::ranges::find(this->present, card) != this->present.end();
    }
};
}
