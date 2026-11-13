// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <bit>
#include <cstdint>
#include <format>
#include <map>
#include <set>
#include <string>
#include <utility>

#include "x/cpp/telem/telem.h"

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/esi/settings.h"
#include "driver/ethercat/telem/telem.h"

namespace driver::ethercat::esi {
namespace {
#include "driver/ethercat/esi/registry_settings.inc"

/// @brief a card table that counts no options, or more than its object can hold.
const x::errors::Error INVALID_OPTION_COUNT = errors::SDO_READ_ERROR.sub(
    "invalid_option_count"
);

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
                INVALID_OPTION_COUNT,
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

/// @brief returns the address of each entry that channel n reads, in every layout, with
/// its calibration. The objects of a calibration are index_step apart from channel to
/// channel.
x::json::json
entries_of(const x::json::json &layouts, const int n, const uint16_t index_step) {
    auto entries = x::json::json::array();
    for (const auto &layout: layouts) {
        const auto index = layout["index"].get<int>() +
                           n * layout["index_step"].get<int>();
        const auto first = layout["sub_index"].get<int>() +
                           n * layout["sub_index_step"].get<int>();
        for (int k = 0; k < layout["count"].get<int>(); k++) {
            x::json::json entry{{"index", index}, {"sub_index", first + k}};
            if (layout.contains("only_read_by"))
                entry["only_read_by"] = layout["only_read_by"];
            if (layout.contains("calibration")) {
                entry["calibration"] = layout["calibration"];
                entry["calibration"]["index"] = layout["calibration"]["index"]
                                                    .get<int>() +
                                                n * index_step;
            }
            entries.push_back(std::move(entry));
        }
    }
    return entries;
}

/// @brief returns the startup parameter that writes value to the object of item.
x::json::json parameter(const x::json::json &item, const x::json::json &value) {
    return {
        {"index", item["index"]},
        {"sub_index", item["sub_index"]},
        {"data_type", item["data_type"]},
        {"value", value},
    };
}

/// @brief returns the mode that the card holds, as read_settings describes it.
std::pair<x::json::json, x::errors::Error> read_mode(
    const x::json::json &mode,
    const uint32_t revision,
    const std::vector<dc::OpMode> &op_modes,
    const coe::Read &read
) {
    const auto [held, err] = read(
        mode["index"].get<uint16_t>(),
        mode["sub_index"].get<uint8_t>()
    );
    if (err) return {{}, err};
    x::json::json result{
        {"index", mode["index"]},
        {"sub_index", mode["sub_index"]},
        {"value", held},
        {"shared", false},
        {"switch", x::json::json::array()},
    };
    if (mode.contains("second_bus"))
        result["bus"] = (revision & mode["second_bus"].get<uint32_t>()) != 0 ? 2 : 1;
    const double value = held;
    const auto &values = mode["values"];
    const auto match = std::ranges::find_if(values, [value](const auto &v) {
        return v["value"].template get<double>() == value;
    });
    if (match != values.end()) {
        for (const auto *key: {"shared_with", "only_read_by"})
            if (match->contains(key)) {
                result["shared"] = true;
                result[key] = (*match)[key];
            }
        if (match->contains("op_mode")) {
            const auto &op = (*match)["op_mode"];
            const auto aa = op["assign_activate"].get<uint16_t>();
            const auto sii = std::ranges::find(
                op_modes,
                aa,
                &dc::OpMode::assign_activate
            );
            if (sii != op_modes.end())
                result["op_mode"] =
                    dc::Choice{
                        .index = static_cast<uint16_t>(sii - op_modes.begin()),
                        .assign_activate = aa,
                        .cycles =
                            {
                                .sync0 = x::telem::TimeSpan(
                                    op.value<int64_t>("sync0_cycle", 0)
                                ),
                                .sync1 = x::telem::TimeSpan(
                                    op.value<int64_t>("sync1_cycle", 0)
                                ),
                            },
                    }
                        .to_json();
        }
    }
    if (value == mode["standalone"].get<double>()) return {result, x::errors::NIL};
    result["switch"].push_back(parameter(mode, mode["standalone"]));
    if (!mode.contains("save")) return {result, x::errors::NIL};
    const auto &save = mode["save"];
    const auto [held_flag, save_err] = read(
        save["index"].get<uint16_t>(),
        save["sub_index"].get<uint8_t>()
    );
    if (save_err.matches(coe::OBJECT_MISSING) ||
        save_err.matches(coe::SUBINDEX_MISSING))
        return {result, x::errors::NIL};
    if (save_err) return {{}, save_err};
    const double flag = held_flag;
    if (std::ranges::any_of(save["when"], [flag](const auto &w) {
            return w.template get<double>() == flag;
        }))
        result["switch"].push_back(parameter(save, save["value"]));
    return {result, x::errors::NIL};
}

using Address = std::pair<uint16_t, uint8_t>;

Address address_of(const x::json::json &item) {
    return {item["index"].get<uint16_t>(), item["sub_index"].get<uint8_t>()};
}

/// @brief keeps the first error in first, or err when it ends the read. Returns whether
/// err ends the read, as each error does except one that shows that the card does not
/// hold what a setting describes.
bool ends(x::errors::Error &first, const x::errors::Error &err) {
    const bool fatal = err && !err.matches(coe::OBJECT_MISSING) &&
                       !err.matches(coe::SUBINDEX_MISSING) &&
                       !err.matches(INVALID_OPTION_COUNT);
    if (!first || fatal) first = err;
    return fatal;
}

/// @brief removes each item for which drop returns true. Returns whether it removed
/// any.
template<typename Drop>
bool remove_if(x::json::json &items, const Drop &drop) {
    auto kept = x::json::json::array();
    for (auto &item: items)
        if (!drop(item)) kept.push_back(std::move(item));
    const bool removed = kept.size() != items.size();
    items = std::move(kept);
    return removed;
}

/// @brief whether a setting holds each address that the conditions of item under key
/// name.
bool names_held(
    const x::json::json &item,
    const char *key,
    const std::set<Address> &held
) {
    if (!item.contains(key)) return true;
    return std::ranges::all_of(item[key], [&held](const auto &condition) {
        return held.contains(address_of(condition));
    });
}

/// @brief removes each setting whose conditions name an address that no setting holds,
/// until none does. Returns the addresses of the settings left.
std::set<Address> prune(x::json::json &settings) {
    std::set<Address> held;
    do {
        held.clear();
        for (const auto &setting: settings)
            held.insert(address_of(setting));
    } while (remove_if(settings, [&held](const auto &setting) {
        return !names_held(setting, "visible_when", held);
    }));
    return held;
}

/// @brief removes each setting for which read_one returns an error. Returns the first
/// error, or at once an error that ends the read.
template<typename ReadOne>
x::errors::Error read_each(x::json::json &settings, const ReadOne &read_one) {
    x::errors::Error first;
    auto kept = x::json::json::array();
    for (auto &setting: settings) {
        const auto err = read_one(setting);
        if (ends(first, err)) return first;
        if (!err) kept.push_back(std::move(setting));
    }
    settings = std::move(kept);
    return first;
}

/// @brief resolves each setting, and removes each one whose options or limit the card
/// does not hold. Returns as read_each does.
x::errors::Error resolve_all(x::json::json &settings, const coe::Read &read) {
    return read_each(settings, [&read](x::json::json &setting) {
        return resolve(setting, read);
    });
}

/// @brief sets the "value" of each setting to the float32 or unsigned value that the
/// card holds, and removes each setting whose value the card does not hold. Reads each
/// address once. Returns as read_each does.
x::errors::Error read_values(x::json::json &settings, const coe::Read &read) {
    std::map<Address, std::pair<uint32_t, x::errors::Error>> held;
    return read_each(settings, [&](x::json::json &setting) -> x::errors::Error {
        const auto address = address_of(setting);
        auto it = held.find(address);
        if (it == held.end())
            it = held.emplace(address, read(address.first, address.second)).first;
        const auto &[raw, err] = it->second;
        if (err) return err;
        if (setting["data_type"] == "float32")
            setting["value"] = std::bit_cast<float>(raw);
        else
            setting["value"] = raw;
        return x::errors::NIL;
    });
}

/// @brief appends the settings of each input that channels describes to out, with the
/// option lists, limits, and values that the card holds. Returns as read_each does.
x::errors::Error read_channels(
    const x::json::json &channels,
    const coe::Read &read,
    x::json::json &out
) {
    x::errors::Error first;
    const auto index_step = channels["index_step"].get<uint16_t>();
    for (int n = 0; n < channels["count"].get<int>(); n++) {
        x::json::json channel{
            {"entries", entries_of(channels["entries"], n, index_step)},
            {"settings", channels["settings"]},
            {"requires", channels["requires"]},
        };
        const auto step = static_cast<uint16_t>(n * index_step);
        for (auto &setting: channel["settings"])
            shift(setting, step);
        for (auto &rule: channel["requires"])
            shift(rule, step);
        if (ends(first, resolve_all(channel["settings"], read))) return first;
        if (ends(first, read_values(channel["settings"], read))) return first;
        const auto held = prune(channel["settings"]);
        remove_if(channel["requires"], [&held](const auto &rule) {
            return !held.contains(address_of(rule["then"])) ||
                   !names_held(rule, "when", held);
        });
        out.push_back(std::move(channel));
    }
    return first;
}
}

std::pair<std::optional<x::json::json>, x::errors::Error> read_settings(
    const uint32_t vendor_id,
    const uint32_t product_code,
    const uint32_t revision,
    const std::vector<dc::OpMode> &op_modes,
    const coe::Read &read
) {
    const auto catalog = x::json::json::parse(
        reinterpret_cast<const char *>(REGISTRY_SETTINGS)
    );
    const auto it = std::ranges::find_if(catalog, [&](const x::json::json &d) {
        if (d["vendor_id"].get<uint32_t>() != vendor_id ||
            d["product_code"].get<uint32_t>() != product_code)
            return false;
        if (!d.contains("revisions")) return true;
        const auto masked = revision & d["revision_mask"].get<uint32_t>();
        return std::ranges::any_of(d["revisions"], [masked](const auto &r) {
            return r.template get<uint32_t>() == masked;
        });
    });
    if (it == catalog.end()) return {std::nullopt, x::errors::NIL};
    x::json::json result{
        {"device", (*it)["device"]},
        {"channels", x::json::json::array()},
    };
    x::errors::Error err;
    if (ends(err, resolve_all(result["device"], read))) return {std::nullopt, err};
    if (ends(err, read_values(result["device"], read))) return {std::nullopt, err};
    prune(result["device"]);
    if (it->contains("channels") &&
        ends(err, read_channels((*it)["channels"], read, result["channels"])))
        return {std::nullopt, err};
    if (!it->contains("mode")) return {std::move(result), err};
    auto [mode, mode_err] = read_mode((*it)["mode"], revision, op_modes, read);
    if (ends(err, mode_err)) return {std::nullopt, err};
    if (!mode_err) result["mode"] = std::move(mode);
    return {std::move(result), err};
}
}
