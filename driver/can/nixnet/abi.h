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

/// @brief the subset of the NI-XNET C ABI the backend calls, transcribed from NI's
/// NI-XNET C API reference.
namespace driver::can::nixnet {
using u8 = std::uint8_t;
using u32 = std::uint32_t;
using u64 = std::uint64_t;
using f64 = double;
using nxStatus_t = std::int32_t;
using nxSessionRef_t = std::uint32_t;

constexpr nxStatus_t nxSuccess = 0;

constexpr u32 nxMode_FrameInStream = 6;
constexpr u32 nxMode_FrameOutStream = 9;
constexpr u32 nxStartStop_Normal = 0;
constexpr u32 nxCANioMode_CAN_FD_BRS = 2;

constexpr u32 nxClass_Session = 0x00100000;
constexpr u32 nxClass_System = 0x00110000;
constexpr u32 nxClass_Device = 0x00120000;
constexpr u32 nxClass_Interface = 0x00130000;
constexpr u32 nxPrptype_u32 = 0x00000000;
constexpr u32 nxPrptype_bool = 0x02000000;
constexpr u32 nxPrptype_string = 0x03000000;
constexpr u32 nxPrptype_ref = 0x05000000;
constexpr u32 nxPrptype_1Dref = 0x06000000;
constexpr u32 nxPrptype_u64 = 0x09000000;

constexpr u32 nxPropSession_IntfBaudRate64 = 0x16 | nxClass_Session | nxPrptype_u64;
constexpr u32 nxPropSession_IntfCanLstnOnly = 0x22 | nxClass_Session | nxPrptype_bool;
constexpr u32 nxPropSession_IntfCanIoMode = 0x26 | nxClass_Session | nxPrptype_u32;
constexpr u32 nxPropSession_IntfCanFdBaudRate64 = 0x27 | nxClass_Session |
                                                  nxPrptype_u64;
constexpr u32 nxPropSys_IntfRefsCAN = 0x04 | nxClass_System | nxPrptype_1Dref;
constexpr u32 nxPropIntf_DevRef = 0x01 | nxClass_Interface | nxPrptype_ref;
constexpr u32 nxPropIntf_Name = 0x02 | nxClass_Interface | nxPrptype_string;
constexpr u32 nxPropDev_Name = 0x03 | nxClass_Device | nxPrptype_string;
constexpr u32 nxPropDev_SerNum = 0x05 | nxClass_Device | nxPrptype_u32;

constexpr u8 nxFrameType_CAN_Data = 0x00;
constexpr u8 nxFrameType_CAN_Remote = 0x01;
constexpr u8 nxFrameType_CAN_BusError = 0x02;
constexpr u8 nxFrameType_CAN20_Data = 0x08;
constexpr u8 nxFrameType_CANFD_Data = 0x10;
constexpr u8 nxFrameType_CANFDBRS_Data = 0x18;
constexpr u8 nxFrameFlags_TransmitEcho = 0x80;
constexpr u32 nxFrameId_CAN_IsExtended = 0x20000000;

/// @brief the offset between the NI-XNET epoch, 1601-01-01, and the Unix epoch in the
/// 100 ns ticks of an NI-XNET timestamp.
constexpr u64 nxTimestamp_UnixOffset = 116444736000000000ULL;

using nxCreateSession_t = nxStatus_t (*)(
    const char *DatabaseName,
    const char *ClusterName,
    const char *List,
    const char *Interface,
    u32 Mode,
    nxSessionRef_t *SessionRef
);
using nxClear_t = nxStatus_t (*)(nxSessionRef_t SessionRef);
using nxSetProperty_t = nxStatus_t (*)(
    nxSessionRef_t SessionRef,
    u32 PropertyID,
    u32 PropertySize,
    void *PropertyValue
);
using nxGetProperty_t = nxStatus_t (*)(
    nxSessionRef_t SessionRef,
    u32 PropertyID,
    u32 PropertySize,
    void *PropertyValue
);
using nxGetPropertySize_t =
    nxStatus_t (*)(nxSessionRef_t SessionRef, u32 PropertyID, u32 *PropertySize);
using nxReadFrame_t = nxStatus_t (*)(
    nxSessionRef_t SessionRef,
    void *Buffer,
    u32 SizeOfBuffer,
    f64 Timeout,
    u32 *NumberOfBytesReturned
);
using nxWriteFrame_t = nxStatus_t (*)(
    nxSessionRef_t SessionRef,
    void *Buffer,
    u32 NumberOfBytesForFrames,
    f64 Timeout
);
using nxStart_t = nxStatus_t (*)(nxSessionRef_t SessionRef, u32 Scope);
using nxStatusToString_t =
    void (*)(nxStatus_t Status, u32 SizeofString, char *StatusDescription);
using nxSystemOpen_t = nxStatus_t (*)(nxSessionRef_t *SystemRef);
using nxSystemClose_t = nxStatus_t (*)(nxSessionRef_t SystemRef);
}
