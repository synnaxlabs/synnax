// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <map>
#include <memory>

#include "driver/ethercat/coe/coe.h"

namespace driver::ethercat::coe {
namespace {
constexpr uint16_t SM_TYPE_INDEX = 0x1C00;
constexpr uint16_t PDO_ASSIGN_INDEX = 0x1C10;
/// @brief the first index past the PDO assign objects.
constexpr uint16_t PDO_ASSIGN_END = 0x1C30;
constexpr uint16_t RXPDO_MAPPING_INDEX = 0x1600;
constexpr uint16_t RXPDO_MAPPING_END = 0x1800;
constexpr uint16_t TXPDO_MAPPING_INDEX = 0x1A00;
constexpr uint16_t TXPDO_MAPPING_END = 0x1C00;
/// @brief complete access sends sub-index 0 as 16 bits.
constexpr size_t COMPLETE_ACCESS_HEADER = 2;
constexpr uint32_t SM_TYPE_MAILBOX_IN = 2;
constexpr uint32_t SM_TYPE_OUTPUTS = 3;
constexpr uint32_t SM_TYPE_INPUTS = 4;
/// @brief sync managers 0 and 1 carry the mailbox.
constexpr uint32_t FIRST_PROCESS_DATA_SM = 2;
/// @brief the most PDOs an assign object, or objects a PDO, can count.
constexpr uint32_t MAX_SUB_INDEX = 254;
constexpr uint32_t ABORT_OBJECT_MISSING = 0x06020000;
constexpr uint32_t ABORT_SUBINDEX_MISSING = 0x06090011;

/// @brief returns the size in bytes of each entry of the sync manager type object, a
/// PDO assign object, or a PDO mapping object, or 0 for another object.
size_t entry_size(const uint16_t index) {
    if (index == SM_TYPE_INDEX) return 1;
    if (index >= PDO_ASSIGN_INDEX && index < PDO_ASSIGN_END) return 2;
    if ((index >= RXPDO_MAPPING_INDEX && index < RXPDO_MAPPING_END) ||
        (index >= TXPDO_MAPPING_INDEX && index < TXPDO_MAPPING_END))
        return 4;
    return 0;
}

/// @brief returns the entries of an object read with complete access, or nullopt when
/// bytes are too short for the count in sub-index 0.
std::optional<std::vector<uint32_t>>
parse_entries(const std::vector<uint8_t> &bytes, const size_t size) {
    if (bytes.size() < COMPLETE_ACCESS_HEADER) return std::nullopt;
    const size_t count = bytes[0];
    if (bytes.size() < COMPLETE_ACCESS_HEADER + (count * size)) return std::nullopt;
    std::vector<uint32_t> entries(count);
    for (size_t i = 0; i < count; i++) {
        const size_t start = COMPLETE_ACCESS_HEADER + (i * size);
        for (size_t b = 0; b < size; b++)
            entries[i] |= static_cast<uint32_t>(bytes[start + b]) << (8 * b);
    }
    return entries;
}

const x::errors::Error &failure_type(const errors::SDOFailure &failure) {
    using Kind = errors::SDOFailure::Kind;
    if (failure.kind == Kind::NO_REPLY) return NO_REPLY;
    if (failure.kind != Kind::ABORTED) return errors::SDO_READ_ERROR;
    if (failure.abort_code == ABORT_OBJECT_MISSING) return OBJECT_MISSING;
    if (failure.abort_code == ABORT_SUBINDEX_MISSING) return SUBINDEX_MISSING;
    return errors::SDO_READ_ERROR;
}
}

Read with_complete_access(Read read, ReadAll read_all) {
    auto objects = std::make_shared<
        std::map<uint16_t, std::optional<std::vector<uint32_t>>>>();
    return [read = std::move(read), read_all = std::move(read_all), objects](
               const uint16_t index,
               const uint8_t sub_index
           ) -> std::pair<uint32_t, x::errors::Error> {
        const size_t size = entry_size(index);
        if (size == 0) return read(index, sub_index);
        auto [it, added] = objects->try_emplace(index);
        if (added)
            if (const auto bytes = read_all(index))
                it->second = parse_entries(*bytes, size);
        if (const auto &entries = it->second; entries && sub_index <= entries->size())
            return {
                sub_index == 0 ? static_cast<uint32_t>(entries->size())
                               : (*entries)[sub_index - 1],
                x::errors::NIL
            };
        return read(index, sub_index);
    };
}

x::errors::Error read_error(
    const uint16_t index,
    const uint8_t sub_index,
    const errors::SDOFailure &failure
) {
    return x::errors::Error(
        failure_type(failure),
        "failed to read " + telem::format_index_sub_index(index, sub_index) + ": " +
            failure.reason()
    );
}

x::errors::Error
read_error(const uint16_t index, const uint8_t sub_index, const uint32_t abort_code) {
    return read_error(
        index,
        sub_index,
        errors::SDOFailure::from_abort_code(abort_code)
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
