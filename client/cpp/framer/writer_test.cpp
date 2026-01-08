// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <thread>

#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/test/xtest.h"

/// @brief it should correctly write a frame of telemetry to the DB.
TEST(WriterTests, testWriteBasic) {
    auto client = new_test_client();
    auto [time, data] = create_indexed_pair(client);
    auto now = x::telem::TimeStamp::now();
    auto writer = ASSERT_NIL_P(client.telem.open_writer(
        synnax::WriterConfig{
            synnax::keys_from_channels(time, data),
            now,
            std::vector{x::telem::AUTH_ABSOLUTE, x::telem::AUTH_ABSOLUTE},
            x::telem::ControlSubject{"test_writer"},
        }
    ));

    auto frame = x::telem::Frame(2);
    frame.emplace(
        time.key,
        x::telem::Series(
            std::vector{
                (now + x::telem::SECOND),
                (now + x::telem::SECOND * 2),
                (now + x::telem::SECOND * 3),
                (now + x::telem::SECOND * 4),
                (now + x::telem::SECOND * 5),
                (now + x::telem::SECOND * 6),
                (now + x::telem::SECOND * 7),
                (now + x::telem::SECOND * 8),
            }
        )
    );
    frame.emplace(data.key, x::telem::Series(std::vector<float>{2, 3, 4, 5, 6, 7, 8, 9}));

    ASSERT_NIL(writer.write(frame));
    auto end = ASSERT_NIL_P(writer.commit());
    ASSERT_EQ(end, now + (x::telem::SECOND * 8 + 1));
    ASSERT_NIL(writer.close());
}

/// @brief it should return a validation error when attempting to open a writer on
/// a non-existent channel.
TEST(WriterTests, testOpenWriterOnNonexistentChannel) {
    auto client = new_test_client();
    auto [time, data] = create_indexed_pair(client);
    const auto now = x::telem::TimeStamp::now();
    ASSERT_OCCURRED_AS_P(
        client.telem.open_writer(
            synnax::WriterConfig{
                std::vector<synnax::ChannelKey>{time.key, 1000},
                now,
                std::vector{x::telem::AUTH_ABSOLUTE},
                x::telem::ControlSubject{"test_writer"},
            }
        ),
        x::errors::NOT_FOUND
    );
}

/// @brief it should return a validation error when attempting to write a frame with
/// an unknown channel.
TEST(WriterTests, testWriteToUnspecifiedChannel) {
    auto client = new_test_client();
    auto [time, _] = create_indexed_pair(client);
    auto writer = ASSERT_NIL_P(client.telem.open_writer(
        synnax::WriterConfig{
            std::vector{time.key},
            x::telem::TimeStamp::now(),
            std::vector{x::telem::AUTH_ABSOLUTE},
            x::telem::ControlSubject{"test_writer"},
        }
    ));
    auto frame = x::telem::Frame(1);
    frame.emplace(1000, x::telem::Series(x::telem::TimeStamp::now()));
    ASSERT_OCCURRED_AS(writer.write(frame), x::errors::VALIDATION);
    ASSERT_OCCURRED_AS_P(writer.commit(), x::errors::VALIDATION);
    ASSERT_OCCURRED_AS(writer.close(), x::errors::VALIDATION);
}

/// @brief it should return a validation error when attempting to write a frame with
/// a series that does not match the data type of the channel.
TEST(WriterTests, testWriteSeriesWithMismatchedDataType) {
    auto client = new_test_client();
    auto [time, data] = create_indexed_pair(client);
    auto writer = ASSERT_NIL_P(client.telem.open_writer(
        synnax::WriterConfig{
            std::vector{time.key, data.key},
            x::telem::TimeStamp::now(),
            std::vector{x::telem::AUTH_ABSOLUTE, x::telem::AUTH_ABSOLUTE},
            x::telem::ControlSubject{"test_writer"},
        }
    ));
    auto frame = x::telem::Frame(2);
    frame.emplace(time.key, x::telem::Series(x::telem::TimeStamp::now()));
    frame.emplace(data.key, x::telem::Series(std::vector<uint32_t>{1}));
    ASSERT_OCCURRED_AS(writer.write(frame), x::errors::VALIDATION);
    ASSERT_OCCURRED_AS_P(writer.commit(), x::errors::VALIDATION);
    ASSERT_OCCURRED_AS(writer.close(), x::errors::VALIDATION);
}

/// @brief it should return an error when attempting to open a writer on channels that
/// are already being written to and err_on_unauthorized is true.
TEST(WriterTests, testWriteErrOnUnauthorized) {
    auto client = new_test_client();
    auto time = ASSERT_NIL_P(client.channels.create(
        make_unique_channel_name("err_on_unauthorized_time"),
        x::telem::TIMESTAMP_T,
        0,
        true
    ));
    auto data = ASSERT_NIL_P(client.channels.create(
        make_unique_channel_name("err_on_unauthorized_data"),
        x::telem::UINT8_T,
        time.key,
        false
    ));
    auto w1 = ASSERT_NIL_P(client.telem.open_writer(
        synnax::WriterConfig{
            .channels = std::vector{time.key, data.key},
            .start = x::telem::TimeStamp::now(),
            .authorities = std::vector{x::telem::AUTH_ABSOLUTE, x::telem::AUTH_ABSOLUTE},
            .subject = x::telem::ControlSubject{"test_writer_1"},
            .err_on_unauthorized = true
        }
    ));
    auto [w2, err] = client.telem.open_writer(
        synnax::WriterConfig{
            .channels = std::vector{time.key, data.key},
            .start = x::telem::TimeStamp::now(),
            .authorities = std::vector{x::telem::AUTH_ABSOLUTE, x::telem::AUTH_ABSOLUTE},
            .subject = x::telem::ControlSubject{"test_writer_2"},
            .err_on_unauthorized = true
        }
    );
    ASSERT_OCCURRED_AS(err, x::errors::UNAUTHORIZED);
    ASSERT_TRUE(err.message().find("test_writer_1") != std::string::npos);
}

/// @brief it should correctly change the authority of a writer.
TEST(WriterTests, testSetAuthority) {
    auto client = new_test_client();
    auto time = ASSERT_NIL_P(client.channels.create(
        make_unique_channel_name("set_authority_time"),
        x::telem::TIMESTAMP_T,
        0,
        true
    ));
    auto data1 = ASSERT_NIL_P(client.channels.create(
        make_unique_channel_name("set_authority_data1"),
        x::telem::UINT8_T,
        time.key,
        false
    ));
    auto data2 = ASSERT_NIL_P(client.channels.create(
        make_unique_channel_name("set_authority_data2"),
        x::telem::UINT8_T,
        time.key,
        false
    ));

    auto writer = ASSERT_NIL_P(client.telem.open_writer(
        synnax::WriterConfig{
            .channels = std::vector{time.key, data1.key, data2.key},
            .start = x::telem::TimeStamp::now(),
            .authorities =
                std::vector{
                    x::telem::AUTH_ABSOLUTE,
                    x::telem::AUTH_ABSOLUTE,
                    x::telem::AUTH_ABSOLUTE
                },
            .subject = x::telem::ControlSubject{"test_writer"},
            .err_on_unauthorized = true
        }
    ));

    // Test setting authority for all channels
    ASSERT_NIL(writer.set_authority(0));

    // Test setting authority for a single channel
    ASSERT_NIL(writer.set_authority(data1.key, x::telem::AUTH_ABSOLUTE));

    // Test setting different authorities for multiple channels
    ASSERT_NIL(writer.set_authority(
        std::vector{time.key, data2.key},
        std::vector{x::telem::AUTH_ABSOLUTE, x::telem::AUTH_ABSOLUTE}
    ));

    ASSERT_NIL(writer.close());
}

/// @brief close can be called as many times as desired and should not return an error
/// when the writer has a nominal shutdown.
TEST(WriterTests, testCloseIdempotency) {
    auto client = new_test_client();
    auto [time, data] = create_indexed_pair(client);
    auto now = x::telem::TimeStamp::now();
    auto writer = ASSERT_NIL_P(client.telem.open_writer(
        synnax::WriterConfig{
            synnax::keys_from_channels(time, data),
            now,
            std::vector{x::telem::AUTH_ABSOLUTE, x::telem::AUTH_ABSOLUTE},
            x::telem::ControlSubject{"test_writer"},
        }
    ));

    auto frame = x::telem::Frame(2);
    frame.emplace(time.key, x::telem::Series(now));
    frame.emplace(data.key, x::telem::Series(std::vector<float>{2}));

    ASSERT_NIL(writer.write(frame));
    auto end = ASSERT_NIL_P(writer.commit());
    ASSERT_EQ(end, now + 1 * x::telem::NANOSECOND);
    ASSERT_NIL(writer.close());
    ASSERT_NIL(writer.close());
    ASSERT_NIL(writer.close());
    ASSERT_NIL(writer.close());
    ASSERT_NIL(writer.close());
    ASSERT_NIL(writer.close());
}

/// @brief once a writer encounters an error, it should continually return that error
/// on any subsequent method calls.
TEST(WriterTests, testErrorCommunication) {
    auto client = new_test_client();
    auto [time, data] = create_indexed_pair(client);
    auto writer = ASSERT_NIL_P(client.telem.open_writer(
        synnax::WriterConfig{
            std::vector{time.key, data.key},
            x::telem::TimeStamp::now(),
            std::vector{x::telem::AUTH_ABSOLUTE, x::telem::AUTH_ABSOLUTE},
            x::telem::ControlSubject{"test_writer"},
        }
    ));
    auto frame = x::telem::Frame(2);
    frame.emplace(time.key, x::telem::Series(x::telem::TimeStamp::now()));
    frame.emplace(data.key, x::telem::Series(std::vector<uint32_t>{1}));
    ASSERT_OCCURRED_AS(writer.write(frame), x::errors::VALIDATION);
    ASSERT_OCCURRED_AS_P(writer.commit(), x::errors::VALIDATION);
    ASSERT_OCCURRED_AS_P(writer.commit(), x::errors::VALIDATION);
    ASSERT_OCCURRED_AS_P(writer.commit(), x::errors::VALIDATION);
    ASSERT_OCCURRED_AS_P(writer.commit(), x::errors::VALIDATION);
    ASSERT_OCCURRED_AS_P(writer.commit(), x::errors::VALIDATION);
    ASSERT_OCCURRED_AS(writer.set_authority(3), x::errors::VALIDATION);
    ASSERT_OCCURRED_AS_P(writer.commit(), x::errors::VALIDATION);
    ASSERT_OCCURRED_AS(writer.set_authority(255), x::errors::VALIDATION);
    ASSERT_OCCURRED_AS_P(writer.commit(), x::errors::VALIDATION);
    ASSERT_OCCURRED_AS(writer.write(frame), x::errors::VALIDATION);
    ASSERT_OCCURRED_AS(writer.close(), x::errors::VALIDATION);
}
