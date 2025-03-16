// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <include/gtest/gtest.h>
#include "client/cpp/framer/framer.h"

/// @brief it should construct a frame with a pre-allocated size.
TEST(FrameTests, testConstructionFromSize) {
    const auto f = synnax::Frame(2);
    auto s = telem::Series(std::vector<float>{1, 2, 3});
    f.emplace(65537, std::move(s));
    ASSERT_EQ(f.size(), 1);
}

/// @brief it should construct a frame from a single series and channel.
TEST(FrameTests, testConstructionFromSingleSeriesAndChannel) {
    const auto f = synnax::Frame(65537, telem::Series(std::vector<float>{1, 2, 3}));
    ASSERT_EQ(f.size(), 1);
    ASSERT_EQ(f.channels->at(0), 65537);
    ASSERT_EQ(f.length(), 3);
    ASSERT_EQ(f.series->at(0).data_type(), telem::FLOAT32_T);
    ASSERT_EQ(f.series->at(0).values<float>()[0], 1);
}

/// @brief it should construct a frame from a proto.
TEST(FrameTests, toProto) {
    const auto f = synnax::Frame(2);
    auto s = telem::Series(std::vector<float>{1, 2, 3});
    f.emplace(65537, std::move(s));
    const auto p = new api::v1::Frame();
    f.to_proto(p);
    ASSERT_EQ(p->keys_size(), 1);
    ASSERT_EQ(p->series_size(), 1);
    const auto f2 = synnax::Frame(*p);
    ASSERT_EQ(f2.size(), 1);
    ASSERT_EQ(f2.channels->at(0), 65537);
    ASSERT_EQ(f2.series->at(0).values<float>()[0], 1);
}

/// @brief test ostream operator.
TEST(FrameTests, ostream) {
    const auto f = synnax::Frame(2);
    auto s = telem::Series(std::vector<float>{1, 2, 3});
    f.emplace(65537, std::move(s));
    std::stringstream ss;
    ss << f;
    ASSERT_EQ(ss.str(),
              "Frame{\n 65537: Series(type: float32, size: 3, cap: 3, data: [1 2 3 ]), \n}");
}

TEST(FrameTests, testClear) {
    const auto f = synnax::Frame(2);
    auto s = telem::Series(std::vector<float>{1, 2, 3});
    f.emplace(65537, std::move(s));
    f.clear();
    ASSERT_EQ(f.size(), 0);
    ASSERT_EQ(f.channels->size(), 0);
    ASSERT_EQ(f.series->size(), 0); 
}

TEST(FrameTests, testReserve) {
    auto f = synnax::Frame(2);
    f.reserve(10);
    ASSERT_EQ(f.size(), 0);
    ASSERT_EQ(f.channels->size(), 0);
    ASSERT_EQ(f.series->size(), 0);
    f.emplace(65537, telem::Series(std::vector<float>{1, 2, 3}));
    ASSERT_EQ(f.size(), 1);
    ASSERT_EQ(f.channels->size(), 1);
    ASSERT_EQ(f.series->size(), 1);
    f.reserve(10);
    ASSERT_EQ(f.size(), 1);
    ASSERT_EQ(f.capacity(), 10);
    ASSERT_EQ(f.channels->capacity(), 10);
    ASSERT_EQ(f.series->capacity(), 10);

}

TEST(FrameTests, testDeepCopy) {
    const auto f = synnax::Frame(2);
    auto s = telem::Series(std::vector<float>{1, 2, 3});
    f.emplace(65537, std::move(s));
    const auto f2 = f.deep_copy();
    ASSERT_EQ(f2.size(), 1);
    ASSERT_EQ(f2.channels->at(0), 65537);
    ASSERT_EQ(f2.series->at(0).values<float>()[0], 1);
    f.emplace(65538, telem::Series(std::vector<float>{4, 5, 6}));
    ASSERT_EQ(f.size(), 2);
    ASSERT_EQ(f2.size(), 1);
    ASSERT_EQ(f2.channels->at(0), 65537);
    ASSERT_EQ(f2.series->at(0).values<float>()[0], 1);
}

TEST(FrameTests, testIteration) {
    // Create a frame with multiple channel-series pairs
    auto frame = synnax::Frame(3);
    
    // Add first channel-series pair
    frame.emplace(65537, telem::Series(std::vector<float>{1.0f, 2.0f, 3.0f}));
    
    // Add second channel-series pair
    frame.emplace(65538, telem::Series(std::vector<double>{4.0, 5.0, 6.0}));
    
    // Add third channel-series pair
    frame.emplace(65539, telem::Series(std::vector<int32_t>{7, 8, 9}));
    
    // Test size before iteration
    ASSERT_EQ(frame.size(), 3);
    
    // Test manual iteration with begin/end
    size_t count = 0;
    for (auto it = frame.begin(); it != frame.end(); ++it) {
        auto [key, series] = *it;
        count++;
        if (key == 65537) {
            ASSERT_EQ(series.data_type(), telem::FLOAT32_T);
            ASSERT_EQ(series.at<float>(0), 1.0f);
            ASSERT_EQ(series.at<float>(1), 2.0f);
            ASSERT_EQ(series.at<float>(2), 3.0f);
        } else if (key == 65538) {
            ASSERT_EQ(series.data_type(), telem::FLOAT64_T);
            ASSERT_EQ(series.at<double>(0), 4.0);
            ASSERT_EQ(series.at<double>(1), 5.0);
            ASSERT_EQ(series.at<double>(2), 6.0);
        } else if (key == 65539) {
            ASSERT_EQ(series.data_type(), telem::INT32_T);
            ASSERT_EQ(series.at<int32_t>(0), 7);
            ASSERT_EQ(series.at<int32_t>(1), 8);
            ASSERT_EQ(series.at<int32_t>(2), 9);
        } else {
            FAIL() << "Unexpected channel key: " << key;
        }
    }
    ASSERT_EQ(count, 3);
    
    // Test range-based for loop
    count = 0;
    std::set<ChannelKey> seen_keys;
    for (auto [key, series] : frame) {
        count++;
        seen_keys.insert(key);

        // Verify we can access and modify the series through the iterator
        if (key == 65537) {
            ASSERT_EQ(series.data_type(), telem::FLOAT32_T);
            std::vector<float> values = series.values<float>();
            ASSERT_EQ(values[0], 1.0f);

            // Modify the series through the iterator
            series.set<float>(0, 10.0f);
        }
    }
    ASSERT_EQ(count, 3);
    ASSERT_EQ(seen_keys.size(), 3);
    ASSERT_TRUE(seen_keys.find(65537) != seen_keys.end());
    ASSERT_TRUE(seen_keys.find(65538) != seen_keys.end());
    ASSERT_TRUE(seen_keys.find(65539) != seen_keys.end());
    
    // Test iteration with const frame
    const auto& const_frame = frame;
    count = 0;
    for (auto [key, s] : const_frame) {
        count++;
        if (key == 65537)
            ASSERT_EQ(s.at<float>(0), 10.0f);  // Should see the modified value
    }
    ASSERT_EQ(count, 3);
    
    // Test empty frame iteration
    auto empty_frame = synnax::Frame(0);
    count = 0;
    for (auto pair : empty_frame) count++;
    ASSERT_EQ(count, 0);
    ASSERT_TRUE(empty_frame.begin() == empty_frame.end());
}

TEST(FrameTests, testIteratorWithSTLAlgorithms) {
    const auto frame = synnax::Frame(3);
    frame.emplace(65537, telem::Series(std::vector<float>{1.0f, 2.0f, 3.0f}));
    frame.emplace(65538, telem::Series(std::vector<double>{4.0, 5.0, 6.0}));
    frame.emplace(65539, telem::Series(std::vector<int32_t>{7, 8, 9}));
    
    // Test std::find_if
    const auto it = std::find_if(frame.begin(), frame.end(),
        [](const auto& pair) { return pair.first == 65538; });
    
    ASSERT_NE(it, frame.end());
    auto [key, s] = *it;
    ASSERT_EQ(key, 65538);
    ASSERT_EQ(s.data_type(), telem::FLOAT64_T);
    ASSERT_EQ(s.values<double>()[0], 4.0);
    
    const auto count = std::count_if(frame.begin(), frame.end(),
        [](const auto& p) { return p.first > 65537; });
    ASSERT_EQ(count, 2);
    
    std::vector<ChannelKey> keys;
    std::for_each(frame.begin(), frame.end(), 
        [&keys](const auto& p) { keys.push_back(p.first); });
    
    ASSERT_EQ(keys.size(), 3);
    ASSERT_EQ(keys[0], 65537);
    ASSERT_EQ(keys[1], 65538);
    ASSERT_EQ(keys[2], 65539);
}