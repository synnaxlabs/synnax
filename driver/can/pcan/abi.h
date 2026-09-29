// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>

/// @brief the subset of the PCAN-Basic ABI the backend calls, transcribed from PEAK's
/// PCAN-Basic documentation. MacCAN's PCBUSB exports the same ABI on macOS.
namespace driver::can::pcan {
#ifdef _WIN32
#define PCAN_API __stdcall
#else
#define PCAN_API
#endif

using TPCANHandle = std::uint16_t;
using TPCANStatus = std::uint32_t;
using TPCANParameter = std::uint8_t;
using TPCANMessageType = std::uint8_t;
using TPCANType = std::uint8_t;
using TPCANBaudrate = std::uint16_t;
using TPCANBitrateFD = char *;
using TPCANTimestampFD = std::uint64_t;

struct TPCANMsg {
    std::uint32_t ID;
    TPCANMessageType MSGTYPE;
    std::uint8_t LEN;
    std::uint8_t DATA[8];
};

struct TPCANTimestamp {
    std::uint32_t millis;
    std::uint16_t millis_overflow;
    std::uint16_t micros;
};

struct TPCANMsgFD {
    std::uint32_t ID;
    TPCANMessageType MSGTYPE;
    std::uint8_t DLC;
    std::uint8_t DATA[64];
};

constexpr TPCANHandle PCAN_USBBUS1 = 0x51;
constexpr TPCANHandle PCAN_USBBUS9 = 0x509;
constexpr TPCANHandle PCAN_PCIBUS1 = 0x41;
constexpr TPCANHandle PCAN_PCIBUS9 = 0x409;
constexpr TPCANHandle PCAN_LANBUS1 = 0x801;

constexpr TPCANStatus PCAN_ERROR_OK = 0x00000;
constexpr TPCANStatus PCAN_ERROR_XMTFULL = 0x00001;
constexpr TPCANStatus PCAN_ERROR_OVERRUN = 0x00002;
constexpr TPCANStatus PCAN_ERROR_BUSLIGHT = 0x00004;
constexpr TPCANStatus PCAN_ERROR_BUSHEAVY = 0x00008;
constexpr TPCANStatus PCAN_ERROR_BUSOFF = 0x00010;
constexpr TPCANStatus PCAN_ERROR_QRCVEMPTY = 0x00020;
constexpr TPCANStatus PCAN_ERROR_QOVERRUN = 0x00040;
constexpr TPCANStatus PCAN_ERROR_QXMTFULL = 0x00080;
constexpr TPCANStatus PCAN_ERROR_ILLPARAMTYPE = 0x04000;
constexpr TPCANStatus PCAN_ERROR_ILLPARAMVAL = 0x08000;
constexpr TPCANStatus PCAN_ERROR_BUSPASSIVE = 0x40000;

constexpr TPCANMessageType PCAN_MESSAGE_STANDARD = 0x00;
constexpr TPCANMessageType PCAN_MESSAGE_RTR = 0x01;
constexpr TPCANMessageType PCAN_MESSAGE_EXTENDED = 0x02;
constexpr TPCANMessageType PCAN_MESSAGE_FD = 0x04;
constexpr TPCANMessageType PCAN_MESSAGE_BRS = 0x08;
constexpr TPCANMessageType PCAN_MESSAGE_ESI = 0x10;
constexpr TPCANMessageType PCAN_MESSAGE_ECHO = 0x20;
constexpr TPCANMessageType PCAN_MESSAGE_ERRFRAME = 0x40;
constexpr TPCANMessageType PCAN_MESSAGE_STATUS = 0x80;

constexpr TPCANParameter PCAN_DEVICE_ID = 0x01;
constexpr TPCANParameter PCAN_LISTEN_ONLY = 0x08;
constexpr TPCANParameter PCAN_CHANNEL_CONDITION = 0x0D;
constexpr TPCANParameter PCAN_HARDWARE_NAME = 0x0E;
constexpr TPCANParameter PCAN_CHANNEL_FEATURES = 0x16;

constexpr std::uint32_t PCAN_PARAMETER_OFF = 0x00;
constexpr std::uint32_t PCAN_PARAMETER_ON = 0x01;
constexpr std::uint32_t PCAN_CHANNEL_UNAVAILABLE = 0x00;
constexpr std::uint32_t PCAN_CHANNEL_AVAILABLE = 0x01;
constexpr std::uint32_t PCAN_CHANNEL_OCCUPIED = 0x02;
constexpr std::uint32_t FEATURE_FD_CAPABLE = 0x01;
constexpr std::uint32_t MAX_LENGTH_HARDWARE_NAME = 33;

constexpr TPCANBaudrate PCAN_BAUD_1M = 0x0014;
constexpr TPCANBaudrate PCAN_BAUD_800K = 0x0016;
constexpr TPCANBaudrate PCAN_BAUD_500K = 0x001C;
constexpr TPCANBaudrate PCAN_BAUD_250K = 0x011C;
constexpr TPCANBaudrate PCAN_BAUD_125K = 0x031C;
constexpr TPCANBaudrate PCAN_BAUD_100K = 0x432F;
constexpr TPCANBaudrate PCAN_BAUD_95K = 0xC34E;
constexpr TPCANBaudrate PCAN_BAUD_83K = 0x852B;
constexpr TPCANBaudrate PCAN_BAUD_50K = 0x472F;
constexpr TPCANBaudrate PCAN_BAUD_47K = 0x1414;
constexpr TPCANBaudrate PCAN_BAUD_33K = 0x8B2F;
constexpr TPCANBaudrate PCAN_BAUD_20K = 0x532F;
constexpr TPCANBaudrate PCAN_BAUD_10K = 0x672F;
constexpr TPCANBaudrate PCAN_BAUD_5K = 0x7F7F;

/// @brief the language code GetErrorText takes for English.
constexpr std::uint16_t LANGUAGE_ENGLISH = 0x09;

using CAN_Initialize_t = TPCANStatus(PCAN_API *)(
    TPCANHandle Channel,
    TPCANBaudrate Btr0Btr1,
    TPCANType HwType,
    std::uint32_t IOPort,
    std::uint16_t Interrupt
);
using CAN_InitializeFD_t =
    TPCANStatus(PCAN_API *)(TPCANHandle Channel, TPCANBitrateFD BitrateFD);
using CAN_Uninitialize_t = TPCANStatus(PCAN_API *)(TPCANHandle Channel);
using CAN_Read_t = TPCANStatus(PCAN_API *)(
    TPCANHandle Channel,
    TPCANMsg *MessageBuffer,
    TPCANTimestamp *TimestampBuffer
);
using CAN_ReadFD_t = TPCANStatus(PCAN_API *)(
    TPCANHandle Channel,
    TPCANMsgFD *MessageBuffer,
    TPCANTimestampFD *TimestampBuffer
);
using CAN_Write_t =
    TPCANStatus(PCAN_API *)(TPCANHandle Channel, TPCANMsg *MessageBuffer);
using CAN_WriteFD_t =
    TPCANStatus(PCAN_API *)(TPCANHandle Channel, TPCANMsgFD *MessageBuffer);
using CAN_GetValue_t = TPCANStatus(PCAN_API *)(
    TPCANHandle Channel,
    TPCANParameter Parameter,
    void *Buffer,
    std::uint32_t BufferLength
);
using CAN_SetValue_t = TPCANStatus(PCAN_API *)(
    TPCANHandle Channel,
    TPCANParameter Parameter,
    void *Buffer,
    std::uint32_t BufferLength
);
using CAN_GetErrorText_t =
    TPCANStatus(PCAN_API *)(TPCANStatus Error, std::uint16_t Language, char *Buffer);
}
