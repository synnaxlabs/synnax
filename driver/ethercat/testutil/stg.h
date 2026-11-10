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
#include <map>
#include <utility>

#include "x/cpp/errors/errors.h"

#include "driver/ethercat/coe/coe.h"

namespace driver::ethercat::testutil {
/// @brief the vendor ID of Dewesoft.
constexpr uint32_t DEWESOFT = 0xDEBE50F7;
/// @brief the product code of the IOLITE 6xSTG.
constexpr uint32_t IOLITE_6XSTG = 252;

/// @brief CoE objects keyed by index and sub-index.
using Objects = std::map<std::pair<uint16_t, uint8_t>, uint32_t>;

/// @brief returns the objects of the 6xSTG on the station when it holds mode in ECAT0
/// App (0x8020:09) and power_on in PowerONdefault (0x8020:02): those two, and the
/// range, filter, and excitation tables of its six inputs. Each input holds the tables
/// of input 1 at its own indices.
inline Objects stg_objects(const uint32_t mode, const uint32_t power_on = 1) {
    const Objects input_1{
        {{0x8050, 3}, 5},
        {{0x8050, 4}, 500},
        {{0x8050, 5}, 100},
        {{0x8050, 6}, 30},
        {{0x8050, 7}, 10},
        {{0x8050, 8}, 1},
        {{0x8060, 1}, 2},
        {{0x8060, 2}, 10000},
        {{0x8060, 3}, 500},
        {{0x8070, 2}, 240},
        {{0x8070, 3}, 42},
        {{0x8070, 4}, 120},
    };
    Objects objects{{{0x8020, 9}, mode}, {{0x8020, 2}, power_on}};
    for (uint16_t input = 0; input < 6; input++)
        for (const auto &[address, value]: input_1)
            objects[{
                static_cast<uint16_t>(address.first + input),
                address.second
            }] = value;
    return objects;
}

/// @brief returns a read of objects that aborts as a missing object does for an
/// address that objects does not hold.
inline coe::Read read_of(Objects objects) {
    return
        [objects = std::move(objects)](const uint16_t index, const uint8_t sub_index) {
            const auto it = objects.find({index, sub_index});
            if (it == objects.end())
                return std::pair{
                    uint32_t{0},
                    coe::read_error(index, sub_index, 0x06020000)
                };
            return std::pair{it->second, x::errors::NIL};
        };
}
}
