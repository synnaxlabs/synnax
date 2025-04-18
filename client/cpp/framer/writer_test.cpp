// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std
#include <thread>

/// external
#include "gtest/gtest.h"

/// module
#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/xtest/xtest.h"


/// @brief it should correctly write a frame of telemetry to the DB.
TEST(WriterTests, testWriteBasic) {
    auto client = new_test_client();
    auto [time, data] = create_indexed_pair(client);
    auto now = telem::TimeStamp::now();
    auto writer = ASSERT_NIL_P(client.telem.open_writer(synnax::WriterConfig{
        synnax::keys_from_channels(time, data),
        now,
        std::vector{telem::AUTH_ABSOLUTE, telem::AUTH_ABSOLUTE},
        telem::ControlSubject{"test_writer"},
    }));

    auto frame = synnax::Frame(2);
    frame.emplace(
        time.key,
        telem::Series(std::vector{
            (now + telem::SECOND),
            (now + telem::SECOND * 2),
            (now + telem::SECOND * 3),
            (now + telem::SECOND * 4),
            (now + telem::SECOND * 5),
            (now + telem::SECOND * 6),
            (now + telem::SECOND * 7),
            (now + telem::SECOND * 8),
        })
    );
    frame.emplace(
        data.key,
        telem::Series(std::vector<float>{2, 3, 4, 5, 6, 7, 8, 9})
    );


    ASSERT_NIL(writer.write(frame));
    auto end = ASSERT_NIL_P(writer.commit());
    ASSERT_EQ(end, now + (telem::SECOND * 8 + 1));
    ASSERT_NIL(writer.close());
}

TEST(WriterTests, testOpenWriterOnNonexistentChannel) {
    auto client = new_test_client();
    auto [time, data] = create_indexed_pair(client);
    const auto now = telem::TimeStamp::now();
    ASSERT_OCCURRED_AS_P(
        client.telem.open_writer(synnax::WriterConfig{
            std::vector<synnax::ChannelKey>{time.key, 1000},
            now,
            std::vector{telem::AUTH_ABSOLUTE},
            telem::ControlSubject{"test_writer"},
        }),
        xerrors::NOT_FOUND
    );
}

TEST(WriterTests, testWriteToUnspecifiedChannel) {
    auto client = new_test_client();
    auto [time, _] = create_indexed_pair(client);
    auto writer = ASSERT_NIL_P(client.telem.open_writer(synnax::WriterConfig{
        std::vector{time.key},
        telem::TimeStamp::now(),
        std::vector{telem::AUTH_ABSOLUTE},
        telem::ControlSubject{"test_writer"},
    }));
    auto frame = synnax::Frame(1);
    frame.emplace(
        1000,
        telem::Series(std::vector<float>{2, 3, 4, 5, 6, 7, 8, 9})
    );
    ASSERT_NIL(writer.write(frame));
    ASSERT_OCCURRED_AS_P(writer.commit(), xerrors::VALIDATION);
}

TEST(WriterTests, testWriteErrOnUnauthorized) {
    auto client = new_test_client();
    auto time = ASSERT_NIL_P(client.channels.create("time", telem::TIMESTAMP_T, 0, true)
    );
    auto data = ASSERT_NIL_P(
        client.channels.create("data", telem::UINT8_T, time.key, false)
    );
    auto w1 = ASSERT_NIL_P(client.telem.open_writer(synnax::WriterConfig{
        .channels = std::vector{time.key, data.key},
        .start = telem::TimeStamp::now(),
        .authorities = std::vector{telem::AUTH_ABSOLUTE, telem::AUTH_ABSOLUTE},
        .subject = telem::ControlSubject{"test_writer_1"},
        .err_on_unauthorized = true
    }));
    auto [w2, err] = client.telem.open_writer(synnax::WriterConfig{
        .channels = std::vector{time.key, data.key},
        .start = telem::TimeStamp::now(),
        .authorities = std::vector{telem::AUTH_ABSOLUTE, telem::AUTH_ABSOLUTE},
        .subject = telem::ControlSubject{"test_writer_2"},
        .err_on_unauthorized = true
    });
    ASSERT_OCCURRED_AS(err, xerrors::UNAUTHORIZED);
    ASSERT_TRUE(err.message().find("test_writer_1") != std::string::npos);
}

TEST(WriterTests, testSetAuthority) {
    auto client = new_test_client();
    auto time = ASSERT_NIL_P(client.channels.create("time", telem::TIMESTAMP_T, 0, true)
    );
    auto data1 = ASSERT_NIL_P(
        client.channels.create("data1", telem::UINT8_T, time.key, false)
    );
    auto data2 = ASSERT_NIL_P(
        client.channels.create("data2", telem::UINT8_T, time.key, false)
    );

    auto writer = ASSERT_NIL_P(client.telem.open_writer(synnax::WriterConfig{
        .channels = std::vector{time.key, data1.key, data2.key},
        .start = telem::TimeStamp::now(),
        .authorities = std::
            vector{telem::AUTH_ABSOLUTE, telem::AUTH_ABSOLUTE, telem::AUTH_ABSOLUTE},
        .subject = telem::ControlSubject{"test_writer"},
        .err_on_unauthorized = true
    }));

    // Test setting authority for all channels
    ASSERT_NIL(writer.set_authority(0));

    // Test setting authority for a single channel
    ASSERT_NIL(writer.set_authority(data1.key, telem::AUTH_ABSOLUTE));

    // Test setting different authorities for multiple channels
    ASSERT_NIL(writer.set_authority(
        std::vector{time.key, data2.key},
        std::vector{telem::AUTH_ABSOLUTE, telem::AUTH_ABSOLUTE}
    ));

    ASSERT_NIL(writer.close());
}
