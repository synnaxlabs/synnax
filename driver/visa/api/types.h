// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>

/// @brief VISA type definitions (defined manually to avoid including visa.h)
/// These match the VISA specification (IVI Foundation).
namespace visa_types {

// Basic types
using ViStatus = int32_t;
using ViSession = uint32_t;
using ViUInt32 = uint32_t;
using ViChar = char;
using ViRsrc = char *;
using ViBuf = uint8_t *;
using ViString = char *;

// Find list handle
using ViFindList = uint32_t;

// Constants - Error codes
constexpr ViStatus VI_SUCCESS = 0;
constexpr ViStatus VI_ERROR_RSRC_NFOUND = static_cast<ViStatus>(0xBFFF0011L);
constexpr ViStatus VI_ERROR_TMO = static_cast<ViStatus>(0xBFFF0015L);
constexpr ViStatus VI_ERROR_CLOSING_FAILED = static_cast<ViStatus>(0xBFFF0016L);
constexpr ViStatus VI_ERROR_INV_OBJECT = static_cast<ViStatus>(0xBFFF000EL);
constexpr ViStatus VI_ERROR_NSUP_OPER = static_cast<ViStatus>(0xBFFF0067L);
constexpr ViStatus VI_ERROR_CONN_LOST = static_cast<ViStatus>(0xBFFF003FL);
constexpr ViStatus VI_ERROR_IO = static_cast<ViStatus>(0xBFFF0036L);

// Constants - Other
constexpr ViSession VI_NULL = 0;
constexpr ViUInt32 VI_FIND_BUFLEN = 256;

}
