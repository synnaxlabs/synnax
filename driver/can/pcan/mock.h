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
#include <condition_variable>
#include <cstring>
#include <deque>
#include <format>
#include <map>
#include <mutex>
#include <optional>
#include <string>
#include <vector>

#include "driver/can/pcan/api.h"

namespace driver::can::pcan {
/// @brief a model of PCAN-Basic with PEAK channels, written from the PCAN-Basic
/// documentation and PEAK's support forum. Each channel keeps its own state: a channel
/// initialized once per process, a receive queue, a bus status, and whether its
/// hardware is attached. An unplugged channel stays occupied, reads empty, and takes
/// writes, and only CAN_GetStatus reports the unplug, as PCAN_ERROR_ILLHW
/// (forum.peak-system.com/viewtopic.php?p=20202). Safe for concurrent use.
class MockAPI final : public API {
public:
    /// @brief one result of Read or ReadFD.
    struct Received {
        TPCANStatus status = PCAN_ERROR_OK;
        TPCANMsgFD msg{};
        std::uint64_t timestamp_us = 0;
    };

    /// @brief a channel the backend can find.
    struct Channel {
        std::string hardware_name;
        std::uint32_t features = 0;
        bool attached = true;
        /// @brief true while another PCAN-Basic application holds the channel.
        bool claimed = false;
        /// @brief the bitrate PCAN-View runs the channel at while it shares it.
        std::optional<std::string> monitor;
        bool initialized = false;
        /// @brief the bitrate this process initialized the channel at.
        std::string bitrate;
        bool listen_only = false;
        /// @brief the bus status bits CAN_GetStatus reports.
        TPCANStatus bus = PCAN_ERROR_OK;
        /// @brief the status Write and WriteFD return while the bus takes frames.
        TPCANStatus write_status = PCAN_ERROR_OK;
        std::deque<Received> reads;
        bool signaled = false;
        std::vector<TPCANMsg> written;
        std::vector<TPCANMsgFD> written_fd;
    };

    /// @brief a receive event the backend destroyed.
    struct Release {
        TPCANHandle channel;
        /// @brief true when Uninitialize released the channel before the event.
        bool after_uninitialize;
    };

    /// @brief the status Initialize and InitializeFD return in place of the model's,
    /// such as PCAN_ERROR_NODRIVER.
    std::optional<TPCANStatus> initialize_status;
    /// @brief the error OpenReceiveEvent returns.
    x::errors::Error receive_event_error = x::errors::NIL;
    /// @brief the error a receive event's wait returns.
    x::errors::Error wait_error = x::errors::NIL;

    /// @brief adds a channel the backend can find.
    void add(const TPCANHandle handle, Channel channel) {
        std::lock_guard lock(this->mu);
        this->channels[handle] = std::move(channel);
    }

    /// @brief unplugs the hardware of a channel.
    void unplug(const TPCANHandle handle) {
        std::lock_guard lock(this->mu);
        this->channels.at(handle).attached = false;
    }

    /// @brief plugs the hardware of a channel back in with an empty bus.
    void plug(const TPCANHandle handle) {
        std::lock_guard lock(this->mu);
        auto &ch = this->channels.at(handle);
        ch.attached = true;
        ch.bus = PCAN_ERROR_OK;
        ch.reads.clear();
    }

    /// @brief makes another PCAN-Basic application hold the channel, or let it go.
    void claim(const TPCANHandle handle, const bool claimed) {
        std::lock_guard lock(this->mu);
        this->channels.at(handle).claimed = claimed;
    }

    /// @brief makes PCAN-View share the channel at a bitrate, or let it go.
    void monitor(const TPCANHandle handle, std::optional<std::string> bitrate) {
        std::lock_guard lock(this->mu);
        this->channels.at(handle).monitor = std::move(bitrate);
    }

    /// @brief sets the bus status bits of a channel, such as PCAN_ERROR_BUSPASSIVE.
    void set_bus(const TPCANHandle handle, const TPCANStatus bus) {
        std::lock_guard lock(this->mu);
        this->channels.at(handle).bus = bus;
    }

    /// @brief sets the status writes to a channel return while its bus takes frames.
    void set_write_status(const TPCANHandle handle, const TPCANStatus status) {
        std::lock_guard lock(this->mu);
        this->channels.at(handle).write_status = status;
    }

    /// @brief queues a read result on a channel and signals its receive event, as
    /// PCAN-Basic does when a frame reaches the receive queue.
    void receive(const TPCANHandle handle, const Received &read) {
        std::lock_guard lock(this->mu);
        auto &ch = this->channels.at(handle);
        if (!ch.attached) return;
        ch.reads.push_back(read);
        ch.signaled = true;
        this->cv.notify_all();
    }

    /// @returns a copy of a channel's state.
    Channel channel(const TPCANHandle handle) {
        std::lock_guard lock(this->mu);
        return this->channels.at(handle);
    }

    /// @returns the handles Uninitialize released, in order.
    std::vector<TPCANHandle> uninitialized() {
        std::lock_guard lock(this->mu);
        return this->released_handles;
    }

    /// @returns the handles OpenReceiveEvent registered an event on.
    std::vector<TPCANHandle> receive_events() {
        std::lock_guard lock(this->mu);
        return this->events;
    }

    /// @returns the receive events the backend destroyed, in order.
    std::vector<Release> released() {
        std::lock_guard lock(this->mu);
        return this->releases;
    }

    /// @returns the number of times a receive event waited.
    std::size_t waits() {
        std::lock_guard lock(this->mu);
        return this->wait_count;
    }

    TPCANStatus Initialize(
        const TPCANHandle handle,
        const TPCANBaudrate btr0btr1,
        TPCANType,
        std::uint32_t,
        std::uint16_t
    ) override {
        std::lock_guard lock(this->mu);
        return this->initialize(handle, std::format("0x{:04X}", btr0btr1));
    }

    TPCANStatus
    InitializeFD(const TPCANHandle handle, TPCANBitrateFD bitrate) override {
        std::lock_guard lock(this->mu);
        return this->initialize(handle, bitrate);
    }

    TPCANStatus Uninitialize(const TPCANHandle handle) override {
        std::lock_guard lock(this->mu);
        const auto it = this->channels.find(handle);
        if (it == this->channels.end() || !it->second.initialized)
            return PCAN_ERROR_INITIALIZE;
        it->second.initialized = false;
        it->second.listen_only = false;
        it->second.reads.clear();
        this->released_handles.push_back(handle);
        return PCAN_ERROR_OK;
    }

    TPCANStatus
    Read(const TPCANHandle handle, TPCANMsg *msg, TPCANTimestamp *timestamp) override {
        std::lock_guard lock(this->mu);
        Received read;
        if (const auto status = this->take(handle, read); status != PCAN_ERROR_OK)
            return status;
        msg->ID = read.msg.ID;
        msg->MSGTYPE = read.msg.MSGTYPE;
        msg->LEN = read.msg.DLC;
        std::memcpy(msg->DATA, read.msg.DATA, sizeof(msg->DATA));
        timestamp->micros = static_cast<std::uint16_t>(read.timestamp_us % 1000);
        const auto millis = read.timestamp_us / 1000;
        timestamp->millis = static_cast<std::uint32_t>(millis & 0xFFFFFFFF);
        timestamp->millis_overflow = static_cast<std::uint16_t>(millis >> 32);
        return read.status;
    }

    TPCANStatus ReadFD(
        const TPCANHandle handle,
        TPCANMsgFD *msg,
        TPCANTimestampFD *timestamp
    ) override {
        std::lock_guard lock(this->mu);
        Received read;
        if (const auto status = this->take(handle, read); status != PCAN_ERROR_OK)
            return status;
        *msg = read.msg;
        *timestamp = read.timestamp_us;
        return read.status;
    }

    TPCANStatus Write(const TPCANHandle handle, TPCANMsg *msg) override {
        std::lock_guard lock(this->mu);
        auto [status, ch] = this->writable(handle);
        if (status == PCAN_ERROR_OK && ch->attached) ch->written.push_back(*msg);
        return status;
    }

    TPCANStatus WriteFD(const TPCANHandle handle, TPCANMsgFD *msg) override {
        std::lock_guard lock(this->mu);
        auto [status, ch] = this->writable(handle);
        if (status == PCAN_ERROR_OK && ch->attached) ch->written_fd.push_back(*msg);
        return status;
    }

    TPCANStatus GetStatus(const TPCANHandle handle) override {
        std::lock_guard lock(this->mu);
        const auto it = this->channels.find(handle);
        if (it == this->channels.end() || !it->second.initialized)
            return PCAN_ERROR_INITIALIZE;
        if (!it->second.attached) return PCAN_ERROR_ILLHW;
        return it->second.bus;
    }

    TPCANStatus GetValue(
        const TPCANHandle handle,
        const TPCANParameter parameter,
        void *buffer,
        const std::uint32_t length
    ) override {
        std::lock_guard lock(this->mu);
        const auto it = this->channels.find(handle);
        if (parameter == PCAN_CHANNEL_CONDITION) {
            std::uint32_t condition = PCAN_CHANNEL_UNAVAILABLE;
            if (it != this->channels.end()) {
                const auto &ch = it->second;
                if (ch.initialized || ch.claimed)
                    condition = PCAN_CHANNEL_OCCUPIED;
                else if (!ch.attached)
                    condition = PCAN_CHANNEL_UNAVAILABLE;
                else if (ch.monitor.has_value())
                    condition = PCAN_CHANNEL_PCANVIEW;
                else
                    condition = PCAN_CHANNEL_AVAILABLE;
            }
            std::memcpy(buffer, &condition, sizeof(condition));
            return PCAN_ERROR_OK;
        }
        if (it == this->channels.end()) return PCAN_ERROR_ILLHW;
        if (parameter == PCAN_HARDWARE_NAME) {
            std::strncpy(
                static_cast<char *>(buffer),
                it->second.hardware_name.c_str(),
                length - 1
            );
            return PCAN_ERROR_OK;
        }
        if (parameter == PCAN_CHANNEL_FEATURES) {
            std::memcpy(buffer, &it->second.features, sizeof(it->second.features));
            return PCAN_ERROR_OK;
        }
        return PCAN_ERROR_ILLPARAMTYPE;
    }

    TPCANStatus SetValue(
        const TPCANHandle handle,
        const TPCANParameter parameter,
        void *buffer,
        std::uint32_t
    ) override {
        std::lock_guard lock(this->mu);
        if (parameter != PCAN_LISTEN_ONLY) return PCAN_ERROR_ILLPARAMTYPE;
        const auto it = this->channels.find(handle);
        if (it == this->channels.end()) return PCAN_ERROR_ILLHW;
        it->second.listen_only = *static_cast<std::uint32_t *>(buffer) ==
                                 PCAN_PARAMETER_ON;
        return PCAN_ERROR_OK;
    }

    TPCANStatus
    GetErrorText(const TPCANStatus error, std::uint16_t, char *buffer) override {
        const auto text = std::format("PCAN status 0x{:X}", error);
        std::strncpy(buffer, text.c_str(), 255);
        return PCAN_ERROR_OK;
    }

    std::pair<std::unique_ptr<ReceiveEvent>, x::errors::Error>
    OpenReceiveEvent(const TPCANHandle handle) override {
        std::lock_guard lock(this->mu);
        if (this->receive_event_error) return {nullptr, this->receive_event_error};
        this->events.push_back(handle);
        return {std::make_unique<Event>(*this, handle), x::errors::NIL};
    }

private:
    /// @brief an auto-reset receive event that a frame reaching the queue signals.
    class Event final : public ReceiveEvent {
        MockAPI &api;
        TPCANHandle handle;

    public:
        Event(MockAPI &api, const TPCANHandle handle): api(api), handle(handle) {}

        ~Event() override {
            std::lock_guard lock(this->api.mu);
            const auto &released = this->api.released_handles;
            this->api.releases.push_back({
                .channel = this->handle,
                .after_uninitialize = std::ranges::find(released, this->handle) !=
                                      released.end(),
            });
        }

        x::errors::Error wait(const x::telem::TimeSpan timeout) override {
            std::unique_lock lock(this->api.mu);
            this->api.wait_count++;
            auto &ch = this->api.channels.at(this->handle);
            this->api.cv.wait_for(lock, timeout.chrono(), [&] { return ch.signaled; });
            ch.signaled = false;
            return this->api.wait_error;
        }
    };

    std::mutex mu;
    std::condition_variable cv;
    std::map<TPCANHandle, Channel> channels;
    std::vector<TPCANHandle> released_handles;
    std::vector<TPCANHandle> events;
    std::vector<Release> releases;
    std::size_t wait_count = 0;

    /// @brief connects this process to a channel. A channel that this or another
    /// PCAN-Basic process holds refuses, and a channel PCAN-View runs at another
    /// bitrate connects with PCAN_ERROR_CAUTION (PCAN-Basic documentation,
    /// CAN_Initialize).
    TPCANStatus initialize(const TPCANHandle handle, const std::string &bitrate) {
        if (this->initialize_status.has_value()) return *this->initialize_status;
        const auto it = this->channels.find(handle);
        if (it == this->channels.end() || !it->second.attached) return PCAN_ERROR_ILLHW;
        auto &ch = it->second;
        if (ch.initialized || ch.claimed) return PCAN_ERROR_INITIALIZE;
        ch.initialized = true;
        ch.bitrate = bitrate;
        if (ch.monitor.has_value() && *ch.monitor != bitrate) return PCAN_ERROR_CAUTION;
        return PCAN_ERROR_OK;
    }

    /// @brief takes the next read result of a channel. An unplugged channel reads
    /// empty.
    TPCANStatus take(const TPCANHandle handle, Received &read) {
        const auto it = this->channels.find(handle);
        if (it == this->channels.end() || !it->second.initialized)
            return PCAN_ERROR_INITIALIZE;
        auto &ch = it->second;
        if (!ch.attached || ch.reads.empty()) return PCAN_ERROR_QRCVEMPTY;
        read = ch.reads.front();
        ch.reads.pop_front();
        return PCAN_ERROR_OK;
    }

    /// @returns the status a write to a channel gets. An unplugged channel takes the
    /// write and drops it.
    std::pair<TPCANStatus, Channel *> writable(const TPCANHandle handle) {
        const auto it = this->channels.find(handle);
        if (it == this->channels.end() || !it->second.initialized)
            return {PCAN_ERROR_INITIALIZE, nullptr};
        auto &ch = it->second;
        if (!ch.attached) return {PCAN_ERROR_OK, &ch};
        if ((ch.bus & PCAN_ERROR_BUSOFF) != 0) return {PCAN_ERROR_BUSOFF, &ch};
        return {ch.write_status, &ch};
    }
};
}
