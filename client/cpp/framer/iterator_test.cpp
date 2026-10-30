// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::framer {
namespace {
/// @brief writes the values at one second apart from start, on the index of data.
void write_seconds(
    Synnax &client,
    const channel::Channel &time,
    const channel::Channel &data,
    const x::telem::TimeStamp start,
    const std::vector<float> &values
) {
    auto writer = ASSERT_NIL_P(client.telem.open_writer(
        WriterConfig{
            channel::keys_from_channels(time, data),
            start,
            std::vector{x::control::AUTHORITY_ABSOLUTE, x::control::AUTHORITY_ABSOLUTE},
            x::control::Subject{"iterator_test_writer"},
        }
    ));
    std::vector<x::telem::TimeStamp> stamps;
    stamps.reserve(values.size());
    for (size_t i = 0; i < values.size(); i++)
        stamps.push_back(start + x::telem::SECOND * static_cast<int64_t>(i));
    auto frame = x::telem::Frame(2);
    frame.emplace(time.key, x::telem::Series(stamps));
    frame.emplace(data.key, x::telem::Series(values));
    ASSERT_NIL(writer.write(frame));
    ASSERT_NIL_P(writer.commit());
    ASSERT_NIL(writer.close());
}
}

/// @brief it should read the stored samples of a channel forward from the first one.
TEST(IteratorTests, testReadsForwardFromFirst) {
    auto client = new_test_client();
    auto [time, data] = create_indexed_pair(client);
    const auto start = x::telem::TimeStamp::now();
    write_seconds(client, time, data, start, {1, 2, 3});
    auto iter = ASSERT_NIL_P(client.telem.open_iterator(
        IteratorConfig{
            .channels = {time.key, data.key},
            .bounds = {start, start + x::telem::SECOND * 10},
        }
    ));
    ASSERT_TRUE(ASSERT_NIL_P(iter.seek_first()));
    ASSERT_TRUE(ASSERT_NIL_P(iter.next()));
    ASSERT_EQ(iter.value.size(), 2);
    ASSERT_EQ(iter.value.at<float>(data.key, 0), 1);
    ASSERT_EQ(iter.value.at<float>(data.key, 2), 3);
    ASSERT_FALSE(ASSERT_NIL_P(iter.next()));
    ASSERT_NIL(iter.close());
}

/// @brief it should read the last stored sample back from the end of the bounds.
TEST(IteratorTests, testReadsBackFromLast) {
    auto client = new_test_client();
    auto [time, data] = create_indexed_pair(client);
    const auto start = x::telem::TimeStamp::now();
    write_seconds(client, time, data, start, {1, 2, 3});
    auto iter = ASSERT_NIL_P(client.telem.open_iterator(
        IteratorConfig{
            .channels = {time.key},
            .bounds = {x::telem::TimeStamp(0), x::telem::TimeStamp::max()},
        }
    ));
    ASSERT_TRUE(ASSERT_NIL_P(iter.seek_last()));
    ASSERT_TRUE(ASSERT_NIL_P(iter.prev()));
    const auto &stamps = iter.value.series->at(0);
    ASSERT_EQ(stamps.at<x::telem::TimeStamp>(-1), start + x::telem::SECOND * 2);
    ASSERT_NIL(iter.close());
}

/// @brief it should read back from a position between two samples that seek_le finds.
TEST(IteratorTests, testReadsBackFromSeekLE) {
    auto client = new_test_client();
    auto [time, data] = create_indexed_pair(client);
    const auto start = x::telem::TimeStamp::now();
    write_seconds(client, time, data, start, {1, 2, 3});
    auto iter = ASSERT_NIL_P(client.telem.open_iterator(
        IteratorConfig{
            .channels = {time.key, data.key},
            .bounds = {start, start + x::telem::SECOND * 10},
        }
    ));
    ASSERT_TRUE(ASSERT_NIL_P(iter.seek_le(start + x::telem::MILLISECOND * 1500)));
    ASSERT_TRUE(ASSERT_NIL_P(iter.prev(x::telem::SECOND * 10)));
    ASSERT_EQ(iter.value.at<float>(data.key, 0), 1);
    ASSERT_EQ(iter.value.at<float>(data.key, -1), 2);
    ASSERT_NIL(iter.close());
}

/// @brief it should read forward from a position between two samples that seek_ge
/// finds.
TEST(IteratorTests, testReadsForwardFromSeekGE) {
    auto client = new_test_client();
    auto [time, data] = create_indexed_pair(client);
    const auto start = x::telem::TimeStamp::now();
    write_seconds(client, time, data, start, {1, 2, 3});
    auto iter = ASSERT_NIL_P(client.telem.open_iterator(
        IteratorConfig{
            .channels = {time.key, data.key},
            .bounds = {start, start + x::telem::SECOND * 10},
        }
    ));
    ASSERT_TRUE(ASSERT_NIL_P(iter.seek_ge(start + x::telem::MILLISECOND * 500)));
    ASSERT_TRUE(ASSERT_NIL_P(iter.next(x::telem::SECOND * 10)));
    ASSERT_EQ(iter.value.at<float>(data.key, 0), 2);
    ASSERT_EQ(iter.value.at<float>(data.key, -1), 3);
    ASSERT_NIL(iter.close());
}

/// @brief it should report a value only after a read at the current position, and keep
/// the value of that read.
TEST(IteratorTests, testValidAfterRead) {
    auto client = new_test_client();
    auto [time, data] = create_indexed_pair(client);
    const auto start = x::telem::TimeStamp::now();
    write_seconds(client, time, data, start, {1, 2, 3});
    auto iter = ASSERT_NIL_P(client.telem.open_iterator(
        IteratorConfig{
            .channels = {time.key, data.key},
            .bounds = {start, start + x::telem::SECOND * 10},
        }
    ));
    ASSERT_TRUE(ASSERT_NIL_P(iter.seek_first()));
    ASSERT_FALSE(ASSERT_NIL_P(iter.valid()));
    ASSERT_TRUE(ASSERT_NIL_P(iter.next()));
    ASSERT_TRUE(ASSERT_NIL_P(iter.valid()));
    ASSERT_EQ(iter.value.at<float>(data.key, 0), 1);
    ASSERT_NIL(iter.close());
}

/// @brief it should find nothing to read in a channel with no stored samples.
TEST(IteratorTests, testFindsNothingInEmptyChannel) {
    auto client = new_test_client();
    auto [time, _] = create_indexed_pair(client);
    auto iter = ASSERT_NIL_P(client.telem.open_iterator(
        IteratorConfig{
            .channels = {time.key},
            .bounds = {x::telem::TimeStamp(0), x::telem::TimeStamp::max()},
        }
    ));
    ASSERT_FALSE(ASSERT_NIL_P(iter.seek_last()));
    ASSERT_FALSE(ASSERT_NIL_P(iter.prev()));
    ASSERT_TRUE(iter.value.empty());
    ASSERT_NIL(iter.close());
}

/// @brief it should fail to open an iterator over a channel that does not exist.
TEST(IteratorTests, testFailsOnUnknownChannel) {
    auto client = new_test_client();
    ASSERT_OCCURRED_AS_P(
        client.telem.open_iterator(
            IteratorConfig{
                .channels = {1 << 30},
                .bounds = {x::telem::TimeStamp(0), x::telem::TimeStamp::max()},
            }
        ),
        x::errors::NOT_FOUND
    );
}
}
