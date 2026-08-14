// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstring>
#include <map>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "client/clib/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/test/test.h"

namespace synnax::clib {
constexpr int32_t OK = 0;

SynnaxClient *open_test_client(SynnaxError *err) {
    SynnaxClient *client = nullptr;
    synnax_client_open(
        "localhost",
        9090,
        "synnax",
        "seldon",
        0,
        nullptr,
        nullptr,
        nullptr,
        0,
        0,
        &client,
        err
    );
    return client;
}

/// @brief opens a writer through the C ABI over the given channel keys, starting at
/// start, in the default persist+stream mode with absolute authority.
SynnaxWriter *open_test_writer(
    SynnaxClient *client,
    const int64_t start,
    const std::vector<uint32_t> &channels,
    SynnaxError *err
) {
    SynnaxWriter *writer = nullptr;
    synnax_writer_open(
        client,
        start,
        channels.data(),
        channels.size(),
        nullptr,
        0,
        "clib_test",
        0,
        SYNNAX_WRITER_MODE_DEFAULT,
        0,
        0,
        0,
        0,
        &writer,
        err
    );
    return writer;
}

/// @brief encodes strings into the uint32-LE length-prefixed, channel-major buffer that
/// synnax_writer_write_strings expects.
std::vector<uint8_t> encode_strings(const std::vector<std::string> &strings) {
    std::vector<uint8_t> buf;
    for (const auto &s: strings) {
        const auto len = static_cast<uint32_t>(s.size());
        buf.push_back(static_cast<uint8_t>(len));
        buf.push_back(static_cast<uint8_t>(len >> 8));
        buf.push_back(static_cast<uint8_t>(len >> 16));
        buf.push_back(static_cast<uint8_t>(len >> 24));
        buf.insert(buf.end(), s.begin(), s.end());
    }
    return buf;
}

template<typename T>
void accumulate_values(
    synnax::framer::Streamer &streamer,
    std::vector<T> &out,
    const size_t n
) {
    while (out.size() < n) {
        auto frame = ASSERT_NIL_P(streamer.read());
        for (size_t i = 0; i < frame.size(); i++) {
            const auto values = frame.series->at(i).values<T>();
            out.insert(out.end(), values.begin(), values.end());
        }
    }
}

void accumulate_strings(
    synnax::framer::Streamer &streamer,
    std::vector<std::string> &out,
    const size_t n
) {
    while (out.size() < n) {
        auto frame = ASSERT_NIL_P(streamer.read());
        for (size_t i = 0; i < frame.size(); i++) {
            const auto values = frame.series->at(i).strings();
            out.insert(out.end(), values.begin(), values.end());
        }
    }
}

/// @brief reads frames until each of keys has accumulated at least n samples, grouping
/// the samples by their source channel so per-channel ordering can be asserted.
template<typename T>
void accumulate_values_by_channel(
    synnax::framer::Streamer &streamer,
    const std::vector<uint32_t> &keys,
    const size_t n,
    std::map<uint32_t, std::vector<T>> &out
) {
    for (const auto key: keys)
        out[key];
    bool complete = false;
    while (!complete) {
        auto frame = ASSERT_NIL_P(streamer.read());
        for (size_t i = 0; i < frame.size(); i++) {
            const auto it = out.find(frame.channels->at(i));
            if (it == out.end()) continue;
            const auto values = frame.series->at(i).values<T>();
            it->second.insert(it->second.end(), values.begin(), values.end());
        }
        complete = true;
        for (const auto key: keys)
            if (out[key].size() < n) complete = false;
    }
}

/// @brief the string analog of accumulate_values_by_channel.
void accumulate_strings_by_channel(
    synnax::framer::Streamer &streamer,
    const std::vector<uint32_t> &keys,
    const size_t n,
    std::map<uint32_t, std::vector<std::string>> &out
) {
    for (const auto key: keys)
        out[key];
    bool complete = false;
    while (!complete) {
        auto frame = ASSERT_NIL_P(streamer.read());
        for (size_t i = 0; i < frame.size(); i++) {
            const auto it = out.find(frame.channels->at(i));
            if (it == out.end()) continue;
            const auto values = frame.series->at(i).strings();
            it->second.insert(it->second.end(), values.begin(), values.end());
        }
        complete = true;
        for (const auto key: keys)
            if (out[key].size() < n) complete = false;
    }
}

/// @brief it should reject opening a writer with a null client.
TEST(ClibWriter, testOpenWithNullClientReturnsValidationError) {
    SynnaxError err;
    SynnaxWriter *writer = nullptr;
    const uint32_t channels[] = {1};
    const int32_t code = synnax_writer_open(
        nullptr,
        0,
        channels,
        1,
        nullptr,
        0,
        "clib_test",
        0,
        SYNNAX_WRITER_MODE_DEFAULT,
        0,
        0,
        0,
        0,
        &writer,
        &err
    );
    EXPECT_NE(code, OK);
    EXPECT_STREQ(err.type, "sy.validation");
}

/// @brief it should reject a write on a null writer.
TEST(ClibWriter, testWriteWithNullWriterReturnsValidationError) {
    SynnaxError err;
    const uint32_t channels[] = {1};
    const float data[] = {1.0f};
    const int64_t timestamps[] = {1};
    const int32_t code = synnax_writer_write(
        nullptr,
        0,
        timestamps,
        channels,
        1,
        data,
        sizeof(data),
        1,
        "float32",
        &err
    );
    EXPECT_NE(code, OK);
    EXPECT_STREQ(err.type, "sy.validation");
}

/// @brief it should reject a commit on a null writer.
TEST(ClibWriter, testCommitWithNullWriterReturnsValidationError) {
    SynnaxError err;
    int64_t end_ts = 0;
    EXPECT_NE(synnax_writer_commit(nullptr, &end_ts, &err), OK);
    EXPECT_STREQ(err.type, "sy.validation");
}

/// @brief it should treat closing a null writer as a no-op success.
TEST(ClibWriter, testCloseWriterOnNullReturnsOk) {
    SynnaxError err;
    EXPECT_EQ(synnax_writer_close(nullptr, &err), OK);
}

/// @brief it should write fixed-width samples that read back unchanged.
TEST(ClibWriter, testWritesFixedWidthRoundTrip) {
    auto cpp_client = new_test_client();
    auto [time, data] = create_indexed_pair(cpp_client);
    auto streamer = ASSERT_NIL_P(cpp_client.telem.open_streamer(
        synnax::framer::StreamerConfig{std::vector{data.key}}
    ));

    SynnaxError err;
    SynnaxClient *client = open_test_client(&err);
    ASSERT_NE(client, nullptr) << err.message;

    const auto base = x::telem::TimeStamp::now();
    std::vector<int64_t> timestamps;
    for (int i = 1; i <= 4; i++)
        timestamps.push_back((base + x::telem::SECOND * i).nanoseconds());
    const std::vector<float> values{1.5f, 2.5f, 3.5f, 4.5f};

    SynnaxWriter *writer = open_test_writer(
        client,
        base.nanoseconds(),
        {time.key, data.key},
        &err
    );
    ASSERT_NE(writer, nullptr) << err.message;

    const uint32_t channels[] = {data.key};
    ASSERT_EQ(
        synnax_writer_write(
            writer,
            time.key,
            timestamps.data(),
            channels,
            1,
            values.data(),
            values.size() * sizeof(float),
            values.size(),
            "float32",
            &err
        ),
        OK
    ) << err.message;

    int64_t end_ts = 0;
    ASSERT_EQ(synnax_writer_commit(writer, &end_ts, &err), OK) << err.message;
    EXPECT_GT(end_ts, base.nanoseconds());

    std::vector<float> received;
    accumulate_values<float>(streamer, received, values.size());
    ASSERT_EQ(received.size(), values.size());
    for (size_t i = 0; i < values.size(); i++)
        EXPECT_FLOAT_EQ(received[i], values[i]);

    ASSERT_EQ(synnax_writer_close(writer, &err), OK) << err.message;
    ASSERT_NIL(streamer.close());
    synnax_client_close(client);
}

/// @brief it should reject a fixed-width write with an unknown data type.
TEST(ClibWriter, testRejectsUnknownDataType) {
    auto cpp_client = new_test_client();
    auto [time, data] = create_indexed_pair(cpp_client);

    SynnaxError err;
    SynnaxClient *client = open_test_client(&err);
    ASSERT_NE(client, nullptr) << err.message;
    SynnaxWriter *writer = open_test_writer(
        client,
        x::telem::TimeStamp::now().nanoseconds(),
        {time.key, data.key},
        &err
    );
    ASSERT_NE(writer, nullptr) << err.message;

    const uint32_t channels[] = {data.key};
    const float values[] = {1.0f};
    const int64_t timestamps[] = {x::telem::TimeStamp::now().nanoseconds()};
    const int32_t code = synnax_writer_write(
        writer,
        time.key,
        timestamps,
        channels,
        1,
        values,
        sizeof(values),
        1,
        "not_a_real_type",
        &err
    );
    EXPECT_NE(code, OK);
    EXPECT_STREQ(err.type, "sy.validation");
    EXPECT_NE(std::strstr(err.message, "unknown data type"), nullptr);

    synnax_writer_close(writer, &err);
    synnax_client_close(client);
}

/// @brief it should reject a fixed-width write whose data buffer size disagrees with
/// channel_count * sample_count * density.
TEST(ClibWriter, testRejectsDataBufferSizeMismatch) {
    auto cpp_client = new_test_client();
    auto [time, data] = create_indexed_pair(cpp_client);

    SynnaxError err;
    SynnaxClient *client = open_test_client(&err);
    ASSERT_NE(client, nullptr) << err.message;
    SynnaxWriter *writer = open_test_writer(
        client,
        x::telem::TimeStamp::now().nanoseconds(),
        {time.key, data.key},
        &err
    );
    ASSERT_NE(writer, nullptr) << err.message;

    const uint32_t channels[] = {data.key};
    const float values[] = {1.0f, 2.0f, 3.0f};
    const int64_t timestamps[] = {x::telem::TimeStamp::now().nanoseconds()};
    const int32_t code = synnax_writer_write(
        writer,
        time.key,
        timestamps,
        channels,
        1,
        values,
        sizeof(float),
        3,
        "float32",
        &err
    );
    EXPECT_NE(code, OK);
    EXPECT_STREQ(err.type, "sy.validation");
    EXPECT_NE(std::strstr(err.message, "data buffer size"), nullptr);

    synnax_writer_close(writer, &err);
    synnax_client_close(client);
}

/// @brief it should reject a fixed-width write whose data buffer is larger than the
/// declared samples require.
TEST(ClibWriter, testRejectsDataBufferWithTrailingBytes) {
    auto cpp_client = new_test_client();
    auto [time, data] = create_indexed_pair(cpp_client);

    SynnaxError err;
    SynnaxClient *client = open_test_client(&err);
    ASSERT_NE(client, nullptr) << err.message;
    SynnaxWriter *writer = open_test_writer(
        client,
        x::telem::TimeStamp::now().nanoseconds(),
        {time.key, data.key},
        &err
    );
    ASSERT_NE(writer, nullptr) << err.message;

    const uint32_t channels[] = {data.key};
    const float values[] = {1.0f, 2.0f, 3.0f, 4.0f};
    const int64_t timestamps[] = {x::telem::TimeStamp::now().nanoseconds()};
    const int32_t code = synnax_writer_write(
        writer,
        time.key,
        timestamps,
        channels,
        1,
        values,
        sizeof(values),
        3,
        "float32",
        &err
    );
    EXPECT_NE(code, OK);
    EXPECT_STREQ(err.type, "sy.validation");
    EXPECT_NE(std::strstr(err.message, "data buffer size"), nullptr);

    synnax_writer_close(writer, &err);
    synnax_client_close(client);
}

/// @brief it should write string samples that read back unchanged.
TEST(ClibWriter, testWritesStringsRoundTrip) {
    auto cpp_client = new_test_client();
    auto index = ASSERT_NIL_P(cpp_client.channels.create(
        make_unique_channel_name("str_index"),
        x::telem::TIMESTAMP_T,
        0,
        true
    ));
    auto data = ASSERT_NIL_P(cpp_client.channels.create(
        make_unique_channel_name("str_data"),
        x::telem::STRING_T,
        index.key,
        false
    ));
    auto streamer = ASSERT_NIL_P(cpp_client.telem.open_streamer(
        synnax::framer::StreamerConfig{std::vector{data.key}}
    ));

    SynnaxError err;
    SynnaxClient *client = open_test_client(&err);
    ASSERT_NE(client, nullptr) << err.message;

    const auto base = x::telem::TimeStamp::now();
    std::vector<int64_t> timestamps;
    for (int i = 1; i <= 3; i++)
        timestamps.push_back((base + x::telem::SECOND * i).nanoseconds());
    const std::vector<std::string> values{"alpha", "beta", "gamma"};
    const auto buffer = encode_strings(values);

    SynnaxWriter *writer = open_test_writer(
        client,
        base.nanoseconds(),
        {index.key, data.key},
        &err
    );
    ASSERT_NE(writer, nullptr) << err.message;

    const uint32_t channels[] = {data.key};
    ASSERT_EQ(
        synnax_writer_write_strings(
            writer,
            index.key,
            timestamps.data(),
            channels,
            1,
            buffer.data(),
            buffer.size(),
            values.size(),
            &err
        ),
        OK
    ) << err.message;
    ASSERT_EQ(synnax_writer_commit(writer, nullptr, &err), OK) << err.message;

    std::vector<std::string> received;
    accumulate_strings(streamer, received, values.size());
    ASSERT_EQ(received.size(), values.size());
    for (size_t i = 0; i < values.size(); i++)
        EXPECT_EQ(received[i], values[i]);

    ASSERT_EQ(synnax_writer_close(writer, &err), OK) << err.message;
    ASSERT_NIL(streamer.close());
    synnax_client_close(client);
}

/// @brief it should reject a string buffer too small to hold a length prefix.
TEST(ClibWriter, testRejectsStringBufferTooSmallForLengthPrefix) {
    auto cpp_client = new_test_client();
    auto index = ASSERT_NIL_P(cpp_client.channels.create(
        make_unique_channel_name("str_index"),
        x::telem::TIMESTAMP_T,
        0,
        true
    ));
    auto data = ASSERT_NIL_P(cpp_client.channels.create(
        make_unique_channel_name("str_data"),
        x::telem::STRING_T,
        index.key,
        false
    ));

    SynnaxError err;
    SynnaxClient *client = open_test_client(&err);
    ASSERT_NE(client, nullptr) << err.message;
    SynnaxWriter *writer = open_test_writer(
        client,
        x::telem::TimeStamp::now().nanoseconds(),
        {index.key, data.key},
        &err
    );
    ASSERT_NE(writer, nullptr) << err.message;

    const uint32_t channels[] = {data.key};
    const int64_t timestamps[] = {x::telem::TimeStamp::now().nanoseconds()};
    const uint8_t buffer[] = {0x01, 0x02};
    const int32_t code = synnax_writer_write_strings(
        writer,
        index.key,
        timestamps,
        channels,
        1,
        buffer,
        sizeof(buffer),
        1,
        &err
    );
    EXPECT_NE(code, OK);
    EXPECT_STREQ(err.type, "sy.validation");
    EXPECT_NE(std::strstr(err.message, "length prefix"), nullptr);

    synnax_writer_close(writer, &err);
    synnax_client_close(client);
}

/// @brief it should reject a string buffer too small to hold a declared payload.
TEST(ClibWriter, testRejectsStringBufferTooSmallForPayload) {
    auto cpp_client = new_test_client();
    auto index = ASSERT_NIL_P(cpp_client.channels.create(
        make_unique_channel_name("str_index"),
        x::telem::TIMESTAMP_T,
        0,
        true
    ));
    auto data = ASSERT_NIL_P(cpp_client.channels.create(
        make_unique_channel_name("str_data"),
        x::telem::STRING_T,
        index.key,
        false
    ));

    SynnaxError err;
    SynnaxClient *client = open_test_client(&err);
    ASSERT_NE(client, nullptr) << err.message;
    SynnaxWriter *writer = open_test_writer(
        client,
        x::telem::TimeStamp::now().nanoseconds(),
        {index.key, data.key},
        &err
    );
    ASSERT_NE(writer, nullptr) << err.message;

    const uint32_t channels[] = {data.key};
    const int64_t timestamps[] = {x::telem::TimeStamp::now().nanoseconds()};
    const uint8_t buffer[] = {100, 0, 0, 0, 'a', 'b', 'c'};
    const int32_t code = synnax_writer_write_strings(
        writer,
        index.key,
        timestamps,
        channels,
        1,
        buffer,
        sizeof(buffer),
        1,
        &err
    );
    EXPECT_NE(code, OK);
    EXPECT_STREQ(err.type, "sy.validation");
    EXPECT_NE(std::strstr(err.message, "payload"), nullptr);

    synnax_writer_close(writer, &err);
    synnax_client_close(client);
}

/// @brief it should reject a string buffer with bytes left over after the declared
/// sample count.
TEST(ClibWriter, testRejectsTrailingBytesAfterDeclaredSamples) {
    auto cpp_client = new_test_client();
    auto index = ASSERT_NIL_P(cpp_client.channels.create(
        make_unique_channel_name("str_index"),
        x::telem::TIMESTAMP_T,
        0,
        true
    ));
    auto data = ASSERT_NIL_P(cpp_client.channels.create(
        make_unique_channel_name("str_data"),
        x::telem::STRING_T,
        index.key,
        false
    ));

    SynnaxError err;
    SynnaxClient *client = open_test_client(&err);
    ASSERT_NE(client, nullptr) << err.message;
    SynnaxWriter *writer = open_test_writer(
        client,
        x::telem::TimeStamp::now().nanoseconds(),
        {index.key, data.key},
        &err
    );
    ASSERT_NE(writer, nullptr) << err.message;

    const uint32_t channels[] = {data.key};
    const int64_t timestamps[] = {x::telem::TimeStamp::now().nanoseconds()};
    const auto buffer = encode_strings({"a", "b"});
    const int32_t code = synnax_writer_write_strings(
        writer,
        index.key,
        timestamps,
        channels,
        1,
        buffer.data(),
        buffer.size(),
        1,
        &err
    );
    EXPECT_NE(code, OK);
    EXPECT_STREQ(err.type, "sy.validation");
    EXPECT_NE(std::strstr(err.message, "trailing bytes"), nullptr);

    synnax_writer_close(writer, &err);
    synnax_client_close(client);
}

/// @brief it should pack and write multiple channels (channel-major) so that each
/// channel reads back its own samples.
TEST(ClibWriter, testWritesMultiChannelFixedWidthRoundTrip) {
    auto cpp_client = new_test_client();
    auto index = ASSERT_NIL_P(
        cpp_client.channels
            .create(make_unique_channel_name("index"), x::telem::TIMESTAMP_T, 0, true)
    );
    auto data1 = ASSERT_NIL_P(cpp_client.channels.create(
        make_unique_channel_name("data1"),
        x::telem::FLOAT32_T,
        index.key,
        false
    ));
    auto data2 = ASSERT_NIL_P(cpp_client.channels.create(
        make_unique_channel_name("data2"),
        x::telem::FLOAT32_T,
        index.key,
        false
    ));
    auto streamer = ASSERT_NIL_P(cpp_client.telem.open_streamer(
        synnax::framer::StreamerConfig{std::vector{data1.key, data2.key}}
    ));

    SynnaxError err;
    SynnaxClient *client = open_test_client(&err);
    ASSERT_NE(client, nullptr) << err.message;

    const auto base = x::telem::TimeStamp::now();
    std::vector<int64_t> timestamps;
    for (int i = 1; i <= 3; i++)
        timestamps.push_back((base + x::telem::SECOND * i).nanoseconds());
    const std::vector<float> ch1{1.5f, 2.5f, 3.5f};
    const std::vector<float> ch2{10.5f, 20.5f, 30.5f};
    std::vector<float> packed(ch1);
    packed.insert(packed.end(), ch2.begin(), ch2.end());

    SynnaxWriter *writer = open_test_writer(
        client,
        base.nanoseconds(),
        {index.key, data1.key, data2.key},
        &err
    );
    ASSERT_NE(writer, nullptr) << err.message;

    const uint32_t channels[] = {data1.key, data2.key};
    ASSERT_EQ(
        synnax_writer_write(
            writer,
            index.key,
            timestamps.data(),
            channels,
            2,
            packed.data(),
            packed.size() * sizeof(float),
            3,
            "float32",
            &err
        ),
        OK
    ) << err.message;
    ASSERT_EQ(synnax_writer_commit(writer, nullptr, &err), OK) << err.message;

    std::map<uint32_t, std::vector<float>> received;
    accumulate_values_by_channel<float>(streamer, {data1.key, data2.key}, 3, received);
    EXPECT_EQ(received[data1.key], ch1);
    EXPECT_EQ(received[data2.key], ch2);

    ASSERT_EQ(synnax_writer_close(writer, &err), OK) << err.message;
    ASSERT_NIL(streamer.close());
    synnax_client_close(client);
}

/// @brief it should write an 8-byte-wide (int64) channel that reads back unchanged.
TEST(ClibWriter, testWritesInt64RoundTrip) {
    auto cpp_client = new_test_client();
    auto index = ASSERT_NIL_P(
        cpp_client.channels
            .create(make_unique_channel_name("index"), x::telem::TIMESTAMP_T, 0, true)
    );
    auto data = ASSERT_NIL_P(cpp_client.channels.create(
        make_unique_channel_name("i64"),
        x::telem::INT64_T,
        index.key,
        false
    ));
    auto streamer = ASSERT_NIL_P(cpp_client.telem.open_streamer(
        synnax::framer::StreamerConfig{std::vector{data.key}}
    ));

    SynnaxError err;
    SynnaxClient *client = open_test_client(&err);
    ASSERT_NE(client, nullptr) << err.message;

    const auto base = x::telem::TimeStamp::now();
    std::vector<int64_t> timestamps;
    for (int i = 1; i <= 3; i++)
        timestamps.push_back((base + x::telem::SECOND * i).nanoseconds());
    const std::vector<int64_t> values{-9000000000LL, 0, 9000000000LL};

    SynnaxWriter *writer = open_test_writer(
        client,
        base.nanoseconds(),
        {index.key, data.key},
        &err
    );
    ASSERT_NE(writer, nullptr) << err.message;

    const uint32_t channels[] = {data.key};
    ASSERT_EQ(
        synnax_writer_write(
            writer,
            index.key,
            timestamps.data(),
            channels,
            1,
            values.data(),
            values.size() * sizeof(int64_t),
            values.size(),
            "int64",
            &err
        ),
        OK
    ) << err.message;
    ASSERT_EQ(synnax_writer_commit(writer, nullptr, &err), OK) << err.message;

    std::vector<int64_t> received;
    accumulate_values<int64_t>(streamer, received, values.size());
    EXPECT_EQ(received, values);

    ASSERT_EQ(synnax_writer_close(writer, &err), OK) << err.message;
    ASSERT_NIL(streamer.close());
    synnax_client_close(client);
}

/// @brief it should write a 1-byte-wide (uint8) channel that reads back unchanged.
TEST(ClibWriter, testWritesUint8RoundTrip) {
    auto cpp_client = new_test_client();
    auto index = ASSERT_NIL_P(
        cpp_client.channels
            .create(make_unique_channel_name("index"), x::telem::TIMESTAMP_T, 0, true)
    );
    auto data = ASSERT_NIL_P(
        cpp_client.channels
            .create(make_unique_channel_name("u8"), x::telem::UINT8_T, index.key, false)
    );
    auto streamer = ASSERT_NIL_P(cpp_client.telem.open_streamer(
        synnax::framer::StreamerConfig{std::vector{data.key}}
    ));

    SynnaxError err;
    SynnaxClient *client = open_test_client(&err);
    ASSERT_NE(client, nullptr) << err.message;

    const auto base = x::telem::TimeStamp::now();
    std::vector<int64_t> timestamps;
    for (int i = 1; i <= 3; i++)
        timestamps.push_back((base + x::telem::SECOND * i).nanoseconds());
    const std::vector<uint8_t> values{0, 127, 255};

    SynnaxWriter *writer = open_test_writer(
        client,
        base.nanoseconds(),
        {index.key, data.key},
        &err
    );
    ASSERT_NE(writer, nullptr) << err.message;

    const uint32_t channels[] = {data.key};
    ASSERT_EQ(
        synnax_writer_write(
            writer,
            index.key,
            timestamps.data(),
            channels,
            1,
            values.data(),
            values.size() * sizeof(uint8_t),
            values.size(),
            "uint8",
            &err
        ),
        OK
    ) << err.message;
    ASSERT_EQ(synnax_writer_commit(writer, nullptr, &err), OK) << err.message;

    std::vector<uint8_t> received;
    accumulate_values<uint8_t>(streamer, received, values.size());
    EXPECT_EQ(received, values);

    ASSERT_EQ(synnax_writer_close(writer, &err), OK) << err.message;
    ASSERT_NIL(streamer.close());
    synnax_client_close(client);
}

/// @brief it should slice a multi-channel string buffer so each channel reads back its
/// own samples.
TEST(ClibWriter, testWritesMultiChannelStringsRoundTrip) {
    auto cpp_client = new_test_client();
    auto index = ASSERT_NIL_P(cpp_client.channels.create(
        make_unique_channel_name("str_index"),
        x::telem::TIMESTAMP_T,
        0,
        true
    ));
    auto data1 = ASSERT_NIL_P(cpp_client.channels.create(
        make_unique_channel_name("str1"),
        x::telem::STRING_T,
        index.key,
        false
    ));
    auto data2 = ASSERT_NIL_P(cpp_client.channels.create(
        make_unique_channel_name("str2"),
        x::telem::STRING_T,
        index.key,
        false
    ));
    auto streamer = ASSERT_NIL_P(cpp_client.telem.open_streamer(
        synnax::framer::StreamerConfig{std::vector{data1.key, data2.key}}
    ));

    SynnaxError err;
    SynnaxClient *client = open_test_client(&err);
    ASSERT_NE(client, nullptr) << err.message;

    const auto base = x::telem::TimeStamp::now();
    std::vector<int64_t> timestamps;
    for (int i = 1; i <= 2; i++)
        timestamps.push_back((base + x::telem::SECOND * i).nanoseconds());
    const std::vector<std::string> ch1{"alpha", "beta"};
    const std::vector<std::string> ch2{"gamma", "delta"};
    std::vector<std::string> all(ch1);
    all.insert(all.end(), ch2.begin(), ch2.end());
    const auto buffer = encode_strings(all);

    SynnaxWriter *writer = open_test_writer(
        client,
        base.nanoseconds(),
        {index.key, data1.key, data2.key},
        &err
    );
    ASSERT_NE(writer, nullptr) << err.message;

    const uint32_t channels[] = {data1.key, data2.key};
    ASSERT_EQ(
        synnax_writer_write_strings(
            writer,
            index.key,
            timestamps.data(),
            channels,
            2,
            buffer.data(),
            buffer.size(),
            2,
            &err
        ),
        OK
    ) << err.message;
    ASSERT_EQ(synnax_writer_commit(writer, nullptr, &err), OK) << err.message;

    std::map<uint32_t, std::vector<std::string>> received;
    accumulate_strings_by_channel(streamer, {data1.key, data2.key}, 2, received);
    EXPECT_EQ(received[data1.key], ch1);
    EXPECT_EQ(received[data2.key], ch2);

    ASSERT_EQ(synnax_writer_close(writer, &err), OK) << err.message;
    ASSERT_NIL(streamer.close());
    synnax_client_close(client);
}

/// @brief it should report a not-found error when opening a writer on a channel that
/// does not exist.
TEST(ClibWriter, testRejectsOpenWriterOnNonexistentChannel) {
    auto cpp_client = new_test_client();
    auto [time, data] = create_indexed_pair(cpp_client);

    SynnaxError err;
    SynnaxClient *client = open_test_client(&err);
    ASSERT_NE(client, nullptr) << err.message;

    const uint32_t channels[] = {time.key, 4000000000u};
    SynnaxWriter *writer = nullptr;
    const int32_t code = synnax_writer_open(
        client,
        x::telem::TimeStamp::now().nanoseconds(),
        channels,
        2,
        nullptr,
        0,
        "clib_test",
        0,
        SYNNAX_WRITER_MODE_DEFAULT,
        0,
        0,
        0,
        0,
        &writer,
        &err
    );
    EXPECT_NE(code, OK);
    EXPECT_NE(std::strstr(err.type, "not_found"), nullptr);

    synnax_client_close(client);
}

/// @brief it should let an auto-index writer omit the index channel, generating
/// server-side timestamps for the written samples.
TEST(ClibWriter, testWritesAutoIndexWithoutIndexChannel) {
    auto cpp_client = new_test_client();
    auto [time, data] = create_indexed_pair(cpp_client);
    auto streamer = ASSERT_NIL_P(cpp_client.telem.open_streamer(
        synnax::framer::StreamerConfig{std::vector{data.key}}
    ));

    SynnaxError err;
    SynnaxClient *client = open_test_client(&err);
    ASSERT_NE(client, nullptr) << err.message;

    const uint32_t open_channels[] = {data.key};
    SynnaxWriter *writer = nullptr;
    ASSERT_EQ(
        synnax_writer_open(
            client,
            0,
            open_channels,
            1,
            nullptr,
            0,
            "clib_test",
            0,
            SYNNAX_WRITER_MODE_DEFAULT,
            0,
            0,
            0,
            1,
            &writer,
            &err
        ),
        OK
    ) << err.message;

    const std::vector<float> values{7.5f, 8.5f, 9.5f};
    const uint32_t channels[] = {data.key};
    ASSERT_EQ(
        synnax_writer_write(
            writer,
            0,
            nullptr,
            channels,
            1,
            values.data(),
            values.size() * sizeof(float),
            values.size(),
            "float32",
            &err
        ),
        OK
    ) << err.message;
    ASSERT_EQ(synnax_writer_commit(writer, nullptr, &err), OK) << err.message;

    std::vector<float> received;
    accumulate_values<float>(streamer, received, values.size());
    ASSERT_EQ(received.size(), values.size());
    for (size_t i = 0; i < values.size(); i++)
        EXPECT_FLOAT_EQ(received[i], values[i]);

    ASSERT_EQ(synnax_writer_close(writer, &err), OK) << err.message;
    ASSERT_NIL(streamer.close());
    synnax_client_close(client);
}
}
