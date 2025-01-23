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

TEST(SetOperatorTest, Basic) {
    auto sink = std::make_shared<MockSink>();
    
    std::unordered_map<std::string, synnax::Channel> channels;
    synnax::Channel ch;
    ch.name = "my_channel";
    ch.data_type = synnax::FLOAT64;
    ch.key = 1;
    channels["my_channel"] = ch;
    ChannelSetOperator op(sink, channels);
    lua_State* L = luaL_newstate();
    luaL_openlibs(L);
    op.bind(L);
    const char* script = "set('my_channel', 1)";

    ASSERT_EQ(luaL_dostring(L, script), 0) << lua_tostring(L, -1);
    op.flush();
    ASSERT_EQ(sink->written_frames.size(), 1);
}

int main(int argc, char **argv) {
    testing::InitGoogleTest(&argc, argv);
    return RUN_ALL_TESTS();
}