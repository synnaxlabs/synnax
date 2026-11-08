// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cmath>
#include <format>
#include <limits>

#include "x/cpp/telem/telem.h"

#include "driver/ethercat/channel/channel.h"
#include "driver/ethercat/telem/telem.h"

namespace driver::ethercat::channel {
namespace {
std::string direction(const bool is_input) {
    return is_input ? "input" : "output";
}

std::string bytes(const size_t n) {
    return std::to_string(n) + (n == 1 ? " byte" : " bytes");
}

/// @brief returns the entry at an address on slave, with the slave's bit length and
/// data_type. Accumulates a field error under path and returns an entry with index 0
/// when the slave lacks the entry or data_type does not fit it.
pdo::Entry bind(
    const bool is_input,
    const uint16_t index,
    const uint8_t sub_index,
    const x::telem::DataType &data_type,
    const slave::Properties &slave,
    const x::json::Parser &parser,
    const std::string &path
) {
    pdo::Entry unbound{.slave_position = slave.position, .is_input = is_input};
    const auto pdo = slave.find_pdo(index, sub_index, is_input);
    if (!pdo) {
        parser.field_err(
            path + "index",
            "slave " + slave.name + " has no " + direction(is_input) + " entry at " +
                telem::format_index_sub_index(index, sub_index)
        );
        return unbound;
    }
    static const std::vector NUMBER_TYPES = {
        x::telem::FLOAT64_T,
        x::telem::FLOAT32_T,
        x::telem::INT64_T,
        x::telem::INT32_T,
        x::telem::INT16_T,
        x::telem::INT8_T,
        x::telem::UINT64_T,
        x::telem::UINT32_T,
        x::telem::UINT16_T,
        x::telem::UINT8_T,
    };
    const auto entry = "entry " + telem::format_index_sub_index(index, sub_index) +
                       " on slave " + slave.name;
    const auto width = telem::infer_type_from_bit_length(pdo->bit_length).density();
    const auto pick = "Pick a " + std::to_string(width) + " byte ";
    if (!data_type.matches(NUMBER_TYPES)) {
        parser.field_err(
            path + "data_type",
            "data type " + data_type.name() + " cannot be exchanged with " + entry +
                ". " + pick + "number type."
        );
        return unbound;
    }
    if (data_type.density() != width) {
        parser.field_err(
            path + "data_type",
            "data type " + data_type.name() + " is " + bytes(data_type.density()) +
                " but " + entry + " holds " + bytes(pdo->byte_length()) + ". " + pick +
                "type."
        );
        return unbound;
    }
    // Only the 24-bit write turns a float type into an integer of a narrower entry.
    const auto bits = data_type.density() * 8;
    if (!is_input && data_type.matches({x::telem::FLOAT32_T, x::telem::FLOAT64_T}) &&
        pdo->bit_length < bits && pdo->bit_length != 24) {
        parser.field_err(
            path + "data_type",
            "data type " + data_type.name() + " needs a " + std::to_string(bits) +
                " bit entry, but " + entry + " holds " +
                std::to_string(pdo->bit_length) + " bits. Pick an integer type."
        );
        return unbound;
    }
    return {
        .slave_position = slave.position,
        .index = index,
        .sub_index = sub_index,
        .bit_length = pdo->bit_length,
        .is_input = is_input,
        .data_type = data_type,
    };
}

/// @brief accumulates a field error under path for an entry name that matches no single
/// entry, listing the addresses that carry it. Returns an entry with index 0.
pdo::Entry reject_unresolved(
    const bool is_input,
    const std::string &name,
    const slave::Properties &slave,
    const x::json::Parser &parser,
    const std::string &path
) {
    std::vector<std::string> addresses;
    std::vector<std::string> labels;
    for (const auto &pdo: is_input ? slave.input_pdos : slave.output_pdos) {
        if (pdo.is_padding() || pdo.name != name) continue;
        auto addr = telem::format_index_sub_index(pdo.index, pdo.sub_index);
        if (std::ranges::find(addresses, addr) != addresses.end()) continue;
        std::string label = addr;
        if (pdo.pdo_name && !pdo.pdo_name->empty()) label += " (" + *pdo.pdo_name + ")";
        addresses.push_back(std::move(addr));
        labels.push_back(std::move(label));
    }
    const auto on_slave = " on slave " + slave.name;
    const auto dir = direction(is_input);
    std::string msg = "'" + name + "' matches ";
    if (labels.empty())
        msg += "no " + dir + " entry" + on_slave;
    else if (labels.size() == 1)
        msg += dir + " entry " + labels.front() + on_slave;
    else {
        msg += "more than one " + dir + " entry" + on_slave + ": ";
        for (size_t i = 0; i < labels.size(); i++)
            msg += (i == 0 ? "" : ", ") + labels[i];
    }
    parser.field_err(path + "pdo", msg + ". Pick an entry.");
    return {.slave_position = slave.position, .is_input = is_input};
}

/// @brief accumulates a field error under path when stop_value does not fit entry.
void check_stop_value(
    const pdo::Entry &entry,
    const double v,
    const slave::Properties &slave,
    const x::json::Parser &parser,
    const std::string &path
) {
    const auto reject = [&](const std::string &range) {
        parser.field_err(
            path + "stop_value",
            std::format(
                "stop value {} does not fit entry {} on slave {}. Pick {}.",
                v,
                telem::format_index_sub_index(entry.index, entry.sub_index),
                slave.name,
                range
            )
        );
    };
    const bool is_float = entry.data_type.matches(
        {x::telem::FLOAT32_T, x::telem::FLOAT64_T}
    );
    // A float32 type on a 24-bit entry holds an unsigned integer.
    const bool narrow_float = is_float &&
                              entry.bit_length < entry.data_type.density() * 8;
    if (!narrow_float && entry.data_type == x::telem::FLOAT32_T) {
        constexpr float MAX = std::numeric_limits<float>::max();
        if (std::abs(v) > MAX) reject(std::format("a value from {} to {}", -MAX, MAX));
        return;
    }
    static const std::vector SIGNED_TYPES = {
        x::telem::INT64_T,
        x::telem::INT32_T,
        x::telem::INT16_T,
        x::telem::INT8_T,
    };
    static const std::vector UNSIGNED_TYPES = {
        x::telem::UINT64_T,
        x::telem::UINT32_T,
        x::telem::UINT16_T,
        x::telem::UINT8_T,
    };
    if (!narrow_float && !entry.data_type.matches(SIGNED_TYPES) &&
        !entry.data_type.matches(UNSIGNED_TYPES))
        return;
    // An entry under 8 bits reads back unsigned, whatever its data type.
    const bool is_signed = entry.bit_length >= 8 &&
                           entry.data_type.matches(SIGNED_TYPES);
    // A double rounds whole numbers of magnitude 2^53 and above, so a larger stop value
    // may not be the one the user entered.
    constexpr int DIGITS = std::numeric_limits<double>::digits;
    constexpr int64_t EXACT = (int64_t{1} << DIGITS) - 1;
    const int bits = is_signed ? entry.bit_length - 1 : entry.bit_length;
    const int64_t max = bits >= DIGITS ? EXACT : (int64_t{1} << bits) - 1;
    const int64_t min = is_signed ? std::max(-EXACT, -max - 1) : 0;
    if (v < static_cast<double>(min) || v > static_cast<double>(max) ||
        v != std::trunc(v))
        reject(std::format("a whole number from {} to {}", min, max));
}
}

std::unique_ptr<Input> make_input(
    const ::synnax::ethercat::ReadChannel &cfg,
    const slave::Properties &slave,
    const x::json::Parser &parser,
    const std::string &path
) {
    if (::synnax::ethercat::base(cfg).disabled) return nullptr;
    if (const auto *c = std::get_if<::synnax::ethercat::UnresolvedReadChannel>(&cfg))
        return std::make_unique<Input>(Input{
            .entry = reject_unresolved(true, c->pdo, slave, parser, path),
            .synnax_key = c->channel,
        });
    const auto &c = std::get<::synnax::ethercat::EntryReadChannel>(cfg);
    return std::make_unique<Input>(Input{
        .entry = bind(true, c.index, c.sub_index, c.data_type, slave, parser, path),
        .synnax_key = c.channel,
        .scale = c.scale,
    });
}

std::unique_ptr<Output> make_output(
    const ::synnax::ethercat::WriteChannel &cfg,
    const slave::Properties &slave,
    const x::json::Parser &parser,
    const std::string &path
) {
    if (::synnax::ethercat::base(cfg).disabled) return nullptr;
    if (const auto *c = std::get_if<::synnax::ethercat::UnresolvedWriteChannel>(&cfg))
        return std::make_unique<Output>(Output{
            .entry = reject_unresolved(false, c->pdo, slave, parser, path),
            .command_key = c->cmd_channel,
            .state_key = c->state_channel,
        });
    const auto &c = std::get<::synnax::ethercat::EntryWriteChannel>(cfg);
    auto out = std::make_unique<Output>(Output{
        .entry = bind(false, c.index, c.sub_index, c.data_type, slave, parser, path),
        .command_key = c.cmd_channel,
        .state_key = c.state_channel,
        .stop_value = c.stop_value,
    });
    check_stop_value(out->entry, out->stop_value, slave, parser, path);
    return out;
}

void Claims::claim(
    const pdo::Entry &entry,
    const slave::Properties &slave,
    const x::json::Parser &parser,
    const std::string &path
) {
    if (entry.index == 0) return;
    const pdo::Key key{
        entry.slave_position,
        entry.index,
        entry.sub_index,
        entry.is_input
    };
    if (this->keys.insert(key).second) return;
    parser.field_err(
        path + "index",
        "another channel already uses entry " +
            telem::format_index_sub_index(entry.index, entry.sub_index) + " on slave " +
            slave.name
    );
}
}
