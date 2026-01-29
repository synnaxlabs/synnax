// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

extern "C" {
#include <lauxlib.h>
#include <lualib.h>
}

#include "x/cpp/lua/lua.h"
#include "x/cpp/test/test.h"

namespace x::lua {
class XLuaTest : public ::testing::Test {
protected:
    void SetUp() override {
        L = luaL_newstate();
        luaL_openlibs(L);
    }

    void TearDown() override { lua_close(L); }

    lua_State *L = nullptr;
};

// Telemetry Value Tests

/// @brief it should set a global float64 telemetry value in Lua.
TEST_F(XLuaTest, SetGlobalTelemFloat64) {
    const auto
        err = x::lua::set_global_sample_value(L, "val", telem::FLOAT64_T, 3.14159);
    ASSERT_NIL(err);
    lua_getglobal(L, "val");
    EXPECT_TRUE(lua_isnumber(L, -1));
    EXPECT_DOUBLE_EQ(lua_tonumber(L, -1), 3.14159);
    lua_pop(L, 1);
}

/// @brief it should set a global float32 telemetry value in Lua.
TEST_F(XLuaTest, SetGlobalTelemFloat32) {
    const auto err = x::lua::set_global_sample_value(L, "val", telem::FLOAT32_T, 3.14f);
    ASSERT_NIL(err);
    lua_getglobal(L, "val");
    EXPECT_FLOAT_EQ(lua_tonumber(L, -1), 3.14f);
    lua_pop(L, 1);
}

/// @brief it should set a global int64 telemetry value in Lua.
TEST_F(XLuaTest, SetGlobalTelemInt64) {
    const auto
        err = x::lua::set_global_sample_value(L, "val", telem::INT64_T, int64_t{42});
    ASSERT_NIL(err);

    lua_getglobal(L, "val");
    EXPECT_TRUE(lua_isinteger(L, -1));
    EXPECT_EQ(lua_tointeger(L, -1), 42);
    lua_pop(L, 1);
}

/// @brief it should set a global int32 telemetry value in Lua.
TEST_F(XLuaTest, SetGlobalTelemInt32) {
    const auto err = x::lua::set_global_sample_value(
        L,
        "val",
        telem::INT32_T,
        int32_t{2147483647}
    );
    ASSERT_NIL(err);
    lua_getglobal(L, "val");
    EXPECT_EQ(lua_tointeger(L, -1), 2147483647);
    lua_pop(L, 1);
}

/// @brief it should set a global int16 telemetry value in Lua.
TEST_F(XLuaTest, SetGlobalTelemInt16) {
    const auto
        err = x::lua::set_global_sample_value(L, "val", telem::INT16_T, int16_t{32767});
    ASSERT_NIL(err);
    lua_getglobal(L, "val");
    EXPECT_EQ(lua_tointeger(L, -1), 32767);
    lua_pop(L, 1);
}

/// @brief it should set a global int8 telemetry value in Lua.
TEST_F(XLuaTest, SetGlobalTelemInt8) {
    const auto
        err = x::lua::set_global_sample_value(L, "val", telem::INT8_T, int8_t{127});
    ASSERT_NIL(err);
    lua_getglobal(L, "val");
    EXPECT_EQ(lua_tointeger(L, -1), 127);
    lua_pop(L, 1);
}

/// @brief it should set a global uint32 telemetry value in Lua.
TEST_F(XLuaTest, SetGlobalTelemUInt32) {
    const auto err = x::lua::set_global_sample_value(
        L,
        "val",
        telem::UINT32_T,
        uint32_t{4294967295}
    );
    ASSERT_NIL(err);
    lua_getglobal(L, "val");
    EXPECT_EQ(lua_tointeger(L, -1), 4294967295);
    lua_pop(L, 1);
}

/// @brief it should set a global uint16 telemetry value in Lua.
TEST_F(XLuaTest, SetGlobalTelemUInt16) {
    const auto err = x::lua::set_global_sample_value(
        L,
        "val",
        telem::UINT16_T,
        uint16_t{65535}
    );
    ASSERT_NIL(err);
    lua_getglobal(L, "val");
    EXPECT_EQ(lua_tointeger(L, -1), 65535);
    lua_pop(L, 1);
}

/// @brief it should set a global uint8 telemetry value in Lua.
TEST_F(XLuaTest, SetGlobalTelemUInt8) {
    const auto
        err = x::lua::set_global_sample_value(L, "val", telem::UINT8_T, uint8_t{255});
    ASSERT_NIL(err);
    lua_getglobal(L, "val");
    EXPECT_EQ(lua_tointeger(L, -1), 255);
    lua_pop(L, 1);
}

/// @brief it should set a global string telemetry value in Lua.
TEST_F(XLuaTest, SetGlobalTelemString) {
    const auto err = x::lua::set_global_sample_value(
        L,
        "val",
        telem::STRING_T,
        std::string("hello")
    );
    ASSERT_NIL(err);
    lua_getglobal(L, "val");
    EXPECT_TRUE(lua_isstring(L, -1));
    EXPECT_STREQ(lua_tostring(L, -1), "hello");
    lua_pop(L, 1);
}

/// @brief it should set a global uint64 telemetry value within normal range.
TEST_F(XLuaTest, SetGlobalTelemUInt64Normal) {
    uint64_t val = 1000;
    const auto err = x::lua::set_global_sample_value(L, "val", telem::UINT64_T, val);
    ASSERT_NIL(err);
    lua_getglobal(L, "val");
    EXPECT_TRUE(lua_isinteger(L, -1));
    EXPECT_EQ(lua_tointeger(L, -1), val);
    lua_pop(L, 1);
}

/// @brief it should handle uint64 overflow by converting to double.
TEST_F(XLuaTest, SetGlobalTelemUInt64Overflow) {
    // Value that exceeds lua_Integer's max value
    uint64_t val = std::numeric_limits<uint64_t>::max();
    const auto err = x::lua::set_global_sample_value(L, "val", telem::UINT64_T, val);
    ASSERT_NIL(err);
    lua_getglobal(L, "val");
    EXPECT_TRUE(lua_isnumber(L, -1)); // Should be a number (double) not an integer
    EXPECT_DOUBLE_EQ(lua_tonumber(L, -1), static_cast<double>(val));
    lua_pop(L, 1);
}

/// @brief it should return validation error for float64 type mismatch.
TEST_F(XLuaTest, SetGlobalTelemTypeMismatchFloat64) {
    const auto err = x::lua::set_global_sample_value(
        L,
        "val",
        telem::FLOAT64_T,
        std::string("wrong type")
    );
    EXPECT_TRUE(err);
    EXPECT_EQ(err, errors::VALIDATION);
    lua_getglobal(L, "val");
    EXPECT_TRUE(lua_isnil(L, -1));
    lua_pop(L, 1);
}

/// @brief it should return validation error for int64 type mismatch.
TEST_F(XLuaTest, SetGlobalTelemTypeMismatchInt64) {
    const auto err = x::lua::set_global_sample_value(L, "val", telem::INT64_T, 3.14159);
    EXPECT_TRUE(err);
    EXPECT_EQ(err, errors::VALIDATION);
    lua_getglobal(L, "val");
    EXPECT_TRUE(lua_isnil(L, -1));
    lua_pop(L, 1);
}

/// @brief it should set a global JSON null value as Lua nil.
TEST_F(XLuaTest, SetGlobalJsonNull) {
    const json::json j_null;
    const auto err = x::lua::set_global_json_value(L, "val", j_null);
    ASSERT_NIL(err);
    lua_getglobal(L, "val");
    EXPECT_TRUE(lua_isnil(L, -1));
    lua_pop(L, 1);
}

/// @brief it should set a global JSON boolean value in Lua.
TEST_F(XLuaTest, SetGlobalJsonBoolean) {
    const json::json j_bool = true;
    const auto err = x::lua::set_global_json_value(L, "val", j_bool);
    ASSERT_NIL(err);
    lua_getglobal(L, "val");
    EXPECT_TRUE(lua_isboolean(L, -1));
    EXPECT_TRUE(lua_toboolean(L, -1));
    lua_pop(L, 1);
}

/// @brief it should set a global JSON integer value in Lua.
TEST_F(XLuaTest, SetGlobalJsonInteger) {
    const json::json j_int = 42;
    const auto err = x::lua::set_global_json_value(L, "val", j_int);
    ASSERT_NIL(err);
    lua_getglobal(L, "val");
    EXPECT_TRUE(lua_isinteger(L, -1));
    EXPECT_EQ(lua_tointeger(L, -1), 42);
    lua_pop(L, 1);
}

/// @brief it should set a global JSON float value in Lua.
TEST_F(XLuaTest, SetGlobalJsonFloat) {
    const json::json j_float = 3.14159;
    ASSERT_NIL(lua::set_global_json_value(L, "val", j_float));
    lua_getglobal(L, "val");
    EXPECT_TRUE(lua_isnumber(L, -1));
    EXPECT_DOUBLE_EQ(lua_tonumber(L, -1), 3.14159);
    lua_pop(L, 1);
}

/// @brief it should set a global JSON string value in Lua.
TEST_F(XLuaTest, SetGlobalJsonString) {
    const json::json j_string = "test string";
    const auto err = x::lua::set_global_json_value(L, "val", j_string);
    ASSERT_NIL(err);
    lua_getglobal(L, "val");
    EXPECT_TRUE(lua_isstring(L, -1));
    EXPECT_STREQ(lua_tostring(L, -1), "test string");
    lua_pop(L, 1);
}

/// @brief it should set a global JSON array as a Lua table.
TEST_F(XLuaTest, SetGlobalJsonArray) {
    const json::json j_array = {1, "two", 3.0};
    const auto err = x::lua::set_global_json_value(L, "val", j_array);
    ASSERT_NIL(err);
    lua_getglobal(L, "val");
    EXPECT_TRUE(lua_istable(L, -1));
    EXPECT_EQ(luaL_len(L, -1), 3);

    lua_rawgeti(L, -1, 1);
    EXPECT_EQ(lua_tointeger(L, -1), 1);
    lua_pop(L, 1);

    lua_rawgeti(L, -1, 2);
    EXPECT_STREQ(lua_tostring(L, -1), "two");
    lua_pop(L, 1);

    lua_rawgeti(L, -1, 3);
    EXPECT_DOUBLE_EQ(lua_tonumber(L, -1), 3.0);
    lua_pop(L, 2);
}

/// @brief it should set a global JSON object as a Lua table.
TEST_F(XLuaTest, SetGlobalJsonObject) {
    const json::json j_object = {
        {"string", "value"},
        {"number", 42},
        {"boolean", true}
    };
    const auto err = x::lua::set_global_json_value(L, "val", j_object);
    ASSERT_NIL(err);
    lua_getglobal(L, "val");
    EXPECT_TRUE(lua_istable(L, -1));

    lua_getfield(L, -1, "string");
    EXPECT_STREQ(lua_tostring(L, -1), "value");
    lua_pop(L, 1);

    lua_getfield(L, -1, "number");
    EXPECT_EQ(lua_tointeger(L, -1), 42);
    lua_pop(L, 1);

    lua_getfield(L, -1, "boolean");
    EXPECT_TRUE(lua_toboolean(L, -1));
    lua_pop(L, 2);
}

/// @brief it should set a global nested JSON structure as Lua tables.
TEST_F(XLuaTest, SetGlobalJsonNestedStructure) {
    const json::json j_nested = {
        {"array", {1, 2, 3}},
        {"object", {{"key", "value"}, {"nested_array", {4, 5, 6}}}}
    };
    const auto err = x::lua::set_global_json_value(L, "val", j_nested);
    ASSERT_NIL(err);
    lua_getglobal(L, "val");
    EXPECT_TRUE(lua_istable(L, -1));

    lua_getfield(L, -1, "array");
    EXPECT_TRUE(lua_istable(L, -1));
    EXPECT_EQ(luaL_len(L, -1), 3);
    lua_pop(L, 1);

    lua_getfield(L, -1, "object");
    EXPECT_TRUE(lua_istable(L, -1));
    lua_getfield(L, -1, "key");
    EXPECT_STREQ(lua_tostring(L, -1), "value");
    lua_pop(L, 3);
}

/// @brief it should set multiple globals from a simple JSON object.
TEST_F(XLuaTest, SetGlobalsFromJsonObjectSimple) {
    const json::json globals = {
        {"string_val", "test string"},
        {"int_val", 42},
        {"float_val", 3.14159},
        {"bool_val", true},
        {"null_val", nullptr}
    };
    EXPECT_TRUE(x::lua::set_globals_from_json_object(L, globals).ok());

    lua_getglobal(L, "string_val");
    EXPECT_TRUE(lua_isstring(L, -1));
    EXPECT_STREQ(lua_tostring(L, -1), "test string");
    lua_pop(L, 1);

    lua_getglobal(L, "int_val");
    EXPECT_TRUE(lua_isinteger(L, -1));
    EXPECT_EQ(lua_tointeger(L, -1), 42);
    lua_pop(L, 1);

    lua_getglobal(L, "float_val");
    EXPECT_TRUE(lua_isnumber(L, -1));
    EXPECT_DOUBLE_EQ(lua_tonumber(L, -1), 3.14159);
    lua_pop(L, 1);

    lua_getglobal(L, "bool_val");
    EXPECT_TRUE(lua_isboolean(L, -1));
    EXPECT_TRUE(lua_toboolean(L, -1));
    lua_pop(L, 1);

    lua_getglobal(L, "null_val");
    EXPECT_TRUE(lua_isnil(L, -1));
    lua_pop(L, 1);
}

/// @brief it should set multiple globals from a complex nested JSON object.
TEST_F(XLuaTest, SetGlobalsFromJsonObjectComplex) {
    const json::json globals = {
        {"array", {1, "two", 3.0}},
        {"object",
         {{"nested", "value"},
          {"numbers", {1, 2, 3}},
          {"deep", {{"key", "deep_value"}}}}}
    };
    EXPECT_TRUE(x::lua::set_globals_from_json_object(L, globals).ok());

    // Test array
    lua_getglobal(L, "array");
    EXPECT_TRUE(lua_istable(L, -1));
    EXPECT_EQ(luaL_len(L, -1), 3);

    lua_rawgeti(L, -1, 1);
    EXPECT_EQ(lua_tointeger(L, -1), 1);
    lua_pop(L, 1);

    lua_rawgeti(L, -1, 2);
    EXPECT_STREQ(lua_tostring(L, -1), "two");
    lua_pop(L, 1);

    lua_rawgeti(L, -1, 3);
    EXPECT_DOUBLE_EQ(lua_tonumber(L, -1), 3.0);
    lua_pop(L, 2);

    // Test nested object
    lua_getglobal(L, "object");
    EXPECT_TRUE(lua_istable(L, -1));

    lua_getfield(L, -1, "nested");
    EXPECT_STREQ(lua_tostring(L, -1), "value");
    lua_pop(L, 1);

    lua_getfield(L, -1, "numbers");
    EXPECT_TRUE(lua_istable(L, -1));
    EXPECT_EQ(luaL_len(L, -1), 3);
    lua_pop(L, 1);

    lua_getfield(L, -1, "deep");
    EXPECT_TRUE(lua_istable(L, -1));
    lua_getfield(L, -1, "key");
    EXPECT_STREQ(lua_tostring(L, -1), "deep_value");
    lua_pop(L, 3);
}

/// @brief it should fail when setting globals from non-object JSON.
TEST_F(XLuaTest, SetGlobalsFromJsonObjectInvalid) {
    // Test with non-object JSON
    const json::json invalid_json = json::json::array({1, 2, 3});
    EXPECT_FALSE(x::lua::set_globals_from_json_object(L, invalid_json).ok());
}

/// @brief it should set a simple JSON telemetry value as a Lua table.
TEST_F(XLuaTest, SetGlobalTelemJsonSimple) {
    const json::json j = {{"key", "value"}, {"number", 42}};
    const auto err = x::lua::set_global_sample_value(
        L,
        "val",
        telem::JSON_T,
        nlohmann::to_string(j)
    );
    ASSERT_NIL(err);
    lua_getglobal(L, "val");
    EXPECT_TRUE(lua_istable(L, -1));
    lua_getfield(L, -1, "key");
    EXPECT_TRUE(lua_isstring(L, -1));
    EXPECT_STREQ(lua_tostring(L, -1), "value");
    lua_pop(L, 1);
    lua_getfield(L, -1, "number");
    EXPECT_TRUE(lua_isinteger(L, -1));
    EXPECT_EQ(lua_tointeger(L, -1), 42);
    lua_pop(L, 2);
}

/// @brief it should set a complex nested JSON telemetry value as Lua tables.
TEST_F(XLuaTest, SetGlobalTelemJsonComplex) {
    const json::json j = {
        {"array", {1, 2, 3}},
        {"object", {{"nested", "value"}, {"bool", true}, {"null", nullptr}}}
    };
    ASSERT_NIL(x::lua::set_global_sample_value(L, "val", telem::JSON_T, j.dump()));

    lua_getglobal(L, "val");
    EXPECT_TRUE(lua_istable(L, -1));

    // Check array
    lua_getfield(L, -1, "array");
    EXPECT_TRUE(lua_istable(L, -1));
    EXPECT_EQ(luaL_len(L, -1), 3);
    lua_rawgeti(L, -1, 1);
    EXPECT_EQ(lua_tointeger(L, -1), 1);
    lua_pop(L, 2); // pop array and first element

    // Check nested object
    lua_getfield(L, -1, "object");
    EXPECT_TRUE(lua_istable(L, -1));

    lua_getfield(L, -1, "nested");
    EXPECT_STREQ(lua_tostring(L, -1), "value");
    lua_pop(L, 1);

    lua_getfield(L, -1, "bool");
    EXPECT_TRUE(lua_toboolean(L, -1));
    lua_pop(L, 1);

    lua_getfield(L, -1, "null");
    EXPECT_TRUE(lua_isnil(L, -1));
    lua_pop(L, 3); // pop null, object, and main table
}

/// @brief it should return validation error for invalid JSON telemetry value.
TEST_F(XLuaTest, SetGlobalTelemJsonInvalid) {
    ASSERT_OCCURRED_AS(
        x::lua::set_global_sample_value(L, "val", telem::JSON_T, "invalid json"),
        x::errors::VALIDATION
    );
    lua_getglobal(L, "val");
    EXPECT_TRUE(lua_isnil(L, -1));
    lua_pop(L, 1);
}

/// @brief it should coerce Lua boolean values to numeric series types.
TEST_F(XLuaTest, ToSeriesBooleanCoercion) {
    // Set up a boolean value in Lua
    lua_pushboolean(L, true);

    // Test coercion to various numeric types
    {
        auto [series, err] = x::lua::to_series(L, -1, telem::FLOAT64_T);
        EXPECT_FALSE(err) << err;
        EXPECT_EQ(series.data_type(), telem::FLOAT64_T);
        EXPECT_DOUBLE_EQ(series.at<double>(0), 1.0);
    }

    {
        auto [series, err] = x::lua::to_series(L, -1, telem::FLOAT32_T);
        EXPECT_FALSE(err) << err;
        EXPECT_EQ(series.data_type(), telem::FLOAT32_T);
        EXPECT_FLOAT_EQ(series.at<float>(0), 1.0f);
    }

    {
        auto [series, err] = x::lua::to_series(L, -1, telem::INT64_T);
        EXPECT_FALSE(err) << err;
        EXPECT_EQ(series.data_type(), telem::INT64_T);
        EXPECT_EQ(series.at<int64_t>(0), 1);
    }

    {
        auto [series, err] = x::lua::to_series(L, -1, telem::INT32_T);
        EXPECT_FALSE(err) << err;
        EXPECT_EQ(series.data_type(), telem::INT32_T);
        EXPECT_EQ(series.at<int32_t>(0), 1);
    }

    {
        auto [series, err] = x::lua::to_series(L, -1, telem::INT16_T);
        EXPECT_FALSE(err) << err;
        EXPECT_EQ(series.data_type(), telem::INT16_T);
        EXPECT_EQ(series.at<int16_t>(0), 1);
    }

    {
        auto [series, err] = x::lua::to_series(L, -1, telem::INT8_T);
        EXPECT_FALSE(err) << err;
        EXPECT_EQ(series.data_type(), telem::INT8_T);
        EXPECT_EQ(series.at<int8_t>(0), 1);
    }

    {
        auto [series, err] = x::lua::to_series(L, -1, telem::UINT64_T);
        EXPECT_FALSE(err) << err;
        EXPECT_EQ(series.data_type(), telem::UINT64_T);
        EXPECT_EQ(series.at<uint64_t>(0), 1);
    }

    {
        auto [series, err] = x::lua::to_series(L, -1, telem::UINT32_T);
        EXPECT_FALSE(err) << err;
        EXPECT_EQ(series.data_type(), telem::UINT32_T);
        EXPECT_EQ(series.at<uint32_t>(0), 1);
    }

    {
        auto [series, err] = x::lua::to_series(L, -1, telem::UINT16_T);
        EXPECT_FALSE(err) << err;
        EXPECT_EQ(series.data_type(), telem::UINT16_T);
        EXPECT_EQ(series.at<uint16_t>(0), 1);
    }

    {
        auto [series, err] = x::lua::to_series(L, -1, telem::UINT8_T);
        EXPECT_FALSE(err) << err;
        EXPECT_EQ(series.data_type(), telem::UINT8_T);
        EXPECT_EQ(series.at<uint8_t>(0), 1);
    }

    lua_pop(L, 1);

    // Test false value
    lua_pushboolean(L, false);

    {
        auto [series, err] = x::lua::to_series(L, -1, telem::FLOAT64_T);
        EXPECT_FALSE(err) << err;
        EXPECT_EQ(series.data_type(), telem::FLOAT64_T);
        EXPECT_DOUBLE_EQ(series.at<double>(0), 0.0);
    }

    {
        auto [series, err] = x::lua::to_series(L, -1, telem::INT32_T);
        EXPECT_FALSE(err) << err;
        EXPECT_EQ(series.data_type(), telem::INT32_T);
        EXPECT_EQ(series.at<int32_t>(0), 0);
    }

    lua_pop(L, 1);
}

/// @brief it should coerce Lua number values to various series types.
TEST_F(XLuaTest, ToSeriesNumberCoercion) {
    // Test integer to various numeric types
    lua_pushinteger(L, 42);

    {
        auto [series, err] = x::lua::to_series(L, -1, telem::FLOAT64_T);
        EXPECT_FALSE(err) << err;
        EXPECT_EQ(series.data_type(), telem::FLOAT64_T);
        EXPECT_DOUBLE_EQ(series.at<double>(0), 42.0);
    }

    {
        auto [series, err] = x::lua::to_series(L, -1, telem::INT32_T);
        EXPECT_FALSE(err) << err;
        EXPECT_EQ(series.data_type(), telem::INT32_T);
        EXPECT_EQ(series.at<int32_t>(0), 42);
    }

    lua_pop(L, 1);

    // Test float to various numeric types
    lua_pushnumber(L, 3.14159);

    {
        auto [series, err] = x::lua::to_series(L, -1, telem::FLOAT64_T);
        EXPECT_FALSE(err) << err;
        EXPECT_EQ(series.data_type(), telem::FLOAT64_T);
        EXPECT_DOUBLE_EQ(series.at<double>(0), 3.14159);
    }

    {
        auto [series, err] = x::lua::to_series(L, -1, telem::FLOAT32_T);
        EXPECT_FALSE(err) << err;
        EXPECT_EQ(series.data_type(), telem::FLOAT32_T);
        EXPECT_FLOAT_EQ(series.at<float>(0), 3.14159f);
    }

    lua_pop(L, 1);
}

/// @brief it should convert Lua strings to string series type.
TEST_F(XLuaTest, ToSeriesStringHandling) {
    // Test string to string type
    lua_pushstring(L, "test string");

    {
        auto [series, err] = x::lua::to_series(L, -1, telem::STRING_T);
        EXPECT_FALSE(err) << err;
        EXPECT_EQ(series.data_type(), telem::STRING_T);
        EXPECT_EQ(series.at<std::string>(0), "test string");
    }

    lua_pop(L, 1);

    // Test empty string
    lua_pushstring(L, "");

    {
        auto [series, err] = x::lua::to_series(L, -1, telem::STRING_T);
        EXPECT_FALSE(err) << err;
        EXPECT_EQ(series.data_type(), telem::STRING_T);
        EXPECT_EQ(series.at<std::string>(0), "");
    }

    lua_pop(L, 1);
}

/// @brief it should convert Lua boolean values to string series.
TEST_F(XLuaTest, booleanToString) {
    // Test true
    lua_pushboolean(L, true);
    auto [series1, err1] = x::lua::to_series(L, -1, telem::STRING_T);
    EXPECT_FALSE(err1) << err1;
    EXPECT_EQ(series1.data_type(), telem::STRING_T);
    EXPECT_EQ(series1.at<std::string>(0), "true");
    lua_pop(L, 1);

    // Test false
    lua_pushboolean(L, false);
    auto [series2, err2] = x::lua::to_series(L, -1, telem::STRING_T);
    EXPECT_FALSE(err2) << err2;
    EXPECT_EQ(series2.data_type(), telem::STRING_T);
    EXPECT_EQ(series2.at<std::string>(0), "false");
    lua_pop(L, 1);
}

/// @brief it should return validation error for incompatible type conversions.
TEST_F(XLuaTest, ToSeriesTypeMismatch) {
    // Test string to numeric type
    lua_pushstring(L, "not a number");
    auto [series, err] = x::lua::to_series(L, -1, telem::FLOAT64_T);
    EXPECT_TRUE(err) << err;
    EXPECT_EQ(err, errors::VALIDATION);
    lua_pop(L, 1);

    lua_pushstring(L, "not a number");
    auto [series2, err2] = x::lua::to_series(L, -1, telem::INT32_T);
    EXPECT_TRUE(err2) << err2;
}

/// @brief it should return validation error when converting nil to series.
TEST_F(XLuaTest, ToSeriesNilHandling) {
    lua_pushnil(L);

    // Test nil to various types
    auto [series1, err1] = x::lua::to_series(L, -1, telem::FLOAT64_T);
    EXPECT_TRUE(err1);
    EXPECT_EQ(err1, errors::VALIDATION);
    auto [series2, err2] = x::lua::to_series(L, -1, telem::INT32_T);
    EXPECT_TRUE(err2);
    EXPECT_EQ(err2, errors::VALIDATION);
    auto [series3, err3] = x::lua::to_series(L, -1, telem::STRING_T);
    EXPECT_TRUE(err3);
    EXPECT_EQ(err3, errors::VALIDATION);

    lua_pop(L, 1);
}

/// @brief it should handle numeric boundary values and special floating point values.
TEST_F(XLuaTest, ToSeriesNumericRanges) {
    // Test integer bounds
    {
        lua_pushinteger(L, std::numeric_limits<int16_t>::max());
        auto [series, err] = x::lua::to_series(L, -1, telem::INT16_T);
        EXPECT_FALSE(err) << err;
        EXPECT_EQ(series.at<int16_t>(0), std::numeric_limits<int16_t>::max());
        lua_pop(L, 1);
    }

    {
        lua_pushinteger(L, std::numeric_limits<int16_t>::min());
        auto [series, err] = x::lua::to_series(L, -1, telem::INT16_T);
        EXPECT_FALSE(err) << err;
        EXPECT_EQ(series.at<int16_t>(0), std::numeric_limits<int16_t>::min());
        lua_pop(L, 1);
    }

    // Test floating point special values
    {
        lua_pushnumber(L, std::numeric_limits<double>::infinity());
        auto [series, err] = x::lua::to_series(L, -1, telem::FLOAT64_T);
        EXPECT_FALSE(err) << err;
        EXPECT_TRUE(std::isinf(series.at<double>(0)));
        lua_pop(L, 1);
    }

    {
        lua_pushnumber(L, -std::numeric_limits<double>::infinity());
        auto [series, err] = x::lua::to_series(L, -1, telem::FLOAT64_T);
        EXPECT_FALSE(err) << err;
        EXPECT_TRUE(std::isinf(series.at<double>(0)));
        EXPECT_LT(series.at<double>(0), 0);
        lua_pop(L, 1);
    }

    {
        lua_pushnumber(L, std::numeric_limits<double>::quiet_NaN());
        auto [series, err] = x::lua::to_series(L, -1, telem::FLOAT64_T);
        EXPECT_FALSE(err) << err;
        EXPECT_TRUE(std::isnan(series.at<double>(0)));
        lua_pop(L, 1);
    }
}

/// @brief it should return validation error for invalid stack index.
TEST_F(XLuaTest, ToSeriesInvalidIndex) {
    auto [series1, err1] = x::lua::to_series(L, 999, telem::FLOAT64_T);
    EXPECT_TRUE(err1);
    EXPECT_EQ(err1, errors::VALIDATION);
}

/// @brief it should return validation error for unsupported Lua types.
TEST_F(XLuaTest, ToSeriesUnsupportedTypes) {
    // Test with table
    lua_newtable(L);
    auto [series1, err1] = x::lua::to_series(L, -1, telem::FLOAT64_T);
    EXPECT_TRUE(err1);
    EXPECT_EQ(err1, errors::VALIDATION);
    lua_pop(L, 1);

    // Test with function
    lua_pushcfunction(L, [](lua_State *) -> int { return 0; });
    auto [series2, err2] = x::lua::to_series(L, -1, telem::FLOAT64_T);
    EXPECT_TRUE(err2);
    EXPECT_EQ(err2, errors::VALIDATION);
    lua_pop(L, 1);

    // Test with userdata
    lua_newuserdata(L, sizeof(int));
    auto [series3, err3] = x::lua::to_series(L, -1, telem::FLOAT64_T);
    EXPECT_TRUE(err3);
    EXPECT_EQ(err3, errors::VALIDATION);
    lua_pop(L, 1);
}

/// @brief it should handle maximum int64 value correctly.
TEST_F(XLuaTest, Int64Max) {
    lua_pushinteger(L, 9223372036854775807);
    auto [series1, err1] = x::lua::to_series(L, -1, telem::INT64_T);
    EXPECT_FALSE(err1) << err1;
    EXPECT_EQ(series1.at<int64_t>(0), 9223372036854775807LL);
}
}
