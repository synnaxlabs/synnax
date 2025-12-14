// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>

extern "C" {
#include "lua.h"
}

#include "nlohmann/json.hpp"

#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/xerrors/errors.h"

using json = nlohmann::json;

/// @brief The xlua namespace provides utilities for interacting between C++ and Lua,
/// specifically focusing on converting between JSON/telemetry data and Lua values.
namespace xlua {
/// @brief Pushes a JSON value onto the Lua stack, converting it to the appropriate
/// Lua type
/// @param L The Lua state
/// @param value The JSON value to push
/// @return xerrors::Error if there's an unsupported JSON type, NIL on success
inline xerrors::Error push_json_value(lua_State *L, const json &value) {
    if (value.is_null())
        lua_pushnil(L);
    else if (value.is_boolean())
        lua_pushboolean(L, static_cast<int>(value.get<bool>()));
    else if (value.is_number_integer())
        lua_pushinteger(L, value.get<int64_t>());
    else if (value.is_number_float())
        lua_pushnumber(L, value.get<double>());
    else if (value.is_string())
        lua_pushstring(L, value.get<std::string>().c_str());
    else if (value.is_array()) {
        lua_createtable(L, static_cast<int>(value.size()), 0);
        lua_Integer index = 1; // Lua arrays are 1-based
        for (const auto &element: value) {
            auto err = push_json_value(L, element);
            if (!err.ok()) return err;
            lua_rawseti(L, -2, index++);
        }
    } else if (value.is_object()) {
        lua_createtable(L, 0, static_cast<int>(value.size()));
        for (auto it = value.begin(); it != value.end(); ++it) {
            lua_pushstring(L, it.key().c_str());
            auto err = push_json_value(L, it.value());
            if (!err.ok()) return err;
            lua_rawset(L, -3);
        }
    } else
        return xerrors::Error("unsupported JSON type");
    return xerrors::NIL;
}

/// @brief Sets a global Lua variable with a JSON value
/// @param L The Lua state
/// @param name The name of the global variable to set
/// @param value The JSON value to set
/// @return xerrors::Error if there's an error pushing the JSON value, NIL on
/// success
inline xerrors::Error
set_global_json_value(lua_State *L, const std::string &name, const json &value) {
    auto err = push_json_value(L, value);
    if (!err.ok()) return err;
    lua_setglobal(L, name.c_str());
    return xerrors::NIL;
}

/// @brief Sets multiple global Lua variables from a JSON object
/// @param L The Lua state
/// @param object The JSON object containing key-value pairs to set as globals
/// @return xerrors::Error if the input is not a JSON object or if setting any value
/// fails
inline xerrors::Error set_globals_from_json_object(lua_State *L, const json &object) {
    if (!object.is_object())
        return xerrors::Error(xerrors::VALIDATION, "input must be a JSON object");
    for (auto it = object.begin(); it != object.end(); ++it) {
        auto err = set_global_json_value(L, it.key(), it.value());
        if (!err.ok()) return err;
    }
    return xerrors::NIL;
}

/// @brief sets a global variable on the lua state with the given name and value,
/// according to the data type.
/// @param L the lua state
/// @param name the name of the global variable
/// @param data_type the data type of the value
/// @param value the value to set
/// @return xerrors::Error if there's a type mismatch or JSON parsing error
[[nodiscard]] inline xerrors::Error set_global_sample_value(
    lua_State *L,
    const std::string &name,
    const telem::DataType &data_type,
    const telem::SampleValue &value
) {
    try {
        if (data_type == telem::FLOAT64_T)
            lua_pushnumber(L, std::get<double>(value));
        else if (data_type == telem::FLOAT32_T)
            lua_pushnumber(L, std::get<float>(value));
        else if (data_type == telem::INT64_T)
            lua_pushinteger(L, std::get<int64_t>(value));
        else if (data_type == telem::INT32_T)
            lua_pushinteger(L, std::get<int32_t>(value));
        else if (data_type == telem::INT16_T)
            lua_pushinteger(L, std::get<int16_t>(value));
        else if (data_type == telem::INT8_T)
            lua_pushinteger(L, std::get<int8_t>(value));
        else if (data_type == telem::UINT64_T) {
            const auto val = std::get<uint64_t>(value);
            if (val > static_cast<uint64_t>(std::numeric_limits<lua_Integer>::max()))
                lua_pushnumber(L, static_cast<lua_Number>(val));
            else
                lua_pushinteger(L, static_cast<lua_Integer>(val));
        } else if (data_type == telem::UINT32_T)
            lua_pushinteger(L, std::get<uint32_t>(value));
        else if (data_type == telem::UINT16_T)
            lua_pushinteger(L, std::get<uint16_t>(value));
        else if (data_type == telem::UINT8_T)
            lua_pushinteger(L, std::get<uint8_t>(value));
        else if (data_type == telem::STRING_T)
            lua_pushstring(L, std::get<std::string>(value).c_str());
        else if (data_type == telem::JSON_T) {
            try {
                const auto &str_val = std::get<std::string>(value);
                const auto parsed = json::parse(str_val);
                const auto err = push_json_value(L, parsed);
                if (!err.ok()) {
                    lua_pushnil(L);
                    return xerrors::Error(
                        xerrors::VALIDATION,
                        "failed to push JSON value for '" + name + "': " + err.message()
                    );
                }
            } catch (const json::parse_error &e) {
                lua_pushnil(L);
                return xerrors::Error(
                    xerrors::VALIDATION,
                    "invalid JSON format for '" + name + "': " + std::string(e.what())
                );
            }
        } else {
            lua_pushnil(L);
            return xerrors::Error(
                xerrors::VALIDATION,
                "unsupported data type for '" + name + "'"
            );
        }

        lua_setglobal(L, name.c_str());
        return xerrors::NIL;
    } catch (const std::bad_variant_access &) {
        lua_pushnil(L);
        lua_setglobal(L, name.c_str());
        return xerrors::Error(
            xerrors::VALIDATION,
            "type mismatch between data_type and value for '" + name + "'"
        );
    }
}

/// @brief Converts a Lua value at the given stack index to a telemetry Series
/// based on the specified channel's data type
/// @param L The Lua state
/// @param index The stack index of the Lua value
/// @param data_type The target data type for the Series
/// @return The converted Series
[[nodiscard]] inline std::pair<telem::Series, xerrors::Error>
to_series(lua_State *L, const int index, const telem::DataType &data_type) {
    // Check if the index contains any value (even nil)
    if (lua_isnone(L, index))
        return {
            telem::Series(telem::UNKNOWN_T, 0),
            xerrors::Error(xerrors::VALIDATION, "Invalid stack index")
        };

    if (lua_isnil(L, index))
        return {
            telem::Series(telem::UNKNOWN_T, 0),
            xerrors::Error(xerrors::VALIDATION, "Expected value but received nil")
        };

    const bool is_numeric = lua_isnumber(L, index) != 0 || lua_isinteger(L, index) != 0;
    const bool is_boolean = lua_isboolean(L, index) != 0;
    const bool is_string = lua_isstring(L, index) != 0;

    // Helper to get numeric value, handling both numbers and booleans
    auto get_numeric_value = [L, is_boolean](const int idx) -> lua_Number {
        if (is_boolean) return lua_toboolean(L, idx);
        return lua_tonumber(L, idx);
    };

    auto get_integer_value = [L, is_boolean](const int idx) -> lua_Integer {
        if (is_boolean) return lua_toboolean(L, idx);
        return lua_tointeger(L, idx);
    };

    if (!data_type.is_variable() && !is_numeric && !is_boolean) {
        std::string error_msg;
        if (is_string)
            error_msg = "cannot convert string value '" +
                        std::string(lua_tostring(L, index)) + "' to " + data_type;
        else
            error_msg = "cannot convert Lua type '" +
                        std::string(lua_typename(L, lua_type(L, index))) + "' to " +
                        data_type;
        return {
            telem::Series(telem::UNKNOWN_T, 0),
            xerrors::Error(xerrors::VALIDATION, error_msg)
        };
    }

    if (data_type == telem::STRING_T) {
        if (is_boolean) {
            const auto val = lua_toboolean(L, index) != 0;
            const std::string value = val ? "true" : "false";
            return {telem::Series(value), xerrors::NIL};
        }
        if (is_numeric) {
            return {
                telem::Series(std::to_string(get_numeric_value(index)), data_type),
                xerrors::NIL
            };
        }
        if (is_string) {
            return {
                telem::Series(std::string(lua_tostring(L, index)), data_type),
                xerrors::NIL
            };
        }
        return {
            telem::Series(telem::UNKNOWN_T, 0),
            xerrors::Error(
                xerrors::VALIDATION,
                "expected string value but received type '" +
                    std::string(lua_typename(L, lua_type(L, index))) + "'"
            )
        };
    }

    if (data_type == telem::FLOAT32_T)
        return {
            telem::Series(static_cast<float>(get_numeric_value(index)), data_type),
            xerrors::NIL
        };
    if (data_type == telem::FLOAT64_T)
        return {telem::Series(get_numeric_value(index), data_type), xerrors::NIL};
    if (data_type == telem::INT8_T)
        return {
            telem::Series(static_cast<int8_t>(get_numeric_value(index)), data_type),
            xerrors::NIL
        };
    if (data_type == telem::INT16_T)
        return {
            telem::Series(static_cast<int16_t>(get_numeric_value(index)), data_type),
            xerrors::NIL
        };
    if (data_type == telem::INT32_T)
        return {
            telem::Series(static_cast<int32_t>(get_numeric_value(index)), data_type),
            xerrors::NIL
        };
    if (data_type == telem::INT64_T)
        return {
            telem::Series(static_cast<int64_t>(get_integer_value(index)), data_type),
            xerrors::NIL
        };
    if (data_type == telem::UINT8_T)
        return {
            telem::Series(static_cast<uint8_t>(get_numeric_value(index)), data_type),
            xerrors::NIL
        };
    if (data_type == telem::UINT16_T)
        return {
            telem::Series(static_cast<uint16_t>(get_numeric_value(index)), data_type),
            xerrors::NIL
        };
    if (data_type == telem::UINT32_T)
        return {
            telem::Series(static_cast<uint32_t>(get_numeric_value(index)), data_type),
            xerrors::NIL
        };
    if (data_type == telem::UINT64_T)
        return {
            telem::Series(static_cast<uint64_t>(get_numeric_value(index)), data_type),
            xerrors::NIL
        };
    return {
        telem::Series(telem::UNKNOWN_T, 0),
        xerrors::Error(xerrors::VALIDATION, "Unsupported data type: " + data_type)
    };
}
}
