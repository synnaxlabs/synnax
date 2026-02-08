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
#include <string>
#include <unordered_map>

#include "x/cpp/telem/series.h"

#include "arc/cpp/stl/series/state.h"
#include "arc/cpp/stl/str/state.h"

namespace arc::stl::stateful {

/// Persistent stateful variable storage. Variables are keyed by (node_key, var_id)
/// and persist across execution cycles. Each node has its own isolated namespace.
class Variables {
    std::string current_node_key;

    std::unordered_map<std::string, std::unordered_map<uint32_t, uint8_t>> u8;
    std::unordered_map<std::string, std::unordered_map<uint32_t, uint16_t>> u16;
    std::unordered_map<std::string, std::unordered_map<uint32_t, uint32_t>> u32;
    std::unordered_map<std::string, std::unordered_map<uint32_t, uint64_t>> u64;
    std::unordered_map<std::string, std::unordered_map<uint32_t, int8_t>> i8;
    std::unordered_map<std::string, std::unordered_map<uint32_t, int16_t>> i16;
    std::unordered_map<std::string, std::unordered_map<uint32_t, int32_t>> i32;
    std::unordered_map<std::string, std::unordered_map<uint32_t, int64_t>> i64;
    std::unordered_map<std::string, std::unordered_map<uint32_t, float>> f32;
    std::unordered_map<std::string, std::unordered_map<uint32_t, double>> f64;
    std::unordered_map<std::string, std::unordered_map<uint32_t, std::string>> string;
    std::unordered_map<std::string, std::unordered_map<uint32_t, telem::Series>> series;

public:
    void set_current_node_key(const std::string &key) { this->current_node_key = key; }

#define DECLARE_VAR_OPS(suffix, cpptype)                                               \
    cpptype load_##suffix(uint32_t var_id, cpptype init_value) {                       \
        auto &inner = this->suffix[this->current_node_key];                            \
        const auto it = inner.find(var_id);                                            \
        if (it != inner.end()) return it->second;                                      \
        inner[var_id] = init_value;                                                    \
        return init_value;                                                             \
    }                                                                                  \
    void store_##suffix(uint32_t var_id, cpptype value) {                              \
        this->suffix[this->current_node_key][var_id] = value;                          \
    }

    DECLARE_VAR_OPS(u8, uint8_t)
    DECLARE_VAR_OPS(u16, uint16_t)
    DECLARE_VAR_OPS(u32, uint32_t)
    DECLARE_VAR_OPS(u64, uint64_t)
    DECLARE_VAR_OPS(i8, int8_t)
    DECLARE_VAR_OPS(i16, int16_t)
    DECLARE_VAR_OPS(i32, int32_t)
    DECLARE_VAR_OPS(i64, int64_t)
    DECLARE_VAR_OPS(f32, float)
    DECLARE_VAR_OPS(f64, double)

#undef DECLARE_VAR_OPS

    uint32_t load_str(uint32_t var_id, uint32_t init_handle, str::State &str_state) {
        auto &inner = this->string[this->current_node_key];
        if (const auto it = inner.find(var_id); it != inner.end())
            return str_state.create(it->second);
        const auto init_str = str_state.get(init_handle);
        inner[var_id] = init_str;
        return str_state.create(inner[var_id]);
    }

    void store_str(uint32_t var_id, uint32_t str_handle, const str::State &str_state) {
        const auto s = str_state.get(str_handle);
        if (!s.empty() || str_state.exists(str_handle))
            this->string[this->current_node_key][var_id] = s;
    }

    uint32_t
    load_series(uint32_t var_id, uint32_t init_handle, series::State &series_state) {
        auto &inner = this->series[this->current_node_key];
        if (const auto it = inner.find(var_id); it != inner.end())
            return series_state.store(it->second.deep_copy());
        if (const auto *init = series_state.get(init_handle); init != nullptr)
            inner.emplace(var_id, init->deep_copy());
        return init_handle;
    }

    void store_series(uint32_t var_id, uint32_t handle, const series::State &s) {
        if (const auto *ser = s.get(handle); ser != nullptr)
            this->series[this->current_node_key].insert_or_assign(
                var_id,
                ser->deep_copy()
            );
    }

    void reset() {
        this->u8.clear();
        this->u16.clear();
        this->u32.clear();
        this->u64.clear();
        this->i8.clear();
        this->i16.clear();
        this->i32.clear();
        this->i64.clear();
        this->f32.clear();
        this->f64.clear();
        this->string.clear();
        this->series.clear();
    }
};

}
