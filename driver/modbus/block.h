// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <cstddef>
#include <cstdint>
#include <string>
#include <vector>

#include "x/cpp/errors/errors.h"

#include "driver/errors/errors.h"

namespace driver::modbus::block {
/// @brief a contiguous range of addresses read or written in one request.
struct Block {
    /// @brief the first address in the block.
    std::uint16_t start = 0;
    /// @brief the number of registers or bits in the block.
    std::size_t count = 0;
    /// @brief the index of the first channel in the block.
    std::size_t first = 0;
    /// @brief one past the index of the last channel in the block.
    std::size_t last = 0;
};

/// @brief groups channels into blocks of at most limit registers or bits.
/// @param channels channels sorted by address, each with an address and a width.
/// @param limit the most registers or bits one request can hold.
/// @param max_gap the most addresses that no channel covers between two channels of a
/// block. A block with a max_gap of 0 holds only channels that touch end to end.
/// @returns the blocks in address order. Each channel is in exactly one block.
template<typename Channel>
std::vector<Block> plan(
    const std::vector<Channel> &channels,
    const std::size_t limit,
    const std::size_t max_gap
) {
    std::vector<Block> blocks;
    std::size_t end = 0;
    for (std::size_t i = 0; i < channels.size(); i++) {
        const auto &ch = channels[i];
        const std::size_t ch_end = ch.address + ch.width();
        if (!blocks.empty()) {
            auto &b = blocks.back();
            const bool reachable = ch.address <= end + max_gap;
            const auto merged_end = std::max(end, ch_end);
            if (reachable && merged_end - b.start <= limit) {
                end = merged_end;
                b.count = end - b.start;
                b.last = i + 1;
                continue;
            }
        }
        end = ch_end;
        blocks.push_back({ch.address, ch.width(), i, i + 1});
    }
    return blocks;
}

/// @returns the address range of a block as "start" or "start-end".
inline std::string addresses(const Block &b) {
    const std::size_t end = b.start + b.count - 1;
    if (end == b.start) return std::to_string(b.start);
    return std::to_string(b.start) + "-" + std::to_string(end);
}

/// @brief names the channels and addresses of a failed request in its error.
/// @param err the error the request returned.
/// @param table the plural name of the table, such as "holding registers".
/// @param b the block the request covered.
/// @param names the names of the channels in the block.
inline x::errors::Error wrap(
    const x::errors::Error &err,
    const std::string &table,
    const Block &b,
    const std::vector<std::string> &names
) {
    std::string joined;
    for (const auto &name: names) {
        if (!joined.empty()) joined += ", ";
        joined += name;
    }
    return errors::wrap_channel_error(err, joined, table + " " + addresses(b));
}
}
