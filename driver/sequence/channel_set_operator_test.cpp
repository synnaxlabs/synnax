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

    freighter::Error write(synnax::Frame& frame) override {
        written_frames.push_back(std::move(frame));
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

        op = std::make_unique<ChannelSetOperator>(sink, channels);
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
    std::unique_ptr<ChannelSetOperator> op;
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

TEST_F(SetOperatorTest, UInt8BooleanValue) {
    SetupChannel(synnax::SY_UINT8);
    RunTest<uint8_t>("true", 1);
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

TEST_F(SetOperatorTest, Uint8ChannelBooleanValue) {
    SetupChannel(synnax::SY_UINT8);
    RunTest<uint8_t>("false", 0);
}

TEST_F(SetOperatorTest, Uint8ChannelFalseValue) {
    SetupChannel(synnax::SY_UINT8);
    RunTest<uint8_t>("false", 0);
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

        op = std::make_unique<ChannelSetOperator>(sink, channels);
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
    std::unique_ptr<ChannelSetOperator> op;
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

int main(int argc, char **argv) {
    testing::InitGoogleTest(&argc, argv);
    return RUN_ALL_TESTS();
}