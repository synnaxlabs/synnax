#include "driver/sequence/json_operator.h"
#include "gtest/gtest.h"
#include <nlohmann/json.hpp>
#include <memory>
extern "C" {
#include <lua.h>
#include <lauxlib.h>
#include <lualib.h>
}

using json = nlohmann::json;
using namespace sequence;

TEST(JSONSourceTest, BasicVariableApplication) {
    lua_State* L = luaL_newstate();
    luaL_openlibs(L);
    
    // Create a JSON object using nlohmann::json
    json test_data = {
        {"number", 42.5},
        {"string", "hello"},
        {"boolean", true},
        {"array", {1, 2, 3}},
        {"nested", {
            {"value", 123}
        }}
    };
    
    JSONOperator source(test_data);
    ASSERT_EQ(source.bind(L), freighter::NIL);
    
    // Test number variable
    lua_getglobal(L, "number");
    ASSERT_TRUE(lua_isnumber(L, -1));
    ASSERT_EQ(lua_tonumber(L, -1), 42.5);
    lua_pop(L, 1);
    
    // Test string variable
    lua_getglobal(L, "string");
    ASSERT_TRUE(lua_isstring(L, -1));
    ASSERT_STREQ(lua_tostring(L, -1), "hello");
    lua_pop(L, 1);
    
    // Test boolean variable
    lua_getglobal(L, "boolean");
    ASSERT_TRUE(lua_isboolean(L, -1));
    ASSERT_TRUE(lua_toboolean(L, -1));
    lua_pop(L, 1);
    
    // Test array
    lua_getglobal(L, "array");
    ASSERT_TRUE(lua_istable(L, -1));
    for (int i = 1; i <= 3; i++) {
        lua_rawgeti(L, -1, i);
        ASSERT_TRUE(lua_isnumber(L, -1));
        ASSERT_EQ(lua_tonumber(L, -1), i);
        lua_pop(L, 1);
    }
    lua_pop(L, 1);
    
    // Test nested object
    lua_getglobal(L, "nested");
    ASSERT_TRUE(lua_istable(L, -1));
    lua_getfield(L, -1, "value");
    ASSERT_TRUE(lua_isnumber(L, -1));
    ASSERT_EQ(lua_tonumber(L, -1), 123);
    lua_pop(L, 2);
    
    lua_close(L);
}

TEST(JSONSourceTest, InvalidJSON) {
    lua_State* L = luaL_newstate();
    luaL_openlibs(L);
    
    // Create an invalid JSON (array instead of object)
    json invalid_json = json::array();
    
    JSONOperator source(invalid_json);
    ASSERT_FALSE(source.bind(L).ok());
    
    lua_close(L);
}