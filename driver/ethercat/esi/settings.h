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
#include <vector>

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"

#include "driver/ethercat/coe/coe.h"
#include "driver/ethercat/dc/dc.h"

namespace driver::ethercat::esi {
/// @brief returns the settings that the catalog describes for a product: the card-wide
/// settings under "device", and under "channels" the settings of each input with the
/// entries that read it in every PDO layout. Each setting of an input holds the value
/// of the card in "value". An entry whose format only one program reads names it in
/// "only_read_by". When the catalog describes the card's mode, "mode" holds the address
/// of the object that holds it ("index" and "sub_index"), the value that the card
/// holds, whether other software sets up the card ("shared"), the sync mode that the
/// value needs as a dc::Choice among op_modes ("op_mode"), and the startup parameters
/// that make the card standalone ("switch"). Reads the values, option lists, limits,
/// and mode from the card through read. Returns nullopt when the catalog describes no
/// settings for the product.
/// @returns the error of a failed read, or errors::SDO_READ_ERROR when the card counts
/// no options, or more than its table can hold. A failed read of the mode gives the
/// other settings with its error.
std::pair<std::optional<x::json::json>, x::errors::Error> read_settings(
    uint32_t vendor_id,
    uint32_t product_code,
    const std::vector<dc::OpMode> &op_modes,
    const coe::Read &read
);
}
