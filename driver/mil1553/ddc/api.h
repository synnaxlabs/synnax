// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <array>
#include <cstdint>
#include <string>

#include "driver/errors/errors.h"

/// @brief the function declarations of the DDC AceXtreme MIL-STD-1553 SDK, from its
/// software reference manual (BU-69092SX Rev G-3/16).
namespace driver::mil1553::ddc::sdk {
using aceInitialize = std::int16_t (*)(
    std::int16_t dev,
    std::uint16_t access,
    std::uint16_t mode,
    std::uint32_t mem_words,
    std::uint32_t reg_addr,
    std::uint32_t mem_addr
);
using aceFree = std::int16_t (*)(std::int16_t dev);
using aceErrorStr =
    std::int16_t (*)(std::int16_t error, char *buffer, std::uint16_t size);
using aceBCConfigure = std::int16_t (*)(std::int16_t dev, std::uint32_t options);
using aceBCStart =
    std::int16_t (*)(std::int16_t dev, std::int16_t frame, std::int32_t count);
using aceBCStop = std::int16_t (*)(std::int16_t dev);
using aceRTSetAddress = std::int16_t (*)(std::int16_t dev, std::uint16_t address);
using aceRTStart = std::int16_t (*)(std::int16_t dev);
using aceRTStop = std::int16_t (*)(std::int16_t dev);
using aceMTStart = std::int16_t (*)(std::int16_t dev);
using aceMTStop = std::int16_t (*)(std::int16_t dev);

/// @brief the functions the loader checks for.
constexpr std::array<const char *, 11> FUNCTIONS = {
    "aceInitialize",
    "aceFree",
    "aceErrorStr",
    "aceBCConfigure",
    "aceBCStart",
    "aceBCStop",
    "aceRTSetAddress",
    "aceRTStart",
    "aceRTStop",
    "aceMTStart",
    "aceMTStop",
};

// TODO: declare the values of ACE_ACCESS_CARD, ACE_MODE_BC, ACE_MODE_RT,
// ACE_MODE_MRT, and ACE_MODE_MT, the MSGSTRUCT layout, and the message, data block,
// and frame functions (aceBCAsyncMsgCreateBCtoRT, aceBCDataBlkRead, aceBCFrameCreate,
// aceRTDataBlkCreate, and the monitor stack reads). The manual names them but does
// not give the constants or the struct layout.
}

namespace driver::mil1553::ddc {
// TODO: confirm both library names against the AceXtreme SDK.
#ifdef _WIN32
const std::string LIBRARY_NAME = "emacepl.dll";
#else
const std::string LIBRARY_NAME = "libemacepl.so";
#endif

const LibraryInfo LIBRARY_INFO = {"DDC AceXtreme MIL-STD-1553 SDK", ""};
}
