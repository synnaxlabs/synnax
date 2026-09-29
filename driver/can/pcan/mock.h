// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstring>
#include <deque>
#include <map>
#include <mutex>
#include <string>
#include <vector>

#include "driver/can/pcan/api.h"

namespace driver::can::pcan {
/// @brief a simulated PCAN-Basic library. Tests queue the messages and statuses reads
/// return, and inspect the messages writes send.
class MockAPI final : public API {
public:
    /// @brief a channel the mock reports to scans.
    struct Channel {
        std::uint32_t condition = PCAN_CHANNEL_AVAILABLE;
        std::string hardware_name;
        std::uint32_t features = 0;
    };

    /// @brief one result of Read or ReadFD.
    struct Received {
        TPCANStatus status = PCAN_ERROR_OK;
        TPCANMsgFD msg{};
        std::uint64_t timestamp_us = 0;
    };

    std::mutex mu;
    /// @brief the channels scans find, by handle.
    std::map<TPCANHandle, Channel> channels;
    /// @brief the results reads return in order. An empty queue reads as
    /// PCAN_ERROR_QRCVEMPTY.
    std::deque<Received> reads;
    /// @brief every classic message Write sent.
    std::vector<TPCANMsg> written;
    /// @brief every message WriteFD sent.
    std::vector<TPCANMsgFD> written_fd;
    /// @brief the status Initialize and InitializeFD return.
    TPCANStatus initialize_status = PCAN_ERROR_OK;
    /// @brief the status Write and WriteFD return.
    TPCANStatus write_status = PCAN_ERROR_OK;
    /// @brief the BTR0BTR1 code of the last Initialize.
    TPCANBaudrate btr0btr1 = 0;
    /// @brief the bitrate string of the last InitializeFD.
    std::string fd_bitrate;
    /// @brief the handles SetValue put in listen only mode.
    std::vector<TPCANHandle> listen_only;
    /// @brief the handles Uninitialize released.
    std::vector<TPCANHandle> uninitialized;

    TPCANStatus Initialize(
        TPCANHandle,
        const TPCANBaudrate btr0btr1,
        TPCANType,
        std::uint32_t,
        std::uint16_t
    ) override {
        std::lock_guard lock(this->mu);
        this->btr0btr1 = btr0btr1;
        return this->initialize_status;
    }

    TPCANStatus InitializeFD(TPCANHandle, TPCANBitrateFD bitrate) override {
        std::lock_guard lock(this->mu);
        this->fd_bitrate = bitrate;
        return this->initialize_status;
    }

    TPCANStatus Uninitialize(const TPCANHandle channel) override {
        std::lock_guard lock(this->mu);
        this->uninitialized.push_back(channel);
        return PCAN_ERROR_OK;
    }

    TPCANStatus Read(TPCANHandle, TPCANMsg *msg, TPCANTimestamp *timestamp) override {
        std::lock_guard lock(this->mu);
        if (this->reads.empty()) return PCAN_ERROR_QRCVEMPTY;
        const auto read = this->reads.front();
        this->reads.pop_front();
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

    TPCANStatus
    ReadFD(TPCANHandle, TPCANMsgFD *msg, TPCANTimestampFD *timestamp) override {
        std::lock_guard lock(this->mu);
        if (this->reads.empty()) return PCAN_ERROR_QRCVEMPTY;
        const auto read = this->reads.front();
        this->reads.pop_front();
        *msg = read.msg;
        *timestamp = read.timestamp_us;
        return read.status;
    }

    TPCANStatus Write(TPCANHandle, TPCANMsg *msg) override {
        std::lock_guard lock(this->mu);
        if (this->write_status == PCAN_ERROR_OK) this->written.push_back(*msg);
        return this->write_status;
    }

    TPCANStatus WriteFD(TPCANHandle, TPCANMsgFD *msg) override {
        std::lock_guard lock(this->mu);
        if (this->write_status == PCAN_ERROR_OK) this->written_fd.push_back(*msg);
        return this->write_status;
    }

    TPCANStatus GetValue(
        const TPCANHandle channel,
        const TPCANParameter parameter,
        void *buffer,
        const std::uint32_t length
    ) override {
        std::lock_guard lock(this->mu);
        const auto it = this->channels.find(channel);
        if (parameter == PCAN_CHANNEL_CONDITION) {
            const std::uint32_t condition = it == this->channels.end()
                                              ? PCAN_CHANNEL_UNAVAILABLE
                                              : it->second.condition;
            std::memcpy(buffer, &condition, sizeof(condition));
            return PCAN_ERROR_OK;
        }
        if (it == this->channels.end()) return PCAN_ERROR_ILLPARAMVAL;
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
        const TPCANHandle channel,
        const TPCANParameter parameter,
        void *buffer,
        std::uint32_t
    ) override {
        std::lock_guard lock(this->mu);
        if (parameter != PCAN_LISTEN_ONLY) return PCAN_ERROR_ILLPARAMTYPE;
        if (*static_cast<std::uint32_t *>(buffer) == PCAN_PARAMETER_ON)
            this->listen_only.push_back(channel);
        return PCAN_ERROR_OK;
    }

    TPCANStatus
    GetErrorText(const TPCANStatus error, std::uint16_t, char *buffer) override {
        const auto text = "mock status " + std::to_string(error);
        std::strncpy(buffer, text.c_str(), 255);
        return PCAN_ERROR_OK;
    }
};
}
