// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std. lib
#include <string>
#include <cstdint>
#include <utility>

/// external.
#include "nlohmann/json.hpp"
#include "freighter/cpp/freighter.h"
extern "C" {
#include <lua.h>
}

/// internal.
#include "driver/sequence/source.h"

using json = nlohmann::json;

namespace sequence {


/// @brief JSON source is an implementation of sequence::Source that binds JSON data
/// into the Lua state. This is useful for binding fixed variable context at the start
/// of a sequence.
class JSONSource final : public Source {
    /// @brief the data to bind to the Lua state.
    json data;

    static freighter::Error push_value(lua_State *L, const json& value) {
        if (value.is_null()) lua_pushnil(L);
        else if (value.is_boolean()) lua_pushboolean(L, value.get<bool>());
        else if (value.is_number_integer()) lua_pushinteger(L, value.get<int64_t>());
        else if (value.is_number_float()) lua_pushnumber(L, value.get<double>());
        else if (value.is_string()) lua_pushstring(L, value.get<std::string>().c_str());
        else if (value.is_array()) {
            lua_createtable(L, value.size(), 0);
            int index = 1; // Lua arrays are 1-based
            for (const auto& element : value) {
                auto err = push_value(L, element);
                if (!err.ok()) return err;
                lua_rawseti(L, -2, index++);
            }
        } else if (value.is_object()) {
            lua_createtable(L, 0, value.size());
            for (auto it = value.begin(); it != value.end(); ++it) {
                lua_pushstring(L, it.key().c_str());
                auto err = push_value(L, it.value());
                if (!err.ok()) return err;
                lua_rawset(L, -3);
            }
        } else
            return freighter::Error("Unsupported JSON type");
        return freighter::NIL;
    }
public:
    explicit JSONSource(json  source_data) : data(std::move(source_data)) {}
    
    freighter::Error bind(lua_State *L) override {
        if (!data.is_object())
            return freighter::Error("Root JSON must be an object");
        for (auto it = data.begin(); it != data.end(); ++it) {
            if (auto err = push_value(L, it.value()); err) return err;
            lua_setglobal(L, it.key().c_str());
        }
        return freighter::NIL;
    }
};
}