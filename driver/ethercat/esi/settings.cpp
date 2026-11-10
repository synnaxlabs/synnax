// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <bit>
#include <cstdint>
#include <format>
#include <string>

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/esi/settings.h"
#include "driver/ethercat/telem/telem.h"

namespace driver::ethercat::esi {
namespace {
#include "driver/ethercat/esi/registry_settings.inc"

/// @brief adds step to each index that a setting, rule, or condition names.
void shift(x::json::json &item, const uint16_t step) {
    if (item.contains("index")) item["index"] = item["index"].get<uint16_t>() + step;
    for (const auto *key: {"options_from", "max_from", "then"})
        if (item.contains(key)) shift(item[key], step);
    for (const auto *key: {"visible_when", "when"})
        if (item.contains(key))
            for (auto &cond: item[key])
                shift(cond, step);
}

/// @brief replaces the options_from and max_from of a setting with the options and
/// limit that the card holds.
x::errors::Error resolve(x::json::json &setting, const coe::Read &read) {
    if (setting.contains("options_from")) {
        const auto from = setting["options_from"];
        const auto index = from["index"].get<uint16_t>();
        const auto count_sub = from["count_sub_index"].get<uint8_t>();
        const auto first = from["first_sub_index"].get<uint8_t>();
        const auto [count, err] = read(index, count_sub);
        if (err) return err;
        if (count == 0 || count > 256u - first)
            return x::errors::Error(
                errors::SDO_READ_ERROR,
                std::format(
                    "{} counts {} options. The count must be from 1 to {}",
                    telem::format_index_sub_index(index, count_sub),
                    count,
                    256 - first
                )
            );
        const auto divisor = from["divisor"].get<double>();
        const auto unit = from["unit"].get<std::string>();
        auto options = x::json::json::array();
        for (uint32_t i = 0; i < count; i++) {
            const auto [raw, read_err] = read(index, static_cast<uint8_t>(first + i));
            if (read_err) return read_err;
            options.push_back({
                {"value", from["first_value"].get<double>() + i},
                {"name", std::format("{} {}", raw / divisor, unit)},
            });
        }
        setting.erase("options_from");
        setting["options"] = std::move(options);
    }
    if (setting.contains("max_from")) {
        const auto from = setting["max_from"];
        const auto [raw, err] = read(
            from["index"].get<uint16_t>(),
            from["sub_index"].get<uint8_t>()
        );
        if (err) return err;
        setting.erase("max_from");
        setting["max"] = raw / from["divisor"].get<double>();
    }
    return x::errors::NIL;
}

/// @brief returns the address of each entry that channel n reads, in every layout.
x::json::json entries_of(const x::json::json &layouts, const int n) {
    auto entries = x::json::json::array();
    for (const auto &layout: layouts) {
        const auto index = layout["index"].get<int>() +
                           n * layout["index_step"].get<int>();
        const auto first = layout["sub_index"].get<int>() +
                           n * layout["sub_index_step"].get<int>();
        for (int k = 0; k < layout["count"].get<int>(); k++)
            entries.push_back({{"index", index}, {"sub_index", first + k}});
    }
    return entries;
}

x::errors::Error resolve_all(x::json::json &settings, const coe::Read &read) {
    for (auto &setting: settings)
        if (auto err = resolve(setting, read)) return err;
    return x::errors::NIL;
}

/// @brief sets the "value" of each setting to the float32 or unsigned value that the
/// card holds.
x::errors::Error read_values(x::json::json &settings, const coe::Read &read) {
    for (auto &setting: settings) {
        const auto [raw, err] = read(
            setting["index"].get<uint16_t>(),
            setting["sub_index"].get<uint8_t>()
        );
        if (err) return err;
        if (setting["data_type"] == "float32")
            setting["value"] = std::bit_cast<float>(raw);
        else
            setting["value"] = raw;
    }
    return x::errors::NIL;
}
}

std::pair<std::optional<x::json::json>, x::errors::Error> read_settings(
    const uint32_t vendor_id,
    const uint32_t product_code,
    const coe::Read &read
) {
    const auto catalog = x::json::json::parse(
        reinterpret_cast<const char *>(REGISTRY_SETTINGS)
    );
    const auto it = std::ranges::find_if(catalog, [&](const x::json::json &d) {
        return d["vendor_id"].get<uint32_t>() == vendor_id &&
               d["product_code"].get<uint32_t>() == product_code;
    });
    if (it == catalog.end()) return {std::nullopt, x::errors::NIL};
    x::json::json result{
        {"device", (*it)["device"]},
        {"channels", x::json::json::array()},
    };
    if (auto err = resolve_all(result["device"], read)) return {std::nullopt, err};
    if (!it->contains("channels")) return {std::move(result), x::errors::NIL};
    const auto &channels = (*it)["channels"];
    const auto index_step = channels["index_step"].get<uint16_t>();
    for (int n = 0; n < channels["count"].get<int>(); n++) {
        x::json::json channel{
            {"entries", entries_of(channels["entries"], n)},
            {"settings", channels["settings"]},
            {"requires", channels["requires"]},
        };
        const auto step = static_cast<uint16_t>(n * index_step);
        for (auto &setting: channel["settings"])
            shift(setting, step);
        for (auto &rule: channel["requires"])
            shift(rule, step);
        if (auto err = resolve_all(channel["settings"], read))
            return {std::nullopt, err};
        if (auto err = read_values(channel["settings"], read))
            return {std::nullopt, err};
        result["channels"].push_back(std::move(channel));
    }
    return {std::move(result), x::errors::NIL};
}
}
