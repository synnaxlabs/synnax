// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <thread>
#include <stdio.h>
#include "gtest/gtest.h"
#include "client/cpp/synnax/synnax.h"
#include "client/cpp/synnax/testutil/testutil.h"


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
            synnax::UINT8,
            time.key,
            false
    );
    ASSERT_FALSE(dErr) << dErr.message();

    auto now = synnax::TimeStamp::now();
    auto [writer, wErr] = client.telem.openWriter(synnax::WriterConfig{
            std::vector<synnax::ChannelKey>{time.key, data.key},
            now,
            std::vector<synnax::Authority>{synnax::ABSOLUTTE, synnax::ABSOLUTTE},
            synnax::ControlSubject{"test_writer"},
    });
    ASSERT_FALSE(wErr) << wErr.message();

    auto frame = synnax::Frame(2);
    frame.add(
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
    frame.add(
            data.key,
            synnax::Series(std::vector<uint8_t>{2, 3, 4, 5, 6, 7, 8, 9})
    );


    ASSERT_TRUE(writer.write(std::move(frame)));
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
        auto [writer, w_err] = client.telem.openWriter(synnax::WriterConfig{
                std::vector<synnax::ChannelKey>{time.key, 1000},
                now,
                std::vector<synnax::Authority>{synnax::ABSOLUTTE},
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
        auto [writer, w_err] = client.telem.openWriter(synnax::WriterConfig{
                std::vector<synnax::ChannelKey>{time.key},
                synnax::TimeStamp::now(),
                std::vector<synnax::Authority>{synnax::ABSOLUTTE},
                synnax::ControlSubject{"test_writer"},
        });
        ASSERT_FALSE(w_err) << w_err.message();
        auto frame = synnax::Frame(1);
        frame.add(
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
