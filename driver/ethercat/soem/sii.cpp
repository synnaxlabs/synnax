// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/ethercat/soem/sii.h"

namespace driver::ethercat::soem::sii {
namespace {
constexpr size_t PDO_HEADER_SIZE = 8;
constexpr size_t PDO_ENTRY_SIZE = 8;
constexpr size_t OP_MODE_SIZE = 24;

/// @brief reads a little-endian value of n bytes from data at offset.
uint32_t read(const std::span<const uint8_t> data, const size_t offset, const int n) {
    uint32_t value = 0;
    for (int i = 0; i < n; ++i)
        value |= static_cast<uint32_t>(data[offset + i]) << (8 * i);
    return value;
}
}

std::vector<pdo::PDO> parse_pdos(
    const std::span<const uint8_t> data,
    const std::function<bool(uint8_t sync)> &keep,
    const Strings &strings
) {
    std::vector<pdo::PDO> pdos;
    size_t off = 0;
    while (off + PDO_HEADER_SIZE <= data.size()) {
        const auto count = data[off + 2];
        const auto sync_manager = data[off + 3];
        const auto end = off + PDO_HEADER_SIZE + count * PDO_ENTRY_SIZE;
        if (end > data.size()) break;
        pdo::PDO pdo{.index = static_cast<uint16_t>(read(data, off, 2)), .entries = {}};
        if (auto name = strings(data[off + 5]); !name.empty())
            pdo.name = std::move(name);
        for (size_t e = off + PDO_HEADER_SIZE; e < end; e += PDO_ENTRY_SIZE)
            pdo.entries.push_back({
                .index = static_cast<uint16_t>(read(data, e, 2)),
                .sub_index = data[e + 2],
                .bit_length = data[e + 5],
                .name = strings(data[e + 3]),
                .type = static_cast<telem::DataType>(data[e + 4]),
            });
        if (keep(sync_manager)) pdos.push_back(std::move(pdo));
        off = end;
    }
    return pdos;
}

std::vector<dc::OpMode>
parse_op_modes(const std::span<const uint8_t> data, const Strings &strings) {
    std::vector<dc::OpMode> modes;
    for (size_t off = 0; off + OP_MODE_SIZE <= data.size(); off += OP_MODE_SIZE)
        modes.push_back({
            .name = strings(data[off + 18]),
            .assign_activate = static_cast<uint16_t>(read(data, off + 14, 2)),
            .sync0_cycle = read(data, off, 4),
            .sync0_factor = static_cast<int16_t>(read(data, off + 16, 2)),
            .sync0_shift = static_cast<int32_t>(read(data, off + 4, 4)),
            .sync1_factor = static_cast<int16_t>(read(data, off + 12, 2)),
            .sync1_shift = static_cast<int32_t>(read(data, off + 8, 4)),
        });
    return modes;
}
}
