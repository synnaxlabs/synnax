// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "driver/can/canlib/abi.h"

namespace driver::can::canlib {
/// @brief the CANlib calls the backend makes. ProdAPI loads them from the vendor
/// library, and MockAPI simulates them in tests.
class API {
public:
    virtual ~API() = default;

    virtual canHandle OpenChannel(int channel, int flags) = 0;
    virtual canStatus Close(canHandle hnd) = 0;
    virtual canStatus BusOn(canHandle hnd) = 0;
    virtual canStatus BusOff(canHandle hnd) = 0;
    virtual canStatus SetBusParams(
        canHandle hnd,
        long freq,
        unsigned int tseg1,
        unsigned int tseg2,
        unsigned int sjw,
        unsigned int no_samp,
        unsigned int syncmode
    ) = 0;
    virtual canStatus SetBusParamsFd(
        canHandle hnd,
        long freq_brs,
        unsigned int tseg1_brs,
        unsigned int tseg2_brs,
        unsigned int sjw_brs
    ) = 0;
    virtual canStatus SetBusOutputControl(canHandle hnd, unsigned int drivertype) = 0;
    virtual canStatus
    IoCtl(canHandle hnd, unsigned int func, void *buf, unsigned int buflen) = 0;
    virtual canStatus ReadWait(
        canHandle hnd,
        long *id,
        void *msg,
        unsigned int *dlc,
        unsigned int *flag,
        unsigned long *time,
        unsigned long timeout
    ) = 0;
    virtual canStatus
    Write(canHandle hnd, long id, void *msg, unsigned int dlc, unsigned int flag) = 0;
    virtual canStatus GetNumberOfChannels(int *count) = 0;
    virtual canStatus
    GetChannelData(int channel, int item, void *buffer, std::size_t size) = 0;
    virtual canStatus GetErrorText(canStatus err, char *buf, unsigned int size) = 0;
};
}
