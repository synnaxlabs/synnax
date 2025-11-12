// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/telem/frame.h"

/// @brief it should construct a frame with a pre-allocated size.
TEST(FrameTests, testConstructionFromSize) {
    const auto f = telem::Frame(2);
    auto s = telem::Series(std::vector<float>{1, 2, 3});
    f.emplace(65537, std::move(s));
    ASSERT_EQ(f.size(), 1);
}

/// @brief it should construct a frame from a single series and channel.
TEST(FrameTests, testConstructionFromSingleSeriesAndChannel) {
    const auto f = telem::Frame(65537, telem::Series(std::vector<float>{1, 2, 3}));
    ASSERT_EQ(f.size(), 1);
    ASSERT_EQ(f.channels->at(0), 65537);
    ASSERT_EQ(f.length(), 3);
    ASSERT_EQ(f.series->at(0).data_type(), telem::FLOAT32_T);
    ASSERT_EQ(f.series->at(0).values<float>()[0], 1);
}

/// @brief it should construct a frame from a proto.
TEST(FrameTests, toProto) {
    const auto f = telem::Frame(2);
    auto s = telem::Series(std::vector<float>{1, 2, 3});
    f.emplace(65537, std::move(s));
    telem::PBFrame p;
    f.to_proto(&p);
    ASSERT_EQ(p.keys_size(), 1);
    ASSERT_EQ(p.series_size(), 1);
    const auto f2 = telem::Frame(p);
    ASSERT_EQ(f2.size(), 1);
    ASSERT_EQ(f2.channels->at(0), 65537);
    ASSERT_EQ(f2.series->at(0).values<float>()[0], 1);
}

/// @brief test ostream operator.
TEST(FrameTests, ostream) {
    const auto f = telem::Frame(2);
    auto s = telem::Series(std::vector<float>{1, 2, 3});
    f.emplace(65537, std::move(s));
    std::stringstream ss;
    ss << f;
    ASSERT_EQ(
        ss.str(),
        "Frame{\n 65537: Series(type: float32, size: 3, cap: 3, data: [1 2 3 ]), \n}"
    );
}

/// @brief it should correctly clear the frame for reuse.
TEST(FrameTests, testClear) {
    const auto f = telem::Frame(2);
    auto s = telem::Series(std::vector<float>{1, 2, 3});
    f.emplace(65537, std::move(s));
    f.clear();
    ASSERT_EQ(f.size(), 0);
    ASSERT_EQ(f.channels->size(), 0);
    ASSERT_EQ(f.series->size(), 0);
}

/// @brief it should correctly add a series to the frame.
TEST(FrameTests, testReserve) {
    auto f = telem::Frame(2);
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

/// @brief it should deep copy the frame.
TEST(FrameTests, testDeepCopy) {
    const auto f = telem::Frame(2);
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

/// @brief it should iterate through the frames keys and values.
TEST(FrameTests, testIteration) {
    auto frame = telem::Frame(3);

    frame.emplace(65537, telem::Series(std::vector{1.0f, 2.0f, 3.0f}));
    frame.emplace(65538, telem::Series(std::vector{4.0, 5.0, 6.0}));
    frame.emplace(65539, telem::Series(std::vector{7, 8, 9}));

    ASSERT_EQ(frame.size(), 3);

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

    count = 0;
    std::set<std::uint32_t> seen_keys;
    for (auto [key, series]: frame) {
        count++;
        seen_keys.insert(key);

        if (key == 65537) {
            ASSERT_EQ(series.data_type(), telem::FLOAT32_T);
            std::vector<float> values = series.values<float>();
            ASSERT_EQ(values[0], 1.0f);
            series.set<float>(0, 10.0f);
        }
    }
    ASSERT_EQ(count, 3);
    ASSERT_EQ(seen_keys.size(), 3);
    ASSERT_TRUE(seen_keys.find(65537) != seen_keys.end());
    ASSERT_TRUE(seen_keys.find(65538) != seen_keys.end());
    ASSERT_TRUE(seen_keys.find(65539) != seen_keys.end());

    const auto &const_frame = frame;
    count = 0;
    for (auto [key, s]: const_frame) {
        count++;
        if (key == 65537) {
            ASSERT_EQ(s.at<float>(0), 10.0f); // Should see the modified value
        }
    }
    ASSERT_EQ(count, 3);

    auto empty_frame = telem::Frame(0);
    count = 0;
    for ([[maybe_unused]] auto pair: empty_frame)
        count++;
    ASSERT_EQ(count, 0);
    ASSERT_TRUE(empty_frame.begin() == empty_frame.end());
}

TEST(FrameTests, testIteratorWithSTLAlgorithms) {
    const auto frame = telem::Frame(3);
    frame.emplace(65537, telem::Series(std::vector{1.0f, 2.0f, 3.0f}));
    frame.emplace(65538, telem::Series(std::vector{4.0, 5.0, 6.0}));
    frame.emplace(65539, telem::Series(std::vector{7, 8, 9}));

    const auto it = std::find_if(frame.begin(), frame.end(), [](const auto &pair) {
        return pair.first == 65538;
    });

    ASSERT_NE(it, frame.end());
    auto [key, s] = *it;
    ASSERT_EQ(key, 65538);
    ASSERT_EQ(s.data_type(), telem::FLOAT64_T);
    ASSERT_EQ(s.values<double>()[0], 4.0);

    const auto count = std::count_if(frame.begin(), frame.end(), [](const auto &p) {
        return p.first > 65537;
    });
    ASSERT_EQ(count, 2);

    std::vector<std::uint32_t> keys;
    std::for_each(frame.begin(), frame.end(), [&keys](const auto &p) {
        keys.push_back(p.first);
    });

    ASSERT_EQ(keys.size(), 3);
    ASSERT_EQ(keys[0], 65537);
    ASSERT_EQ(keys[1], 65538);
    ASSERT_EQ(keys[2], 65539);
}
