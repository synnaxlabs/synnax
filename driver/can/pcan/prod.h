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
#include <utility>

#include "x/cpp/errors/errors.h"
#include "x/cpp/lib/lib.h"

#include "driver/can/pcan/api.h"

namespace driver::can::pcan {
/// @brief the PCAN-Basic library, PCANBasic.dll on Windows and MacCAN's
/// libPCBUSB.dylib on macOS, loaded at runtime.
class ProdAPI final : public API {
    std::unique_ptr<x::lib::Shared> lib;
    CAN_Initialize_t initialize;
    CAN_InitializeFD_t initialize_fd;
    CAN_Uninitialize_t uninitialize;
    CAN_Read_t read;
    CAN_ReadFD_t read_fd;
    CAN_Write_t write;
    CAN_WriteFD_t write_fd;
    CAN_GetValue_t get_value;
    CAN_SetValue_t set_value;
    CAN_GetErrorText_t get_error_text;

public:
    explicit ProdAPI(std::unique_ptr<x::lib::Shared> lib);

    /// @brief loads the library for this platform.
    /// @returns UNSUPPORTED_ERROR on a platform PCAN-Basic does not serve, the missing
    /// library error when the library is not installed, and CRITICAL_HARDWARE_ERROR
    /// when the library lacks a function the backend calls.
    static std::pair<std::shared_ptr<API>, x::errors::Error> load();

    TPCANStatus Initialize(
        TPCANHandle channel,
        TPCANBaudrate btr0btr1,
        TPCANType hw_type,
        std::uint32_t io_port,
        std::uint16_t interrupt
    ) override;
    TPCANStatus InitializeFD(TPCANHandle channel, TPCANBitrateFD bitrate) override;
    TPCANStatus Uninitialize(TPCANHandle channel) override;
    TPCANStatus
    Read(TPCANHandle channel, TPCANMsg *msg, TPCANTimestamp *timestamp) override;
    TPCANStatus
    ReadFD(TPCANHandle channel, TPCANMsgFD *msg, TPCANTimestampFD *timestamp) override;
    TPCANStatus Write(TPCANHandle channel, TPCANMsg *msg) override;
    TPCANStatus WriteFD(TPCANHandle channel, TPCANMsgFD *msg) override;
    TPCANStatus GetValue(
        TPCANHandle channel,
        TPCANParameter parameter,
        void *buffer,
        std::uint32_t length
    ) override;
    TPCANStatus SetValue(
        TPCANHandle channel,
        TPCANParameter parameter,
        void *buffer,
        std::uint32_t length
    ) override;
    TPCANStatus
    GetErrorText(TPCANStatus error, std::uint16_t language, char *buffer) override;
};
}
