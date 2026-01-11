// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std.
#include <memory>
#include <utility>

#include "gtest/gtest.h"
extern "C" {
#include <lualib.h>
}

/// internal.
#include "driver/pipeline/mock/pipeline.h"
#include "driver/sequence/plugins/mock/plugins.h"
#include "driver/sequence/plugins/plugins.h"

class SetOperatorTest : public testing::Test {
protected:
    void SetupChannel(x::telem::DataType data_type) {
        channels.clear();
        sink = std::make_shared<plugins::mock::FrameSink>();

        synnax::channel::Channel::Channel ch;
        ch.name = "my_channel";
        ch.data_type = std::move(data_type);
        ch.key = 1;
        ch.is_virtual = true;
        channels.push_back(ch);

        op = std::make_unique<plugins::ChannelWrite>(sink, channels);
        L = luaL_newstate();
        luaL_openlibs(L);
        op->before_all(L);
    }

    void TearDown() override {
        lua_close(L);
        channels.clear();
    }

    template<typename T>
    void RunTest(const char *lua_value, T expected_value) {
        const std::string script = "set('my_channel', " + std::string(lua_value) + ")";
        ASSERT_EQ(luaL_dostring(L, script.c_str()), 0) << lua_tostring(L, -1);
        op->after_next(L);
        ASSERT_EQ(sink->writes->size(), 1);
        const x::telem::Series ser = std::move(sink->writes->at(0).series->at(0));
        EXPECT_EQ(ser.at<T>(0), expected_value);
    }

    void RunStringTest(const char *lua_value, const std::string &expected_value) const {
        const std::string script = "set('my_channel', " + std::string(lua_value) + ")";
        ASSERT_EQ(luaL_dostring(L, script.c_str()), 0) << lua_tostring(L, -1);
        op->after_next(L);
        ASSERT_EQ(sink->writes->size(), 1);
        const x::telem::Series ser = std::move(sink->writes->at(0).series->at(0));
        const auto value = ser.at<std::string>(0);
        ASSERT_EQ(value, expected_value);
    }

    std::shared_ptr<plugins::mock::FrameSink> sink;
    std::vector<synnax::channel::Channel::Channel> channels;
    std::unique_ptr<plugins::ChannelWrite> op;
    lua_State *L{};
};

/// @brief it should write float32 values to channel.
TEST_F(SetOperatorTest, Float32Value) {
    SetupChannel(x::telem::FLOAT32_T);
    RunTest<float>("3.14", 3.14f);
}

/// @brief it should write float64 values to channel.
TEST_F(SetOperatorTest, Float64Value) {
    SetupChannel(x::telem::FLOAT64_T);
    RunTest<double>("3.14159265359", 3.14159265359);
}

/// @brief it should write int8 values to channel.
TEST_F(SetOperatorTest, Int8Value) {
    SetupChannel(x::telem::INT8_T);
    RunTest<int8_t>("127", 127);
}

/// @brief it should write int16 values to channel.
TEST_F(SetOperatorTest, Int16Value) {
    SetupChannel(x::telem::INT16_T);
    RunTest<int16_t>("32767", 32767);
}

/// @brief it should write int32 values to channel.
TEST_F(SetOperatorTest, Int32Value) {
    SetupChannel(x::telem::INT32_T);
    RunTest<int32_t>("2147483647", 2147483647);
}

/// @brief it should write int64 values to channel.
TEST_F(SetOperatorTest, Int64Value) {
    SetupChannel(x::telem::INT64_T);
    RunTest<int64_t>("9223372036854775807", 9223372036854775807LL);
}

/// @brief it should write uint8 zero value to channel.
TEST_F(SetOperatorTest, Uint8NumberValue) {
    SetupChannel(x::telem::UINT8_T);
    RunTest<uint8_t>("0", 0);
}

/// @brief it should write uint8 one value to channel.
TEST_F(SetOperatorTest, Uint8Number1Value) {
    SetupChannel(x::telem::UINT8_T);
    RunTest<uint8_t>("1", 1);
}

/// @brief it should write boolean false as uint8 zero to channel.
TEST_F(SetOperatorTest, Uint8ChannelBooleanValue) {
    SetupChannel(x::telem::UINT8_T);
    RunTest<uint8_t>("false", 0);
}

/// @brief it should write Lua false as uint8 zero to channel.
TEST_F(SetOperatorTest, Uint8ChannelFalseValue) {
    SetupChannel(x::telem::UINT8_T);
    RunTest<uint8_t>("false", 0);
}

/// @brief it should write uint16 values to channel.
TEST_F(SetOperatorTest, UInt16Value) {
    SetupChannel(x::telem::UINT16_T);
    RunTest<uint16_t>("65535", 65535);
}

/// @brief it should write uint32 values to channel.
TEST_F(SetOperatorTest, UInt32Value) {
    SetupChannel(x::telem::UINT32_T);
    RunTest<uint32_t>("4294967295", 4294967295);
}

/// @brief it should write string values to channel.
TEST_F(SetOperatorTest, StringValue) {
    SetupChannel(x::telem::STRING_T);
    RunStringTest("'hello'", "hello");
}

/// @brief it should convert number to string when writing to string channel.
TEST_F(SetOperatorTest, StringTypeMismatch) {
    SetupChannel(x::telem::STRING_T);
    RunStringTest("123", "123.000000");
}

/// @brief it should reject string when writing to float32 channel.
TEST_F(SetOperatorTest, Float32TypeMismatch) {
    SetupChannel(x::telem::FLOAT32_T);
    ASSERT_NE(luaL_dostring(L, "set('my_channel', 'not a number')"), 0);
    EXPECT_EQ(sink->writes->size(), 0);
}

/// @brief it should reject string when writing to int32 channel.
TEST_F(SetOperatorTest, Int32TypeMismatch) {
    SetupChannel(x::telem::INT32_T);
    ASSERT_NE(luaL_dostring(L, "set('my_channel', 'not an integer')"), 0);
    EXPECT_EQ(sink->writes->size(), 0);
}

/// @brief it should reject string when writing to uint8 boolean channel.
TEST_F(SetOperatorTest, BooleanTypeMismatch) {
    SetupChannel(x::telem::UINT8_T);
    ASSERT_NE(luaL_dostring(L, "set('my_channel', 'not a boolean')"), 0);
    EXPECT_EQ(sink->writes->size(), 0);
}

/// @brief it should return error when writing to nonexistent channel.
TEST_F(SetOperatorTest, ChannelNotFound) {
    SetupChannel(x::telem::FLOAT32_T);
    ASSERT_NE(luaL_dostring(L, "set('nonexistent_channel', 3.14)"), 0);
    EXPECT_EQ(sink->writes->size(), 0);
    // Verify the error message contains the channel name
    const char *error_msg = lua_tostring(L, -1);
    EXPECT_NE(std::string(error_msg).find("nonexistent_channel"), std::string::npos);
    EXPECT_NE(std::string(error_msg).find("not found"), std::string::npos);
}

class SetOperatorWithIndexTest : public testing::Test {
protected:
    void SetupChannels(x::telem::DataType data_type) {
        channels.clear();
        sink = std::make_shared<plugins::mock::FrameSink>();

        // Add index channel
        synnax::channel::Channel::Channel index_ch;
        index_ch.name = "index";
        index_ch.data_type = x::telem::INT64_T;
        index_ch.key = 1;
        index_ch.is_index = true;
        channels.push_back(index_ch);

        // Add value channel
        synnax::channel::Channel::Channel value_ch;
        value_ch.name = "value";
        value_ch.data_type = std::move(data_type);
        value_ch.key = 2;
        value_ch.index = index_ch.key;
        channels.push_back(value_ch);

        op = std::make_unique<plugins::ChannelWrite>(sink, channels);
        L = luaL_newstate();
        luaL_openlibs(L);
        op->before_all(L);
    }

    void TearDown() override {
        lua_close(L);
        channels.clear();
    }

    template<typename T>
    void RunIndexedTest(const char *lua_value, T expected_value) {
        std::string script = "set('value', " + std::string(lua_value) + ")";
        ASSERT_EQ(luaL_dostring(L, script.c_str()), 0) << lua_tostring(L, -1);
        op->after_next(L);
        ASSERT_EQ(sink->writes->size(), 1);

        const x::telem::Series index_ser = std::move(sink->writes->at(0).series->at(1));
        const x::telem::Series value_ser = std::move(sink->writes->at(0).series->at(0));

        EXPECT_GT(index_ser.at<int64_t>(0), 0);
        EXPECT_EQ(value_ser.at<T>(0), expected_value);
    }

    std::shared_ptr<plugins::mock::FrameSink> sink;
    std::vector<synnax::channel::Channel::Channel> channels;
    std::unique_ptr<plugins::ChannelWrite> op;
    lua_State *L{};
};

/// @brief it should write float32 values with automatic index timestamp.
TEST_F(SetOperatorWithIndexTest, Float32ValueWithIndex) {
    SetupChannels(x::telem::FLOAT32_T);
    RunIndexedTest<float>("3.14", 3.14f);
}

/// @brief it should write int32 values with automatic index timestamp.
TEST_F(SetOperatorWithIndexTest, Int32ValueWithIndex) {
    SetupChannels(x::telem::INT32_T);
    RunIndexedTest<int32_t>("42", 42);
}

/// @brief it should write boolean values with automatic index timestamp.
TEST_F(SetOperatorWithIndexTest, BooleanValueWithIndex) {
    SetupChannels(x::telem::UINT8_T);
    RunIndexedTest<uint8_t>("true", 1);
}

class SetAuthorityTest : public testing::Test {
protected:
    void SetUp() override {
        sink = std::make_shared<plugins::mock::FrameSink>();
        // Add three test channels
        synnax::channel::Channel::Channel ch1;
        ch1.name = "channel1";
        ch1.key = 1;
        channels.push_back(ch1);

        synnax::channel::Channel::Channel ch2;
        ch2.name = "channel2";
        ch2.key = 2;
        channels.push_back(ch2);

        synnax::channel::Channel::Channel ch3;
        ch3.name = "channel3";
        ch3.key = 3;
        channels.push_back(ch3);

        op = std::make_unique<plugins::ChannelWrite>(sink, channels);
        L = luaL_newstate();
        luaL_openlibs(L);
        op->before_all(L);
    }

    void TearDown() override {
        lua_close(L);
        channels.clear();
    }

    std::shared_ptr<plugins::mock::FrameSink> sink;
    std::vector<synnax::channel::Channel::Channel> channels;
    std::unique_ptr<plugins::ChannelWrite> op;
    lua_State *L{};
};

/// @brief it should set the authority of all channels.
TEST_F(SetAuthorityTest, SingleAuthForAllChannels) {
    ASSERT_EQ(luaL_dostring(L, "set_authority(42)"), 0);
    ASSERT_EQ(sink->authority_calls.size(), 1);
    const auto &[keys, auths] = sink->authority_calls[0];
    ASSERT_EQ(keys.size(), 3);
    ASSERT_EQ(auths.size(), 3);
    for (const auto &auth: auths)
        EXPECT_EQ(auth, 42);
}

/// @brief it should set authority on a single channel by name.
TEST_F(SetAuthorityTest, SingleChannelAuth) {
    ASSERT_EQ(luaL_dostring(L, "set_authority('channel1', 42)"), 0);
    ASSERT_EQ(sink->authority_calls.size(), 1);
    const auto &[keys, auths] = sink->authority_calls[0];
    ASSERT_EQ(keys.size(), 1);
    ASSERT_EQ(auths.size(), 1);
    EXPECT_EQ(keys[0], 1);
    EXPECT_EQ(auths[0], 42);
}

/// @brief it should set same authority on multiple channels.
TEST_F(SetAuthorityTest, MultipleChannelsSameAuth) {
    ASSERT_EQ(luaL_dostring(L, "set_authority({'channel1', 'channel2'}, 42)"), 0);
    ASSERT_EQ(sink->authority_calls.size(), 1);
    const auto &[keys, auths] = sink->authority_calls[0];
    ASSERT_EQ(keys.size(), 2);
    ASSERT_EQ(auths.size(), 2);
    for (const auto &auth: auths)
        EXPECT_EQ(auth, 42);
}

/// @brief it should set different authorities on multiple channels.
TEST_F(SetAuthorityTest, MultipleChannelsDifferentAuth) {
    ASSERT_EQ(
        luaL_dostring(
            L,
            "set_authority({channel1 = 42, channel2 = 43, channel3 = 44})"
        ),
        0
    );

    ASSERT_EQ(sink->authority_calls.size(), 1);
    const auto &[keys, auths] = sink->authority_calls[0];
    ASSERT_EQ(keys.size(), 3);
    ASSERT_EQ(keys.size(), 3);

    // Create a map of channel keys to their authorities for easier verification
    std::map<synnax::channel::Channel::Key, x::telem::Authority> auth_map;
    for (size_t i = 0; i < keys.size(); i++)
        auth_map[keys[i]] = auths[i];
    EXPECT_EQ(auth_map[1], 42); // channel1
    EXPECT_EQ(auth_map[2], 43); // channel2
    EXPECT_EQ(auth_map[3], 44); // channel3
}

/// @brief it should reject authority set on nonexistent channel.
TEST_F(SetAuthorityTest, InvalidChannelName) {
    ASSERT_NE(luaL_dostring(L, "set_authority('nonexistent', 42)"), 0);
    EXPECT_EQ(sink->authority_calls.size(), 0);
}

/// @brief it should reject invalid arguments to set_authority.
TEST_F(SetAuthorityTest, InvalidArguments) {
    ASSERT_NE(luaL_dostring(L, "set_authority()"), 0);
    ASSERT_NE(luaL_dostring(L, "set_authority('channel1')"), 0);
    ASSERT_NE(luaL_dostring(L, "set_authority('channel1', 'not_a_number')"), 0);
    EXPECT_EQ(sink->authority_calls.size(), 0);
}

/// @brief it should safely handle stop being called before start.
TEST(ChannelWriteLifecycle, StopBeforeStart) {
    auto sink = std::make_shared<plugins::mock::FrameSink>();
    synnax::channel::Channel::Channel ch;
    ch.name = "test_channel";
    ch.key = 1;
    ch.data_type = x::telem::FLOAT64_T;

    auto plugin = plugins::ChannelWrite(sink, std::vector{ch});
    const auto L = luaL_newstate();
    luaL_openlibs(L);

    // Stopping before starting should be safe
    plugin.after_all(L);
    lua_close(L);
}

/// @brief it should safely handle being started twice.
TEST(ChannelWriteLifecycle, DoubleStart) {
    const auto sink = std::make_shared<plugins::mock::FrameSink>();
    synnax::channel::Channel::Channel ch;
    ch.name = "test_channel";
    ch.key = 1;
    ch.data_type = x::telem::FLOAT64_T;

    auto plugin = plugins::ChannelWrite(sink, std::vector{ch});
    const auto L = luaL_newstate();
    luaL_openlibs(L);

    // Starting twice should be safe
    plugin.before_all(L);
    plugin.before_all(L);
    plugin.after_all(L);
    lua_close(L);
}

/// @brief it should safely handle being stopped twice.
TEST(ChannelWriteLifecycle, DoubleStop) {
    const auto sink = std::make_shared<plugins::mock::FrameSink>();
    synnax::channel::Channel::Channel ch;
    ch.name = "test_channel";
    ch.key = 1;
    ch.data_type = x::telem::FLOAT64_T;

    auto plugin = plugins::ChannelWrite(sink, std::vector{ch});
    const auto L = luaL_newstate();
    luaL_openlibs(L);

    plugin.before_all(L);
    plugin.after_all(L);
    plugin.after_all(L);
    lua_close(L);
}
