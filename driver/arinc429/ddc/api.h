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
#include <memory>
#include <string>
#include <utility>

#include "x/cpp/errors/errors.h"
#include "x/cpp/lib/lib.h"

#include "driver/errors/errors.h"

/// @brief the function declarations of the DDC DD-42992 ARINC 429 SDK, from its
/// software manual (Rev P-9/18). On 64-bit Windows the calling convention does not
/// matter.
namespace driver::arinc429::ddc::sdk {
/// @brief returned by every call that succeeds. Errors are negative.
constexpr short SUCCESS = 0;
/// @brief DD429_ENABLE.
constexpr short ENABLE = 1;
/// @brief DD429_DISABLE.
constexpr short DISABLE = 0;
/// @brief DD429_LOW_SPEED.
constexpr short LOW_SPEED = 0;
/// @brief DD429_HIGH_SPEED.
constexpr short HIGH_SPEED = 1;
/// @brief DD429_ODD_PARITY.
constexpr short ODD_PARITY = 1;
/// @brief DD429_BITFORMAT_ORIG: label in bits 0 to 7, SDI in 8 and 9, data in 10 to
/// 28, SSM in 29 and 30, and parity in 31. On receive, bit 31 is 1 when the word
/// arrived with even parity.
constexpr short BITFORMAT_ORIG = 0;
/// @brief the most words LoadTxQueueMore takes in one call.
constexpr short MAX_LOAD = 256;
/// @brief the shortest buffer GetErrorMsg writes into.
constexpr std::size_t ERROR_MESSAGE_SIZE = 80;

/// @brief CHANCOUNT_t: the number of channels of each type on a card.
struct ChannelCount {
    std::uint8_t tx;
    std::uint8_t rx;
    std::uint8_t group;
    std::uint8_t discrete;
    std::uint8_t avionic;
    std::uint8_t board_model;
    std::uint8_t mil1553;
    std::uint8_t arinc429_prog;
    std::uint8_t uart;
    std::uint8_t rs232;
    std::uint8_t rs485;
    std::uint8_t can_bus;
    std::uint8_t arinc717_rx;
    std::uint8_t arinc717_tx;
    std::uint8_t arinc717_prog;
};

using InitCard = short (*)(short card);
using FreeCard = short (*)(short card);
using GetChannelCount = short (*)(short card, ChannelCount *count);
using GetErrorMsg = short (*)(short error, char message[]);
using EnableRx = short (*)(short card, short receiver, short enable);
using EnableTx = short (*)(short card, short transmitter, short enable);
using SetRxChannelSpeed = short (*)(short card, short receiver, short speed);
using SetTxSpeed = short (*)(short card, short transmitter, short speed);
using SetRxChannelParity = short (*)(short card, short receiver, short parity);
using SetTxParity = short (*)(short card, short transmitter, short parity);
using SetBitFormat = short (*)(short card, short format);
// TODO: confirm the width of unsigned long in the Linux SDK. It is 64 bits on LP64
// Linux, and the manual documents only 32-bit words.
/// @brief returns the number of words read, or an error.
using ReadRxQueueIrigMore = short (*)(
    short card,
    short receiver,
    short n,
    unsigned long *data,
    long *stamp_hi,
    long *stamp_lo
);
/// @brief returns the number of words loaded, or an error.
using LoadTxQueueMore =
    short (*)(short card, short transmitter, short n, unsigned long *data);
}

namespace driver::arinc429::ddc {
// TODO: confirm the Linux library name. The Linux SDK ships as source that the user
// builds, and the manual names only the Windows DLL.
#ifdef _WIN32
const std::string LIBRARY_NAME = "DD42992.dll";
#else
const std::string LIBRARY_NAME = "libdd42992.so";
#endif

const LibraryInfo LIBRARY_INFO = {"DDC DD-42992 ARINC 429 SDK", ""};

/// @brief API is the DD-42992 functions the backend calls, loaded from the SDK's
/// shared library.
class API {
public:
    /// @brief loads the SDK library and resolves every function.
    /// @returns the missing library error when the library or a function is absent.
    static std::pair<std::shared_ptr<API>, x::errors::Error> load();

    /// @returns an error describing a negative SDK return code, or NIL for any other.
    [[nodiscard]] x::errors::Error error(short code) const;

    sdk::InitCard init_card = nullptr;
    sdk::FreeCard free_card = nullptr;
    sdk::GetChannelCount get_channel_count = nullptr;
    sdk::GetErrorMsg get_error_msg = nullptr;
    sdk::EnableRx enable_rx = nullptr;
    sdk::EnableTx enable_tx = nullptr;
    sdk::SetRxChannelSpeed set_rx_channel_speed = nullptr;
    sdk::SetTxSpeed set_tx_speed = nullptr;
    sdk::SetRxChannelParity set_rx_channel_parity = nullptr;
    sdk::SetTxParity set_tx_parity = nullptr;
    sdk::SetBitFormat set_bit_format = nullptr;
    sdk::ReadRxQueueIrigMore read_rx_queue_irig_more = nullptr;
    sdk::LoadTxQueueMore load_tx_queue_more = nullptr;

private:
    std::unique_ptr<x::lib::Shared> lib;
};
}
