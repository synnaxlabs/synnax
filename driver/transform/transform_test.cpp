// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

/// local
#include "x/cpp/xtest/xtest.h"

#include "driver/transform/transform.h"

namespace transform {
class MockTransform final : public Transform {
public:
    explicit MockTransform(bool should_fail = false): should_fail_(should_fail) {}

    xerrors::Error transform(telem::Frame &frame) override {
        was_called_ = true;
        if (should_fail_)
            return xerrors::Error(xerrors::INTERNAL, "Mock transform failed");
        return xerrors::NIL;
    }

    [[nodiscard]] bool was_called() const { return was_called_; }

private:
    bool was_called_ = false;
    bool should_fail_;
};

/// @brief it should correctly execute a chain transform.
TEST(TransformTests, ChainTransform) {
    Chain chain;
    const auto mock1 = std::make_shared<MockTransform>();
    const auto mock2 = std::make_shared<MockTransform>();

    chain.add(mock1);
    chain.add(mock2);

    telem::Frame frame;
    ASSERT_NIL(chain.transform(frame));
    ASSERT_TRUE(mock1->was_called());
    ASSERT_TRUE(mock2->was_called());
}

/// @brief it should not call subsequence transforms when a previous transform
/// returns an error.
TEST(TransformTests, ChainTransformFailure) {
    Chain chain;
    const auto mock1 = std::make_shared<MockTransform>();
    const auto mock2 = std::make_shared<MockTransform>(true); // This one will fail
    const auto mock3 = std::make_shared<MockTransform>();
    // This one shouldn't be called

    chain.add(mock1);
    chain.add(mock2);
    chain.add(mock3);

    telem::Frame frame;
    ASSERT_OCCURRED_AS(chain.transform(frame), xerrors::INTERNAL);
    ASSERT_TRUE(mock1->was_called());
    ASSERT_TRUE(mock2->was_called());
    ASSERT_FALSE(mock3->was_called());
}

/// @brief it should do nothing in an empty chain.
TEST(TransformTests, EmptyChain) {
    Chain chain;
    telem::Frame frame;
    ASSERT_NIL(chain.transform(frame));
}

class TareTests : public ::testing::Test {
protected:
    void SetUp() override {
        synnax::Channel ch1;
        ch1.key = 1;
        ch1.name = "test1";
        ch1.data_type = telem::FLOAT64_T;

        synnax::Channel ch2;
        ch2.key = 2;
        ch2.name = "test2";
        ch2.data_type = telem::FLOAT32_T;
        channels = {ch1, ch2};

        frame.reserve(2);
        auto series1 = telem::Series(telem::FLOAT64_T, 2);
        series1.write(10.0);
        series1.write(20.0);
        frame.emplace(1, std::move(series1));

        auto series2 = telem::Series(telem::FLOAT32_T, 2);
        series2.write(5.0f);
        series2.write(15.0f);
        frame.emplace(2, std::move(series2));
    }

    std::vector<synnax::Channel> channels;
    telem::Frame frame;
};

/// @brief it should tare the value of a channel.
TEST_F(TareTests, BasicTare) {
    Tare tare(channels);

    ASSERT_NIL(tare.transform(frame));

    ASSERT_EQ(frame.at<double>(1, -1), 20.0);
    ASSERT_EQ(frame.at<float>(2, -1), 15.0f);

    json tare_args = json::object();
    ASSERT_NIL(tare.tare(tare_args));

    telem::Frame new_frame(2);
    auto new_series1 = telem::Series(telem::FLOAT64_T, 2);
    new_series1.write(30.0);
    new_series1.write(40.0);
    new_frame.emplace(1, std::move(new_series1));

    auto new_series2 = telem::Series(telem::FLOAT32_T, 2);
    new_series2.write(25.0f);
    new_series2.write(35.0f);
    new_frame.emplace(2, std::move(new_series2));

    ASSERT_NIL(tare.transform(new_frame));

    // Using averages: avg1 = 35, avg2 = 30
    ASSERT_EQ(new_frame.at<double>(1, 0), -5.0); // 30 - 35
    ASSERT_EQ(new_frame.at<double>(1, 1), 5.0); // 40 - 35
    ASSERT_EQ(new_frame.at<float>(2, 0), -5.0f); // 25 - 30
    ASSERT_EQ(new_frame.at<float>(2, 1), 5.0f); // 35 - 30
}

/// @brief it should tare only specific channels.
TEST_F(TareTests, TareSpecificChannels) {
    Tare tare(channels);

    ASSERT_NIL(tare.transform(frame));

    // Request tare of only channel 1
    json tare_args = {{"keys", {1}}};
    ASSERT_NIL(tare.tare(tare_args));

    telem::Frame new_frame(2);
    auto new_series1 = telem::Series(telem::FLOAT64_T, 2);
    new_series1.write(30.0);
    new_series1.write(40.0);
    new_frame.emplace(1, std::move(new_series1));

    auto new_series2 = telem::Series(telem::FLOAT32_T, 2);
    new_series2.write(25.0f);
    new_series2.write(35.0f);
    new_frame.emplace(2, std::move(new_series2));

    ASSERT_NIL(tare.transform(new_frame));

    // Only channel 1 should be tared, using average value (35)
    ASSERT_EQ(new_frame.at<double>(1, 0), -5.0); // 30 - 35
    ASSERT_EQ(new_frame.at<double>(1, 1), 5.0); // 40 - 35
    ASSERT_EQ(new_frame.at<float>(2, 0), 25.0f); // Unchanged
    ASSERT_EQ(new_frame.at<float>(2, 1), 35.0f); // Unchanged

    // Subsequent frame should use same tare values
    telem::Frame third_frame(2);
    auto third_series1 = telem::Series(telem::FLOAT64_T, 2);
    third_series1.write(50.0);
    third_series1.write(60.0);
    third_frame.emplace(1, std::move(third_series1));

    auto third_series2 = telem::Series(telem::FLOAT32_T, 2);
    third_series2.write(45.0f);
    third_series2.write(55.0f);
    third_frame.emplace(2, std::move(third_series2));

    ASSERT_NIL(tare.transform(third_frame));
    ASSERT_EQ(third_frame.at<double>(1, 0), 15.0); // 50 - 35
    ASSERT_EQ(third_frame.at<double>(1, 1), 25.0); // 60 - 35
    ASSERT_EQ(third_frame.at<float>(2, 0), 45.0f); // Unchanged
    ASSERT_EQ(third_frame.at<float>(2, 1), 55.0f); // Unchanged
}

/// @brief it should return an error when the channel key is invalid.
TEST_F(TareTests, InvalidChannelKey) {
    Tare tare(channels);

    ASSERT_NIL(tare.transform(frame));

    json tare_args = {{"keys", {999}}};
    const auto err = tare.tare(tare_args);
    ASSERT_TRUE(err);
}

/// @brief it should correctly apply a linear scale to a channel
TEST(ScaleTests, LinearScale) {
    json config = {
        {"channels",
         {{{"channel", 1},
           {"scale", {{"type", "linear"}, {"slope", 2.0}, {"offset", 5.0}}}}}}
    };

    // Create channel map
    std::unordered_map<synnax::ChannelKey, synnax::Channel> channels;
    synnax::Channel ch1;
    ch1.key = 1;
    ch1.data_type = telem::FLOAT64_T;
    channels[1] = ch1;

    xjson::Parser parser(config);
    Scale scale(parser, channels);

    telem::Frame frame(1);
    auto series = telem::Series(telem::FLOAT64_T, 2);
    series.write(10.0);
    series.write(20.0);
    frame.emplace(1, std::move(series));

    ASSERT_NIL(scale.transform(frame));

    ASSERT_EQ(frame.at<double>(1, 0), 25.0); // 10 * 2 + 5
    ASSERT_EQ(frame.at<double>(1, 1), 45.0); // 20 * 2 + 5
}

/// @brief it should properly apply a map scale to a channel.
TEST(ScaleTests, MapScale) {
    json config = {
        {"channels",
         {{{"channel", 1},
           {"scale",
            {{"type", "map"},
             {"pre_scaled_min", 0.0},
             {"pre_scaled_max", 100.0},
             {"scaled_min", 0.0},
             {"scaled_max", 1.0}}}}}}
    };

    std::unordered_map<synnax::ChannelKey, synnax::Channel> channels;
    synnax::Channel ch1;
    ch1.key = 1;
    ch1.data_type = telem::FLOAT64_T;
    channels[1] = ch1;

    xjson::Parser parser(config);
    Scale scale(parser, channels);

    telem::Frame frame(1);
    auto series = telem::Series(telem::FLOAT64_T, 3);
    series.write(0.0);
    series.write(50.0);
    series.write(100.0);
    frame.emplace(1, std::move(series));

    ASSERT_NIL(scale.transform(frame));

    ASSERT_NEAR(frame.at<double>(1, 0), 0.0, 0.001);
    ASSERT_NEAR(frame.at<double>(1, 1), 0.5, 0.001);
    ASSERT_NEAR(frame.at<double>(1, 2), 1.0, 0.001);
}

/// @brief it should correctly apply a scale to multiple channels.
TEST(ScaleTests, MultipleChannels) {
    json config = {
        {"channels",
         {{{"channel", 1},
           {"scale", {{"type", "linear"}, {"slope", 2.0}, {"offset", 0.0}}}},
          {{"channel", 2},
           {"scale",
            {{"type", "map"},
             {"pre_scaled_min", 0.0},
             {"pre_scaled_max", 10.0},
             {"scaled_min", 0.0},
             {"scaled_max", 100.0}}}}}}
    };

    std::unordered_map<synnax::ChannelKey, synnax::Channel> channels;
    synnax::Channel ch1;
    ch1.key = 1;
    ch1.data_type = telem::FLOAT64_T;
    channels[1] = ch1;

    synnax::Channel ch2;
    ch2.key = 2;
    ch2.data_type = telem::FLOAT64_T;
    channels[2] = ch2;

    xjson::Parser parser(config);
    Scale scale(parser, channels);

    telem::Frame frame(2);
    auto series1 = telem::Series(telem::FLOAT64_T, 1);
    series1.write(5.0);
    frame.emplace(1, std::move(series1));
    auto series2 = telem::Series(telem::FLOAT64_T, 1);
    series2.write(5.0);
    frame.emplace(2, std::move(series2));
    ASSERT_NIL(scale.transform(frame));
    ASSERT_EQ(frame.at<double>(1, 0), 10.0); // Linear: 5 * 2 + 0
    ASSERT_EQ(frame.at<double>(2, 0),
              50.0); // Map: (5 - 0) / (10 - 0) * (100 - 0) + 0
}

/// @brief it should correctly ignore channels that are not configured for scaling.
TEST(ScaleTests, IgnoreUnknownChannels) {
    json config = {
        {"channels",
         {{{"channel", 1},
           {"scale", {{"type", "linear"}, {"slope", 2.0}, {"offset", 0.0}}}}}}
    };

    std::unordered_map<synnax::ChannelKey, synnax::Channel> channels;
    synnax::Channel ch1;
    ch1.key = 1;
    ch1.data_type = telem::FLOAT64_T;
    channels[1] = ch1;

    xjson::Parser parser(config);
    Scale scale(parser, channels);

    telem::Frame frame(2);

    auto series1 = telem::Series(telem::FLOAT64_T, 1);
    series1.write(5.0);
    frame.emplace(1, std::move(series1));

    auto series2 = telem::Series(telem::FLOAT64_T, 1);
    series2.write(5.0);
    frame.emplace(2, std::move(series2));

    ASSERT_NIL(scale.transform(frame));

    ASSERT_EQ(frame.at<double>(1, 0), 10.0); // Scaled: 5 * 2 + 0
    ASSERT_EQ(frame.at<double>(2, 0), 5.0); // Unchanged
}

/// @brief it should correctly ignore disabled channels.
TEST(ScaleTests, DisabledChannel) {
    json config = {
        {"channels",
         {{{"channel", 1},
           {"enabled", true},
           {"scale", {{"type", "linear"}, {"slope", 2.0}, {"offset", 5.0}}}},
          {{"channel", 2},
           {"enabled", false},
           {"scale", {{"type", "linear"}, {"slope", 3.0}, {"offset", 10.0}}}}}}
    };

    std::unordered_map<synnax::ChannelKey, synnax::Channel> channels;

    synnax::Channel ch1;
    ch1.key = 1;
    ch1.data_type = telem::FLOAT64_T;
    channels[1] = ch1;

    xjson::Parser parser(config);
    Scale scale(parser, channels);

    telem::Frame frame(2);

    auto series1 = telem::Series(telem::FLOAT64_T, 1);
    series1.write(10.0);
    frame.emplace(1, std::move(series1));

    auto series2 = telem::Series(telem::FLOAT64_T, 1);
    series2.write(10.0);
    frame.emplace(2, std::move(series2));

    ASSERT_NIL(scale.transform(frame));

    ASSERT_EQ(frame.at<double>(1, 0), 25.0); // Enabled: 10 * 2 + 5
    ASSERT_EQ(frame.at<double>(2, 0), 10.0); // Disabled: unchanged
}

/// @brief it should apply transformations directly to the frame.
TEST(ScaleTests, TransformInplaceUsage) {
    json config = {
        {"channels",
         {{{"channel", 1},
           {"scale", {{"type", "linear"}, {"slope", 3.0}, {"offset", 2.0}}}}}}
    };

    std::unordered_map<synnax::ChannelKey, synnax::Channel> channels;
    synnax::Channel ch1;
    ch1.key = 1;
    ch1.data_type = telem::FLOAT64_T;
    channels[1] = ch1;

    xjson::Parser parser(config);
    Scale scale(parser, channels);

    telem::Frame frame(3);

    auto series1 = telem::Series(telem::FLOAT64_T, 2);
    series1.write(1.0);
    series1.write(2.0);
    frame.emplace(1, std::move(series1));

    auto series2 = telem::Series(telem::INT32_T, 2);
    series2.write(10);
    series2.write(20);
    frame.emplace(2, std::move(series2));

    auto series3 = telem::Series(telem::FLOAT32_T, 2);
    series3.write(1.5f);
    series3.write(2.5f);
    frame.emplace(3, std::move(series3));

    ASSERT_NIL(scale.transform(frame));
    ASSERT_EQ(frame.at<double>(1, 0), 5.0); // 1.0 * 3.0 + 2.0
    ASSERT_EQ(frame.at<double>(1, 1), 8.0); // 2.0 * 3.0 + 2.0

    ASSERT_EQ(frame.at<int32_t>(2, 0), 10);
    ASSERT_EQ(frame.at<int32_t>(2, 1), 20);
    ASSERT_EQ(frame.at<float>(3, 0), 1.5f);
    ASSERT_EQ(frame.at<float>(3, 1), 2.5f);
}

// @brief it should correctly tare channels with different data types.
TEST_F(TareTests, TareWithDifferentDataTypes) {
    std::vector<synnax::Channel> channels;

    synnax::Channel ch1;
    ch1.key = 1;
    ch1.name = "int32";
    ch1.data_type = telem::INT32_T;

    synnax::Channel ch2;
    ch2.key = 2;
    ch2.name = "float32";
    ch2.data_type = telem::FLOAT32_T;

    synnax::Channel ch3;
    ch3.key = 3;
    ch3.name = "float64";
    ch3.data_type = telem::FLOAT64_T;

    channels = {ch1, ch2, ch3};

    Tare tare(channels);

    telem::Frame frame(3);
    auto series1 = telem::Series(telem::INT32_T, 2);
    series1.write(100);
    series1.write(200);
    frame.emplace(1, std::move(series1));

    auto series2 = telem::Series(telem::FLOAT32_T, 2);
    series2.write(10.5f);
    series2.write(20.5f);
    frame.emplace(2, std::move(series2));

    auto series3 = telem::Series(telem::FLOAT64_T, 2);
    series3.write(1000.25);
    series3.write(2000.25);
    frame.emplace(3, std::move(series3));

    ASSERT_NIL(tare.transform(frame));

    json tare_args = json::object();
    ASSERT_NIL(tare.tare(tare_args));

    telem::Frame new_frame(3);
    auto new_series1 = telem::Series(telem::INT32_T, 2);
    new_series1.write(300);
    new_series1.write(400);
    new_frame.emplace(1, std::move(new_series1));

    auto new_series2 = telem::Series(telem::FLOAT32_T, 2);
    new_series2.write(30.5f);
    new_series2.write(40.5f);
    new_frame.emplace(2, std::move(new_series2));

    auto new_series3 = telem::Series(telem::FLOAT64_T, 2);
    new_series3.write(3000.25);
    new_series3.write(4000.25);
    new_frame.emplace(3, std::move(new_series3));

    ASSERT_NIL(tare.transform(new_frame));

    // Values should be tared using averages from this frame
    // avg1 = 350, avg2 = 35.5, avg3 = 3500.25
    ASSERT_EQ(new_frame.at<int32_t>(1, 0), -50); // 300 - 350
    ASSERT_EQ(new_frame.at<int32_t>(1, 1), 50); // 400 - 350
    ASSERT_EQ(new_frame.at<float>(2, 0), -5.0f); // 30.5 - 35.5
    ASSERT_EQ(new_frame.at<float>(2, 1), 5.0f); // 40.5 - 35.5
    ASSERT_EQ(new_frame.at<double>(3, 0), -500.0); // 3000.25 - 3500.25
    ASSERT_EQ(new_frame.at<double>(3, 1), 500.0); // 4000.25 - 3500.25

    // Test subsequent frame with same tare values
    telem::Frame third_frame(3);
    auto third_series1 = telem::Series(telem::INT32_T, 2);
    third_series1.write(500);
    third_series1.write(600);
    third_frame.emplace(1, std::move(third_series1));

    auto third_series2 = telem::Series(telem::FLOAT32_T, 2);
    third_series2.write(50.5f);
    third_series2.write(60.5f);
    third_frame.emplace(2, std::move(third_series2));

    auto third_series3 = telem::Series(telem::FLOAT64_T, 2);
    third_series3.write(5000.25);
    third_series3.write(6000.25);
    third_frame.emplace(3, std::move(third_series3));

    ASSERT_NIL(tare.transform(third_frame));
    ASSERT_EQ(third_frame.at<int32_t>(1, 0), 150); // 500 - 350
    ASSERT_EQ(third_frame.at<int32_t>(1, 1), 250); // 600 - 350
    ASSERT_EQ(third_frame.at<float>(2, 0), 15.0f); // 50.5 - 35.5
    ASSERT_EQ(third_frame.at<float>(2, 1), 25.0f); // 60.5 - 35.5
    ASSERT_EQ(third_frame.at<double>(3, 0), 1500.0); // 5000.25 - 3500.25
    ASSERT_EQ(third_frame.at<double>(3, 1), 2500.0); // 6000.25 - 3500.25
}

/// @brief it should correctly execute a chain with a tare and scale transform.
TEST(ChainTests, ComplexTransformChain) {
    std::vector<synnax::Channel> channels;

    synnax::Channel ch1;
    ch1.key = 1;
    ch1.name = "test";
    ch1.data_type = telem::FLOAT64_T;

    channels = {ch1};

    auto tare = std::make_shared<Tare>(channels);

    json config = {
        {"channels",
         {{{"channel", 1},
           {"scale", {{"type", "linear"}, {"slope", 2.0}, {"offset", 10.0}}}}}}
    };

    std::unordered_map<synnax::ChannelKey, synnax::Channel> channel_map;
    channel_map[1] = ch1;

    xjson::Parser parser(config);
    auto scale = std::make_shared<Scale>(parser, channel_map);

    Chain chain;
    chain.add(tare);
    chain.add(scale);

    json tare_args = json::object();
    ASSERT_NIL(tare->tare(tare_args));

    telem::Frame frame(1);
    auto series = telem::Series(telem::FLOAT64_T, 1);
    series.write(50.0);
    frame.emplace(1, std::move(series));

    ASSERT_NIL(chain.transform(frame));

    // Create second frame
    telem::Frame frame2(1);
    auto series2 = telem::Series(telem::FLOAT64_T, 1);
    series2.write(70.0);
    frame2.emplace(1, std::move(series2));

    // Second pass through the chain
    // First tare will subtract 50, then scale will multiply by 2 and add 10
    ASSERT_NIL(chain.transform(frame2));

    // Check the result: (70 - 50) * 2 + 10 = 50
    ASSERT_EQ(frame2.at<double>(1, 0), 50.0);
}
}
