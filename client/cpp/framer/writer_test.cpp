// Copyright 2025 Synnax Labs, Inc.
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


/// @brief it should correctly write a frame of telemetry to the DB.
TEST(FramerTests, testWriteBasic) {
    auto client = new_test_client();
    auto [time, tErr] = client.channels.create(
        "time",
        synnax::TIMESTAMP,
        0,
        true
    );
    ASSERT_FALSE(tErr) << tErr.message();
    auto [data, dErr] = client.channels.create(
        "data",
        synnax::SY_UINT8,
        time.key,
        false
    );
    ASSERT_FALSE(dErr) << dErr.message();

    auto now = synnax::TimeStamp::now();
    auto [writer, wErr] = client.telem.open_writer(synnax::WriterConfig{
        std::vector<synnax::ChannelKey>{time.key, data.key},
        now,
        std::vector<synnax::Authority>{synnax::AUTH_ABSOLUTE, synnax::AUTH_ABSOLUTE},
        synnax::ControlSubject{"test_writer"},
    });
    ASSERT_FALSE(wErr) << wErr.message();

    auto frame = synnax::Frame(2);
    frame.emplace(
        time.key,
        synnax::Series(std::vector<std::uint64_t>{
                           (now.value + synnax::SECOND).value,
                           (now + synnax::SECOND * 2).value,
                           (now + synnax::SECOND * 3).value,
                           (now + synnax::SECOND * 4).value,
                           (now + synnax::SECOND * 5).value,
                           (now + synnax::SECOND * 6).value,
                           (now + synnax::SECOND * 7).value,
                           (now + synnax::SECOND * 8).value,
                       }, synnax::TIMESTAMP)
    );
    frame.emplace(
        data.key,
        synnax::Series(std::vector<uint8_t>{2, 3, 4, 5, 6, 7, 8, 9})
    );


    ASSERT_TRUE(writer.write(frame));
    auto [end, ok] = writer.commit();
    ASSERT_TRUE(ok);
    ASSERT_EQ(end.value, (now + (synnax::SECOND * 8 + 1)).value);
    auto err = writer.close();
    ASSERT_FALSE(err) << err.message();
}

TEST(FramerTests, testOpenWriterOnNonexistentChannel) {
    auto client = new_test_client();
    auto [time, t_err] = client.channels.create(
        "time",
        synnax::TIMESTAMP,
        0,
        true
    );
    ASSERT_FALSE(t_err) << t_err.message();
    auto now = synnax::TimeStamp::now();
    auto [writer, w_err] = client.telem.open_writer(synnax::WriterConfig{
        std::vector<synnax::ChannelKey>{time.key, 1000},
        now,
        std::vector<synnax::Authority>{synnax::AUTH_ABSOLUTE},
        synnax::ControlSubject{"test_writer"},
    });
    ASSERT_TRUE(w_err) << w_err.message();
    ASSERT_TRUE(w_err.matches(synnax::QUERY_ERROR));
}

TEST(FramerTests, testWriteToUnspecifiedChannel) {
    auto client = new_test_client();
    auto [time, t_err] = client.channels.create(
        "time",
        synnax::TIMESTAMP,
        0,
        true
    );
    ASSERT_FALSE(t_err) << t_err.message();
    auto [writer, w_err] = client.telem.open_writer(synnax::WriterConfig{
        std::vector<synnax::ChannelKey>{time.key},
        synnax::TimeStamp::now(),
        std::vector<synnax::Authority>{synnax::AUTH_ABSOLUTE},
        synnax::ControlSubject{"test_writer"},
    });
    ASSERT_FALSE(w_err) << w_err.message();
    auto frame = synnax::Frame(1);
    frame.emplace(
        1000,
        synnax::Series(std::vector<uint8_t>{2, 3, 4, 5, 6, 7, 8, 9})
    );
    ASSERT_TRUE(writer.write(frame));
    auto [end, ok] = writer.commit();
    ASSERT_FALSE(ok);
    auto err = writer.error();
    ASSERT_TRUE(err) << err.message();
    ASSERT_TRUE(err.matches(synnax::VALIDATION_ERROR)) << err.message();
}

TEST(FramerTests, testWriteErrOnUnauthorized) {
    auto client = new_test_client();
    auto [time, t_err] = client.channels.create(
        "time",
        synnax::TIMESTAMP,
        0,
        true
    );
    ASSERT_FALSE(t_err) << t_err.message();
    auto [data, d_err] = client.channels.create(
        "data",
        synnax::SY_UINT8,
        time.key,
        false
    );
    auto [w1, w_err] = client.telem.open_writer(synnax::WriterConfig{
        .channels = std::vector{time.key, data.key},
        .start = synnax::TimeStamp::now(),
        .authorities = std::vector{synnax::AUTH_ABSOLUTE, synnax::AUTH_ABSOLUTE},
        .subject = synnax::ControlSubject{"test_writer_1"},
        .err_on_unauthorized = true
    });
    ASSERT_FALSE(w_err) << w_err.message();
    auto [w2, w2_err] = client.telem.open_writer(synnax::WriterConfig{
        .channels = std::vector{time.key, data.key},
        .start = synnax::TimeStamp::now(),
        .authorities = std::vector{synnax::AUTH_ABSOLUTE, synnax::AUTH_ABSOLUTE},
        .subject = synnax::ControlSubject{"test_writer_2"},
        .err_on_unauthorized = true
    });
    ASSERT_TRUE(w2_err) << w2_err.message();
    ASSERT_TRUE(w2_err.matches(synnax::UNAUTHORIZED_ERROR));
    ASSERT_TRUE(w2_err.message().find("test_writer_1") != std::string::npos);
    ASSERT_TRUE(w2.close());
}

TEST(FramerTests, testSetAuthority) {
    auto client = new_test_client();
    auto [time, t_err] = client.channels.create(
        "time",
        synnax::TIMESTAMP,
        0,
        true
    );
    ASSERT_FALSE(t_err) << t_err.message();
    auto [data1, d1_err] = client.channels.create(
        "data1",
        synnax::SY_UINT8,
        time.key,
        false
    );
    ASSERT_FALSE(d1_err) << d1_err.message();
    auto [data2, d2_err] = client.channels.create(
        "data2",
        synnax::SY_UINT8,
        time.key,
        false
    );
    ASSERT_FALSE(d2_err) << d2_err.message();

    auto [writer, w_err] = client.telem.open_writer(synnax::WriterConfig{
        .channels = std::vector{time.key, data1.key, data2.key},
        .start = synnax::TimeStamp::now(),
        .authorities = std::vector{synnax::AUTH_ABSOLUTE, synnax::AUTH_ABSOLUTE, synnax::AUTH_ABSOLUTE},
        .subject = synnax::ControlSubject{"test_writer"},
        .err_on_unauthorized = true
    });
    ASSERT_FALSE(w_err) << w_err.message();

    // Test setting authority for all channels
    ASSERT_TRUE(writer.set_authority(0));

    // Test setting authority for a single channel
    ASSERT_TRUE(writer.set_authority(data1.key, synnax::AUTH_ABSOLUTE));

    // Test setting different authorities for multiple channels
    ASSERT_TRUE(writer.set_authority(
        std::vector{time.key, data2.key},
        std::vector{synnax::AUTH_ABSOLUTE, synnax::AUTH_ABSOLUTE}
    ));

    auto err = writer.close();
    ASSERT_FALSE(err) << err.message();
}
