// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>

#include "driver/ethercat/coe/coe.h"

namespace driver::ethercat::coe {
namespace {
constexpr uint16_t SM_TYPE_INDEX = 0x1C00;
constexpr uint16_t PDO_ASSIGN_INDEX = 0x1C10;
constexpr uint32_t SM_TYPE_MAILBOX_IN = 2;
constexpr uint32_t SM_TYPE_OUTPUTS = 3;
constexpr uint32_t SM_TYPE_INPUTS = 4;
/// @brief sync managers 0 and 1 carry the mailbox.
constexpr uint32_t FIRST_PROCESS_DATA_SM = 2;
/// @brief the most PDOs an assign object, or objects a PDO, can count.
constexpr uint32_t MAX_SUB_INDEX = 254;
constexpr uint32_t ABORT_OBJECT_MISSING = 0x06020000;
}

x::errors::Error
read_error(const uint16_t index, const uint8_t sub_index, const uint32_t abort_code) {
    return x::errors::Error(
        abort_code == ABORT_OBJECT_MISSING ? OBJECT_MISSING : errors::SDO_READ_ERROR,
        "failed to read " + telem::format_index_sub_index(index, sub_index) + ": " +
            errors::sdo_abort_reason(abort_code)
    );
}

std::pair<std::vector<pdo::Sync>, x::errors::Error>
read_layout(const Read &read, const uint32_t max_syncs, const OnError on_error) {
    x::errors::Error first_err;
    const auto stop = [&](const x::errors::Error &err) {
        if (err && !first_err) first_err = err;
        return err && on_error == OnError::FAIL;
    };
    const auto read_count =
        [&](const uint16_t index,
            const uint32_t max) -> std::pair<uint32_t, x::errors::Error> {
        const auto [count, err] = read(index, 0);
        if (err) return {0, err};
        if (count <= max) return {count, x::errors::NIL};
        return {
            max,
            x::errors::Error(
                errors::PDO_DISCOVERY_ERROR,
                telem::format_index_sub_index(index, 0) + " counts " +
                    std::to_string(count) + " entries, above the maximum of " +
                    std::to_string(max)
            )
        };
    };
    const auto [sm_count, count_err] = read(SM_TYPE_INDEX, 0);
    if (count_err.matches(OBJECT_MISSING)) return {{}, x::errors::NIL};
    if (count_err) return {{}, count_err};
    std::vector<pdo::Sync> syncs;
    uint32_t type_shift = 0;
    // SOEM configures only the first max_syncs sync managers of a slave with more.
    for (uint32_t sm = FIRST_PROCESS_DATA_SM; sm < std::min(sm_count, max_syncs);
         sm++) {
        auto [type, type_err] = read(SM_TYPE_INDEX, static_cast<uint8_t>(sm + 1));
        if (stop(type_err)) return {{}, type_err};
        if (type_err) continue;
        // SOEM reads the types that some slaves get wrong this way, and maps them so.
        if (sm == 2 && type == SM_TYPE_MAILBOX_IN) type_shift = 1;
        if (type != 0) type += type_shift;
        if (type == 0 && sm == 2) type = SM_TYPE_OUTPUTS;
        if (type == 0 && sm == 3) type = SM_TYPE_INPUTS;
        if (type != SM_TYPE_OUTPUTS && type != SM_TYPE_INPUTS) continue;
        const auto assign_index = static_cast<uint16_t>(PDO_ASSIGN_INDEX + sm);
        const auto [pdo_count, pdo_count_err] = read_count(assign_index, MAX_SUB_INDEX);
        const bool fixed = pdo_count_err.matches(OBJECT_MISSING);
        if (!fixed && stop(pdo_count_err)) return {{}, pdo_count_err};
        auto &sync = syncs.emplace_back(
            pdo::Sync{
                .index = static_cast<uint8_t>(sm),
                .is_input = type == SM_TYPE_INPUTS,
                .pdos = {},
                .fixed = fixed,
            }
        );
        for (uint32_t i = 1; i <= pdo_count; i++) {
            const auto [pdo_index, pdo_err] = read(
                assign_index,
                static_cast<uint8_t>(i)
            );
            if (stop(pdo_err)) return {{}, pdo_err};
            if (pdo_err || pdo_index == 0) continue;
            auto &pdo = sync.pdos.emplace_back(
                pdo::PDO{.index = static_cast<uint16_t>(pdo_index), .entries = {}}
            );
            const auto [entry_count, entry_count_err] = read_count(
                pdo.index,
                MAX_SUB_INDEX
            );
            if (stop(entry_count_err)) return {{}, entry_count_err};
            for (uint32_t j = 1; j <= entry_count; j++) {
                const auto [mapping, mapping_err] = read(
                    pdo.index,
                    static_cast<uint8_t>(j)
                );
                if (stop(mapping_err)) return {{}, mapping_err};
                if (mapping_err) continue;
                pdo.entries.push_back({
                    .index = static_cast<uint16_t>(mapping >> 16),
                    .sub_index = static_cast<uint8_t>(mapping >> 8),
                    .bit_length = static_cast<uint8_t>(mapping),
                });
            }
        }
    }
    return {syncs, first_err};
}
}
