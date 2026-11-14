// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <array>
#include <format>
#include <string_view>
#include <utility>

#include "driver/ethercat/errors/errors.h"

namespace driver::ethercat::errors {
namespace {
// The SDO abort codes of CiA 301 and ETG.1000.6.
constexpr std::array<std::pair<uint32_t, std::string_view>, 32> ABORT_REASONS{{
    {0x05030000, "Toggle bit not changed"},
    {0x05040000, "SDO protocol timed out"},
    {0x05040001, "Client/server command specifier not valid or unknown"},
    {0x05040005, "Out of memory"},
    {0x06010000, "Unsupported access to an object"},
    {0x06010001, "Attempt to read a write-only object"},
    {0x06010002, "Attempt to write a read-only object"},
    {0x06010003, "Subindex cannot be written, SI0 must be 0 for write access"},
    {0x06010004, "SDO complete access not supported for variable length objects"},
    {0x06010005, "Object length exceeds mailbox size"},
    {0x06010006, "Object mapped to RxPDO, SDO download blocked"},
    {0x06020000, "Object does not exist in the object dictionary"},
    {0x06040041, "Object cannot be mapped to the PDO"},
    {0x06040042, "Number and length of the objects to be mapped exceed the PDO length"},
    {0x06040043, "General parameter incompatibility"},
    {0x06040047, "General internal incompatibility in the device"},
    {0x06060000, "Access failed due to a hardware error"},
    {0x06070010,
     "Data type does not match, length of service parameter does not match"},
    {0x06070012, "Data type does not match, length of service parameter too high"},
    {0x06070013, "Data type does not match, length of service parameter too low"},
    {0x06090011, "Subindex does not exist"},
    {0x06090030, "Value range of parameter exceeded"},
    {0x06090031, "Value of parameter written too high"},
    {0x06090032, "Value of parameter written too low"},
    {0x06090036, "Maximum value is less than minimum value"},
    {0x060A0023, "Resource not available"},
    {0x08000000, "General error"},
    {0x08000020, "Data cannot be transferred or stored to the application"},
    {0x08000021,
     "Data cannot be transferred or stored to the application because of local "
     "control"},
    {0x08000022,
     "Data cannot be transferred or stored to the application because of the present "
     "device state"},
    {0x08000023, "Object dictionary dynamic generation failed or no object dictionary"},
    {0x08000024, "No data available"},
}};

// Aborts about the transfer or the device state, not the object or its value.
constexpr std::array<uint32_t, 7> TEMPORARY_ABORTS{
    0x05030000,
    0x05040000,
    0x05040005,
    0x060A0023,
    0x08000020,
    0x08000021,
    0x08000022,
};
}

std::string sdo_abort_reason(const uint32_t code) {
    if (code == 0) return "no response";
    for (const auto &[known, reason]: ABORT_REASONS)
        if (known == code) return std::string(reason);
    return std::format("unknown abort code 0x{:08X}", code);
}

const x::errors::Error &startup_parameter_error(const uint32_t code) {
    if (code == 0 ||
        std::ranges::find(TEMPORARY_ABORTS, code) != TEMPORARY_ABORTS.end())
        return STARTUP_PARAMETER_ERROR;
    return STARTUP_PARAMETER_REJECTED;
}
}
