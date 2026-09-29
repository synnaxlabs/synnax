// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <string>
#include <utility>

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "driver/can/pcan/abi.h"

namespace driver::can::pcan {
/// @brief signals that a channel's receive queue got a frame. A signal only follows a
/// frame that reaches the queue after the queue was read empty, so a reader must drain
/// the queue before it waits.
class ReceiveEvent {
public:
    virtual ~ReceiveEvent() = default;

    /// @brief blocks until the event signals or the timeout elapses.
    /// @returns CRITICAL_HARDWARE_ERROR when the wait fails.
    [[nodiscard]] virtual x::errors::Error wait(x::telem::TimeSpan timeout) = 0;
};

/// @brief the PCAN-Basic calls the backend makes. ProdAPI loads them from the vendor
/// library, and MockAPI simulates them in tests.
class API {
public:
    virtual ~API() = default;

    virtual TPCANStatus Initialize(
        TPCANHandle channel,
        TPCANBaudrate btr0btr1,
        TPCANType hw_type,
        std::uint32_t io_port,
        std::uint16_t interrupt
    ) = 0;
    virtual TPCANStatus InitializeFD(TPCANHandle channel, TPCANBitrateFD bitrate) = 0;
    virtual TPCANStatus Uninitialize(TPCANHandle channel) = 0;
    virtual TPCANStatus
    Read(TPCANHandle channel, TPCANMsg *msg, TPCANTimestamp *timestamp) = 0;
    virtual TPCANStatus
    ReadFD(TPCANHandle channel, TPCANMsgFD *msg, TPCANTimestampFD *timestamp) = 0;
    virtual TPCANStatus Write(TPCANHandle channel, TPCANMsg *msg) = 0;
    virtual TPCANStatus WriteFD(TPCANHandle channel, TPCANMsgFD *msg) = 0;
    virtual TPCANStatus GetValue(
        TPCANHandle channel,
        TPCANParameter parameter,
        void *buffer,
        std::uint32_t length
    ) = 0;
    virtual TPCANStatus SetValue(
        TPCANHandle channel,
        TPCANParameter parameter,
        void *buffer,
        std::uint32_t length
    ) = 0;
    virtual TPCANStatus
    GetErrorText(TPCANStatus error, std::uint16_t language, char *buffer) = 0;
    /// @brief registers a receive event on an initialized channel. The event must be
    /// destroyed after Uninitialize releases the channel.
    /// @returns CRITICAL_HARDWARE_ERROR when PCAN-Basic cannot register the event.
    virtual std::pair<std::unique_ptr<ReceiveEvent>, x::errors::Error>
    OpenReceiveEvent(TPCANHandle channel) = 0;
};

/// @returns PCAN-Basic's description of a status code.
[[nodiscard]] std::string describe(API &api, TPCANStatus status);
}
