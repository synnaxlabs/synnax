//
// Created by Emiliano Bonilla on 1/18/25.
//

#include "gtest/gtest.h"
#include "driver/sequence/sequence.h"

class MockSource : public Source {
public:
    synnax::Frame read() override {
        auto frame = synnax::Frame(2);  // Frame with 2 channels
        frame.add(1, synnax::Series(2.0f));  // Channel "a" = 2.0
        frame.add(2, synnax::Series(3.0f));  // Channel "b" = 3.0
        return frame;
    }
};

class MockSink : public Sink {
public:
    freighter::Error write(synnax::Frame frame) override {
        last_frame = std::move(frame);
        return freighter::Error();
    }

    synnax::Frame last_frame;
};

TEST(SequenceTests, TestMultiplication) {
    std::cout << "DOG" << std::endl;
    // Create channels map
    std::unordered_map<synnax::ChannelKey, synnax::Channel> channels = {
        {1, synnax::Channel("a", synnax::FLOAT32)},
        {2, synnax::Channel("b", synnax::FLOAT32)},
        {3, synnax::Channel("result", synnax::FLOAT32)}
    };

    // Create source and sink
    auto source = std::make_shared<MockSource>();
    auto sink = std::make_shared<MockSink>();

    // Lua script to multiply values - using direct variable access
    std::string script = R"(
        set("result", a * b)
    )";

    // Create and run sequence
    auto seq = Sequence(synnax::Rate(1), source, sink, channels, script);
    seq.main(1);

    auto val = sink->last_frame.at<double>(3, 0);
    EXPECT_FLOAT_EQ(val, 6.0f);  // 2.0 * 3.0 = 6.0

    // Check result
    // ASSERT_TRUE(sink->last_frame.contains(3));  // Check if result channel exists
    // EXPECT_FLOAT_EQ(sink->last_frame[3].value<float>(), 6.0f);  // 2.0 * 3.0 = 6.0
}

int main(int argc, char **argv) {
    testing::InitGoogleTest(&argc, argv);
    return RUN_ALL_TESTS();
}