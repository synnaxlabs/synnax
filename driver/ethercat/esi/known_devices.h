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
#include <optional>
#include <string_view>

#include "driver/ethercat/slave/slave.h"

namespace driver::ethercat::esi {
/// @brief looks up PDO definitions for a device in the ESI registry.
bool lookup_device_pdos(
    uint32_t vendor_id,
    uint32_t product_code,
    uint32_t revision,
    slave::Properties &slave
);

/// @brief returns the vendor name for a given vendor ID.
std::optional<std::string_view> vendor_name(uint32_t vendor_id);

/// @brief checks if a device is in the registry (any revision).
bool is_device_known(uint32_t vendor_id, uint32_t product_code);

}
