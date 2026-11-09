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
#include <utility>

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"

#include "driver/ethercat/coe/coe.h"

namespace driver::ethercat::esi {
/// @brief returns the settings that the catalog describes for a product: the card-wide
/// settings under "device", and under "channels" the settings of each input with the
/// entries that read it in every PDO layout.
/// Reads the option lists and limits that the catalog takes from the card through
/// read. Returns nullopt when the catalog describes no settings for the product.
/// @returns the error of a failed read, or errors::SDO_READ_ERROR when the card counts
/// more options than its table can hold.
std::pair<std::optional<x::json::json>, x::errors::Error>
read_settings(uint32_t vendor_id, uint32_t product_code, const coe::Read &read);
}
