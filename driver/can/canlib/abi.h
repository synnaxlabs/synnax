// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <cstdint>

/// @brief the subset of the Kvaser CANlib ABI the backend calls, transcribed from
/// Kvaser's CANlib SDK documentation. Kvaser's SDK license forbids sharing canlib.h, so
/// the Driver does not ship it.
namespace driver::can::canlib {
#ifdef _WIN32
#define CANLIB_API __stdcall
#else
#define CANLIB_API
#endif

using canHandle = int;
using canStatus = int;

constexpr canStatus canOK = 0;
constexpr canStatus canERR_PARAM = -1;
constexpr canStatus canERR_NOMSG = -2;
constexpr canStatus canERR_NOTFOUND = -3;
constexpr canStatus canERR_NOMEM = -4;
constexpr canStatus canERR_NOCHANNELS = -5;
constexpr canStatus canERR_INTERRUPTED = -6;
constexpr canStatus canERR_TIMEOUT = -7;
constexpr canStatus canERR_INVHANDLE = -10;
constexpr canStatus canERR_DRIVER = -12;
constexpr canStatus canERR_TXBUFOFL = -13;
constexpr canStatus canERR_HARDWARE = -15;
constexpr canStatus canERR_DYNALOAD = -16;
constexpr canStatus canERR_DYNALIB = -17;
constexpr canStatus canERR_DYNAINIT = -18;
constexpr canStatus canERR_NOT_SUPPORTED = -19;
constexpr canStatus canERR_DRIVERLOAD = -23;
constexpr canStatus canERR_NOCARD = -26;
constexpr canStatus canERR_REGISTRY = -28;
constexpr canStatus canERR_LICENSE = -29;
constexpr canStatus canERR_INTERNAL = -30;
constexpr canStatus canERR_NO_ACCESS = -31;
constexpr canStatus canERR_NOT_IMPLEMENTED = -32;

constexpr unsigned int canMSG_RTR = 0x0001;
constexpr unsigned int canMSG_STD = 0x0002;
constexpr unsigned int canMSG_EXT = 0x0004;
constexpr unsigned int canMSG_ERROR_FRAME = 0x0020;
constexpr unsigned int canMSGERR_HW_OVERRUN = 0x0200;
constexpr unsigned int canMSGERR_OVERRUN = 0x0600;
constexpr unsigned int canMSGERR_STUFF = 0x0800;
constexpr unsigned int canMSGERR_FORM = 0x1000;
constexpr unsigned int canMSGERR_CRC = 0x2000;
constexpr unsigned int canMSGERR_BIT0 = 0x4000;
constexpr unsigned int canMSGERR_BIT1 = 0x8000;
constexpr unsigned int canFDMSG_FDF = 0x010000;
constexpr unsigned int canFDMSG_BRS = 0x020000;
constexpr unsigned int canFDMSG_ESI = 0x040000;

constexpr int canOPEN_EXCLUSIVE = 0x0008;
constexpr int canOPEN_ACCEPT_VIRTUAL = 0x0020;
constexpr int canOPEN_CAN_FD = 0x0400;

constexpr unsigned int canDRIVER_SILENT = 1;
constexpr unsigned int canDRIVER_NORMAL = 4;

constexpr unsigned int canIOCTL_SET_TIMER_SCALE = 6;
constexpr unsigned int canIOCTL_SET_LOCAL_TXECHO = 32;

constexpr int canCHANNELDATA_CHANNEL_CAP = 1;
constexpr int canCHANNELDATA_CHAN_NO_ON_CARD = 6;
constexpr int canCHANNELDATA_CARD_SERIAL_NO = 7;
constexpr int canCHANNELDATA_CARD_UPC_NO = 11;
constexpr int canCHANNELDATA_DEVDESCR_ASCII = 26;

constexpr std::uint32_t canCHANNEL_CAP_VIRTUAL = 0x00010000;

constexpr long canBITRATE_1M = -1;
constexpr long canBITRATE_500K = -2;
constexpr long canBITRATE_250K = -3;
constexpr long canBITRATE_125K = -4;
constexpr long canBITRATE_100K = -5;
constexpr long canBITRATE_62K = -6;
constexpr long canBITRATE_50K = -7;
constexpr long canBITRATE_83K = -8;
constexpr long canBITRATE_10K = -9;
constexpr long canFD_BITRATE_500K_80P = -1000;
constexpr long canFD_BITRATE_1M_80P = -1001;
constexpr long canFD_BITRATE_2M_80P = -1002;
constexpr long canFD_BITRATE_4M_80P = -1003;
constexpr long canFD_BITRATE_8M_60P = -1004;

using canInitializeLibrary_t = void(CANLIB_API *)();
using canOpenChannel_t = canHandle(CANLIB_API *)(int channel, int flags);
using canClose_t = canStatus(CANLIB_API *)(canHandle hnd);
using canBusOn_t = canStatus(CANLIB_API *)(canHandle hnd);
using canSetBusParams_t = canStatus(CANLIB_API *)(
    canHandle hnd,
    long freq,
    unsigned int tseg1,
    unsigned int tseg2,
    unsigned int sjw,
    unsigned int noSamp,
    unsigned int syncmode
);
using canSetBusParamsFd_t = canStatus(CANLIB_API *)(
    canHandle hnd,
    long freq_brs,
    unsigned int tseg1_brs,
    unsigned int tseg2_brs,
    unsigned int sjw_brs
);
using canSetBusOutputControl_t =
    canStatus(CANLIB_API *)(canHandle hnd, unsigned int drivertype);
using canIoCtl_t = canStatus(CANLIB_API *)(
    canHandle hnd,
    unsigned int func,
    void *buf,
    unsigned int buflen
);
using canReadWait_t = canStatus(CANLIB_API *)(
    canHandle hnd,
    long *id,
    void *msg,
    unsigned int *dlc,
    unsigned int *flag,
    unsigned long *time,
    unsigned long timeout
);
using canWrite_t = canStatus(CANLIB_API *)(
    canHandle hnd,
    long id,
    void *msg,
    unsigned int dlc,
    unsigned int flag
);
using canEnumHardwareEx_t = canStatus(CANLIB_API *)(int *channelCount);
using canGetChannelData_t =
    canStatus(CANLIB_API *)(int channel, int item, void *buffer, std::size_t bufsize);
using canGetErrorText_t =
    canStatus(CANLIB_API *)(canStatus err, char *buf, unsigned int bufsiz);
}

#undef CANLIB_API
