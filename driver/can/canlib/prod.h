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

#include "driver/can/canlib/api.h"

namespace driver::can::canlib {
/// @brief Kvaser's canlib32.dll, loaded at runtime.
class ProdAPI final : public API {
    std::unique_ptr<x::lib::Shared> lib;
    canInitializeLibrary_t initialize_library = nullptr;
    canOpenChannel_t open_channel = nullptr;
    canClose_t close = nullptr;
    canBusOn_t bus_on = nullptr;
    canBusOff_t bus_off = nullptr;
    canSetBusParams_t set_bus_params = nullptr;
    canSetBusParamsFd_t set_bus_params_fd = nullptr;
    canSetBusOutputControl_t set_bus_output_control = nullptr;
    canIoCtl_t io_ctl = nullptr;
    canReadWait_t read_wait = nullptr;
    canWrite_t write = nullptr;
    canGetNumberOfChannels_t get_number_of_channels = nullptr;
    canGetChannelData_t get_channel_data = nullptr;
    canGetErrorText_t get_error_text = nullptr;

public:
    explicit ProdAPI(std::unique_ptr<x::lib::Shared> lib): lib(std::move(lib)) {}

    /// @brief loads and initializes CANlib.
    /// @returns UNSUPPORTED_ERROR off Windows, the missing library error when CANlib is
    /// not installed, and CRITICAL_HARDWARE_ERROR when the library lacks a function the
    /// backend calls.
    static std::pair<std::shared_ptr<API>, x::errors::Error> load();

    canHandle OpenChannel(int channel, int flags) override;
    canStatus Close(canHandle hnd) override;
    canStatus BusOn(canHandle hnd) override;
    canStatus BusOff(canHandle hnd) override;
    canStatus SetBusParams(
        canHandle hnd,
        long freq,
        unsigned int tseg1,
        unsigned int tseg2,
        unsigned int sjw,
        unsigned int no_samp,
        unsigned int syncmode
    ) override;
    canStatus SetBusParamsFd(
        canHandle hnd,
        long freq_brs,
        unsigned int tseg1_brs,
        unsigned int tseg2_brs,
        unsigned int sjw_brs
    ) override;
    canStatus SetBusOutputControl(canHandle hnd, unsigned int drivertype) override;
    canStatus
    IoCtl(canHandle hnd, unsigned int func, void *buf, unsigned int buflen) override;
    canStatus ReadWait(
        canHandle hnd,
        long *id,
        void *msg,
        unsigned int *dlc,
        unsigned int *flag,
        unsigned long *time,
        unsigned long timeout
    ) override;
    canStatus Write(
        canHandle hnd,
        long id,
        void *msg,
        unsigned int dlc,
        unsigned int flag
    ) override;
    canStatus GetNumberOfChannels(int *count) override;
    canStatus
    GetChannelData(int channel, int item, void *buffer, std::size_t size) override;
    canStatus GetErrorText(canStatus err, char *buf, unsigned int size) override;
};
}
