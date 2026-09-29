// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "driver/can/pcan/abi.h"

namespace driver::can::pcan {
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
};
}
