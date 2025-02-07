//
// Created by Emiliano Bonilla on 1/21/25.
//

#include <memory>

#include "driver/sequence/channel_set_operator.h"

#include "gtest/gtest.h"

extern "C" {
#include <lualib.h>
}


class MockSink final : public Sink {
public:
    std::vector<synnax::Frame> written_frames;
    std::vector<std::pair<std::vector<synnax::ChannelKey>, std::vector<synnax::Authority>>> authority_calls;

    freighter::Error write(synnax::Frame& frame) override {
        written_frames.push_back(std::move(frame));
        return freighter::NIL;
    }

    freighter::Error set_authority(const std::vector<synnax::ChannelKey>& keys, 
                                 const std::vector<synnax::Authority>& authorities) override {
        authority_calls.emplace_back(keys, authorities);
        return freighter::NIL;
    }
};

class SetOperatorTest : public testing::Test {
protected:
    void SetupChannel(synnax::DataType data_type) {
        channels.clear();
        sink = std::make_shared<MockSink>();
        
        synnax::Channel ch;
        ch.name = "my_channel";
        ch.data_type = data_type;
        ch.key = 1;
        ch.is_virtual = true;
        channels.push_back(ch);

        op = std::make_unique<SetChannelValueOperator>(sink, channels);
        L = luaL_newstate();
        luaL_openlibs(L);
        op->bind(L);
    }

    void TearDown() override {
        lua_close(L);
        channels.clear();
    }

    template<typename T>
    void RunTest(const char* lua_value, T expected_value) {
        std::string script = "set('my_channel', " + std::string(lua_value) + ")";
        ASSERT_EQ(luaL_dostring(L, script.c_str()), 0) << lua_tostring(L, -1);
        op->flush();
        ASSERT_EQ(sink->written_frames.size(), 1);
        synnax::Series ser = std::move(sink->written_frames[0].series->at(0));
        EXPECT_EQ(ser.at<T>(0), expected_value);
    }

    void RunStringTest(const char* lua_value, const std::string& expected_value) {
        std::string script = "set('my_channel', " + std::string(lua_value) + ")";
        ASSERT_EQ(luaL_dostring(L, script.c_str()), 0) << lua_tostring(L, -1);
        op->flush();
        ASSERT_EQ(sink->written_frames.size(), 1);
        synnax::Series ser = std::move(sink->written_frames[0].series->at(0));
        EXPECT_EQ(ser.at<std::string>(0), expected_value);
    }

    std::shared_ptr<MockSink> sink;
    std::vector<synnax::Channel> channels;
    std::unique_ptr<SetChannelValueOperator> op;
    lua_State* L;
};

TEST_F(SetOperatorTest, Float32Value) {
    SetupChannel(synnax::FLOAT32);
    RunTest<float>("3.14", 3.14f);
}

TEST_F(SetOperatorTest, Float64Value) {
    SetupChannel(synnax::FLOAT64);
    RunTest<double>("3.14159265359", 3.14159265359);
}

TEST_F(SetOperatorTest, Int8Value) {
    SetupChannel(synnax::INT8);
    RunTest<int8_t>("127", 127);
}

TEST_F(SetOperatorTest, Int16Value) {
    SetupChannel(synnax::INT16);
    RunTest<int16_t>("32767", 32767);
}

TEST_F(SetOperatorTest, Int32Value) {
    SetupChannel(synnax::INT32);
    RunTest<int32_t>("2147483647", 2147483647);
}

TEST_F(SetOperatorTest, Int64Value) {
    SetupChannel(synnax::INT64);
    RunTest<int64_t>("9223372036854775807", 9223372036854775807L);
}

TEST_F(SetOperatorTest, Uint8NumberValue) {
    SetupChannel(synnax::SY_UINT8);
    RunTest<uint8_t>("0", 0);
}

TEST_F(SetOperatorTest, Uint8Number1Value) {
    SetupChannel(synnax::SY_UINT8);
    RunTest<uint8_t>("1", 1);
}

TEST_F(SetOperatorTest, Uint8ChannelBooleanValue) {
    SetupChannel(synnax::SY_UINT8);
    RunTest<uint8_t>("false", 0);
}

TEST_F(SetOperatorTest, Uint8ChannelFalseValue) {
    SetupChannel(synnax::SY_UINT8);
    RunTest<uint8_t>("false", 0);
}

TEST_F(SetOperatorTest, UInt16Value) {
    SetupChannel(synnax::SY_UINT16);
    RunTest<uint16_t>("65535", 65535);
}

TEST_F(SetOperatorTest, UInt32Value) {
    SetupChannel(synnax::UINT32);
    RunTest<uint32_t>("4294967295", 4294967295);
}

TEST_F(SetOperatorTest, UInt64Value) {
    SetupChannel(synnax::UINT64);
    RunTest<uint64_t>("18446744073709551615", 18446744073709551615ULL);
}



class SetOperatorWithIndexTest : public testing::Test {
protected:
    void SetupChannels(synnax::DataType data_type) {
        channels.clear();
        sink = std::make_shared<MockSink>();
        
        // Add index channel
        synnax::Channel index_ch;
        index_ch.name = "index";
        index_ch.data_type = synnax::INT64;
        index_ch.key = 1;
        index_ch.is_index = true;
        channels.push_back(index_ch);

        // Add value channel
        synnax::Channel value_ch;
        value_ch.name = "value";
        value_ch.data_type = data_type;
        value_ch.key = 2;
        value_ch.index = index_ch.key;
        channels.push_back(value_ch);

        op = std::make_unique<SetChannelValueOperator>(sink, channels);
        L = luaL_newstate();
        luaL_openlibs(L);
        op->bind(L);
    }

    void TearDown() override {
        lua_close(L);
        channels.clear();
    }

    template<typename T>
    void RunIndexedTest(const char* lua_value, T expected_value) {
        std::string script = "set('value', " + std::string(lua_value) + ")";
        ASSERT_EQ(luaL_dostring(L, script.c_str()), 0) << lua_tostring(L, -1);
        op->flush();
        ASSERT_EQ(sink->written_frames.size(), 1);
        
        synnax::Series index_ser = std::move(sink->written_frames[0].series->at(1));
        synnax::Series value_ser = std::move(sink->written_frames[0].series->at(0));
        
        EXPECT_GT(index_ser.at<int64_t>(0), 0);
        EXPECT_EQ(value_ser.at<T>(0), expected_value);
    }

    std::shared_ptr<MockSink> sink;
    std::vector<synnax::Channel> channels;
    std::unique_ptr<SetChannelValueOperator> op;
    lua_State* L;
};

TEST_F(SetOperatorWithIndexTest, Float32ValueWithIndex) {
    SetupChannels(synnax::FLOAT32);
    RunIndexedTest<float>("3.14", 3.14f);
}

TEST_F(SetOperatorWithIndexTest, Int32ValueWithIndex) {
    SetupChannels(synnax::INT32);
    RunIndexedTest<int32_t>("42", 42);
}

TEST_F(SetOperatorWithIndexTest, BooleanValueWithIndex) {
    SetupChannels(synnax::SY_UINT8);
    RunIndexedTest<uint8_t>("true", 1);
}

class SetAuthorityTest : public testing::Test {
protected:
    void SetUp() override {
        channels.clear();
        sink = std::make_shared<MockSink>();
        
        // Add three test channels
        synnax::Channel ch1;
        ch1.name = "channel1";
        ch1.key = 1;
        channels.push_back(ch1);

        synnax::Channel ch2;
        ch2.name = "channel2";
        ch2.key = 2;
        channels.push_back(ch2);

        synnax::Channel ch3;
        ch3.name = "channel3";
        ch3.key = 3;
        channels.push_back(ch3);

        op = std::make_unique<SetChannelValueOperator>(sink, channels);
        L = luaL_newstate();
        luaL_openlibs(L);
        op->bind(L);
    }

    void TearDown() override {
        lua_close(L);
        channels.clear();
    }

    std::shared_ptr<MockSink> sink;
    std::vector<synnax::Channel> channels;
    std::unique_ptr<SetChannelValueOperator> op;
    lua_State* L;
};

TEST_F(SetAuthorityTest, SingleAuthForAllChannels) {
    ASSERT_EQ(luaL_dostring(L, "set_authority(42)"), 0);
    
    ASSERT_EQ(sink->authority_calls.size(), 1);
    const auto& call = sink->authority_calls[0];
    ASSERT_EQ(call.first.size(), 3);  // All three channels
    ASSERT_EQ(call.second.size(), 3);
    
    // Check all channels got the same authority
    for (const auto& auth : call.second) {
        EXPECT_EQ(auth, 42);
    }
}

TEST_F(SetAuthorityTest, SingleChannelAuth) {
    ASSERT_EQ(luaL_dostring(L, "set_authority('channel1', 42)"), 0);
    
    ASSERT_EQ(sink->authority_calls.size(), 1);
    const auto& call = sink->authority_calls[0];
    ASSERT_EQ(call.first.size(), 1);
    ASSERT_EQ(call.second.size(), 1);
    EXPECT_EQ(call.first[0], 1);  // channel1's key
    EXPECT_EQ(call.second[0], 42);
}

TEST_F(SetAuthorityTest, MultipleChannelsSameAuth) {
    ASSERT_EQ(luaL_dostring(L, "set_authority({'channel1', 'channel2'}, 42)"), 0);
    
    ASSERT_EQ(sink->authority_calls.size(), 1);
    const auto& call = sink->authority_calls[0];
    ASSERT_EQ(call.first.size(), 2);
    ASSERT_EQ(call.second.size(), 2);
    
    // Check both channels got the same authority
    for (const auto& auth : call.second) {
        EXPECT_EQ(auth, 42);
    }
}

TEST_F(SetAuthorityTest, MultipleChannelsDifferentAuth) {
    ASSERT_EQ(luaL_dostring(L, 
        "set_authority({channel1 = 42, channel2 = 43, channel3 = 44})"), 0);
    
    ASSERT_EQ(sink->authority_calls.size(), 1);
    const auto& call = sink->authority_calls[0];
    ASSERT_EQ(call.first.size(), 3);
    ASSERT_EQ(call.second.size(), 3);
    
    // Create a map of channel keys to their authorities for easier verification
    std::map<synnax::ChannelKey, synnax::Authority> auth_map;
    for (size_t i = 0; i < call.first.size(); i++) {
        auth_map[call.first[i]] = call.second[i];
    }
    
    EXPECT_EQ(auth_map[1], 42);  // channel1
    EXPECT_EQ(auth_map[2], 43);  // channel2
    EXPECT_EQ(auth_map[3], 44);  // channel3
}

TEST_F(SetAuthorityTest, InvalidChannelName) {
    ASSERT_NE(luaL_dostring(L, "set_authority('nonexistent', 42)"), 0);
    EXPECT_EQ(sink->authority_calls.size(), 0);
}

TEST_F(SetAuthorityTest, InvalidArguments) {
    ASSERT_NE(luaL_dostring(L, "set_authority()"), 0);
    ASSERT_NE(luaL_dostring(L, "set_authority('channel1')"), 0);
    ASSERT_NE(luaL_dostring(L, "set_authority('channel1', 'not_a_number')"), 0);
    EXPECT_EQ(sink->authority_calls.size(), 0);
}

int main(int argc, char **argv) {
    testing::InitGoogleTest(&argc, argv);
    return RUN_ALL_TESTS();
}