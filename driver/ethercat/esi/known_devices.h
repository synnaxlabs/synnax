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

#include "driver/ethercat/master/slave_info.h"

namespace ethercat::esi {

/// Looks up PDO definitions for a device in the ESI registry.
/// Uses binary search with revision fallback.
/// @param vendor_id EtherCAT vendor ID.
/// @param product_code Device product code.
/// @param revision Device revision (falls back to any revision if exact match not
/// found).
/// @param slave Output parameter populated with PDO information.
/// @returns true if device was found, false otherwise.
bool lookup_device_pdos(
    uint32_t vendor_id,
    uint32_t product_code,
    uint32_t revision,
    SlaveInfo &slave
);

/// Returns the vendor name for a given vendor ID.
/// @param vendor_id EtherCAT vendor ID.
/// @returns Vendor name if found, std::nullopt otherwise.
std::optional<std::string_view> vendor_name(uint32_t vendor_id);

/// Checks if a device is in the registry (any revision).
/// @param vendor_id EtherCAT vendor ID.
/// @param product_code Device product code.
/// @returns true if any revision of this device is known.
bool is_device_known(uint32_t vendor_id, uint32_t product_code);

}
