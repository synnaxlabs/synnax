// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstring>
#include <string>

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

/// @brief it should reject a channel retrieve with a null client.
TEST(ClibChannel, testRetrieveWithNullClientReturnsValidationError) {
    SynnaxError err;
    uint32_t keys[1] = {0};
    const int32_t code = synnax_channel_retrieve_keys(
        nullptr,
        "name",
        1,
        keys,
        nullptr,
        nullptr,
        0,
        &err
    );
    EXPECT_NE(code, OK);
    EXPECT_STREQ(err.type, "sy.validation");
}

/// @brief it should reject a retrieve whose name_count disagrees with the names string.
TEST(ClibChannel, testRejectsNameCountMismatch) {
    SynnaxError err;
    SynnaxClient *client = open_test_client(&err);
    ASSERT_NE(client, nullptr) << err.message;

    uint32_t keys[3] = {0, 0, 0};
    const int32_t code = synnax_channel_retrieve_keys(
        client,
        "a\nb",
        3,
        keys,
        nullptr,
        nullptr,
        0,
        &err
    );
    EXPECT_NE(code, OK);
    EXPECT_STREQ(err.type, "sy.validation");
    EXPECT_NE(std::strstr(err.message, "name_count"), nullptr);

    synnax_client_close(client);
}

/// @brief it should resolve names to their keys, index keys, and data types.
TEST(ClibChannel, testResolvesKeysIndexAndDataTypes) {
    auto cpp_client = new_test_client();
    auto [index, data] = create_indexed_pair(cpp_client);

    SynnaxError err;
    SynnaxClient *client = open_test_client(&err);
    ASSERT_NE(client, nullptr) << err.message;

    const std::string names = index.name + "\n" + data.name;
    uint32_t keys[2] = {0, 0};
    uint32_t index_keys[2] = {0, 0};
    char dtypes[128] = {0};
    const int32_t code = synnax_channel_retrieve_keys(
        client,
        names.c_str(),
        2,
        keys,
        index_keys,
        dtypes,
        sizeof(dtypes),
        &err
    );
    ASSERT_EQ(code, OK) << err.message;
    EXPECT_EQ(keys[0], index.key);
    EXPECT_EQ(keys[1], data.key);
    EXPECT_EQ(index_keys[0], index.index);
    EXPECT_EQ(index_keys[1], data.index);
    const std::string expected_dtypes = index.data_type.name() + "\n" +
                                        data.data_type.name();
    EXPECT_STREQ(dtypes, expected_dtypes.c_str());

    synnax_client_close(client);
}

/// @brief it should report a not-found error for a missing channel while keeping the
/// data type slots aligned with the requested names.
TEST(ClibChannel, testReportsMissingChannelWithAlignedDataTypes) {
    auto cpp_client = new_test_client();
    auto [index, data] = create_indexed_pair(cpp_client);
    const std::string missing = make_unique_channel_name("missing");

    SynnaxError err;
    SynnaxClient *client = open_test_client(&err);
    ASSERT_NE(client, nullptr) << err.message;

    const std::string names = data.name + "\n" + missing + "\n" + index.name;
    uint32_t keys[3] = {9, 9, 9};
    uint32_t index_keys[3] = {9, 9, 9};
    char dtypes[128] = {0};
    const int32_t code = synnax_channel_retrieve_keys(
        client,
        names.c_str(),
        3,
        keys,
        index_keys,
        dtypes,
        sizeof(dtypes),
        &err
    );
    EXPECT_NE(code, OK);
    EXPECT_NE(std::strstr(err.type, "not_found"), nullptr);
    EXPECT_NE(std::strstr(err.message, missing.c_str()), nullptr);

    EXPECT_EQ(keys[0], data.key);
    EXPECT_EQ(keys[1], 0u);
    EXPECT_EQ(keys[2], index.key);

    const std::string expected_dtypes = data.data_type.name() + "\n\n" +
                                        index.data_type.name();
    EXPECT_STREQ(dtypes, expected_dtypes.c_str());

    synnax_client_close(client);
}

/// @brief it should reject a retrieve whose out_dtypes buffer cannot hold the joined
/// data type names.
TEST(ClibChannel, testRejectsOutDataTypesBufferTooSmall) {
    auto cpp_client = new_test_client();
    auto [index, data] = create_indexed_pair(cpp_client);

    SynnaxError err;
    SynnaxClient *client = open_test_client(&err);
    ASSERT_NE(client, nullptr) << err.message;

    const std::string names = index.name + "\n" + data.name;
    uint32_t keys[2] = {0, 0};
    uint32_t index_keys[2] = {0, 0};
    char dtypes[4] = {0};
    const int32_t code = synnax_channel_retrieve_keys(
        client,
        names.c_str(),
        2,
        keys,
        index_keys,
        dtypes,
        sizeof(dtypes),
        &err
    );
    EXPECT_NE(code, OK);
    EXPECT_STREQ(err.type, "sy.validation");
    EXPECT_NE(std::strstr(err.message, "out_dtypes too small"), nullptr);

    synnax_client_close(client);
}

/// @brief it should resolve keys when the optional out_index_keys and out_dtypes
/// outputs are null.
TEST(ClibChannel, testRetrievesWithNullOptionalOutputs) {
    auto cpp_client = new_test_client();
    auto [index, data] = create_indexed_pair(cpp_client);

    SynnaxError err;
    SynnaxClient *client = open_test_client(&err);
    ASSERT_NE(client, nullptr) << err.message;

    const std::string names = index.name + "\n" + data.name;
    uint32_t keys[2] = {0, 0};
    const int32_t code = synnax_channel_retrieve_keys(
        client,
        names.c_str(),
        2,
        keys,
        nullptr,
        nullptr,
        0,
        &err
    );
    ASSERT_EQ(code, OK) << err.message;
    EXPECT_EQ(keys[0], index.key);
    EXPECT_EQ(keys[1], data.key);

    synnax_client_close(client);
}

/// @brief it should fill a raw 644-byte buffer, as LabVIEW passes it, so the code, type,
/// and message can be read back at their fixed offsets.
TEST(ClibChannel, testRetrieveFillsErrorByteBufferForLabVIEW) {
    uint8_t buf[644] = {0};
    uint32_t keys[1] = {0};
    const int32_t code = synnax_channel_retrieve_keys(
        nullptr,
        "name",
        1,
        keys,
        nullptr,
        nullptr,
        0,
        reinterpret_cast<SynnaxError *>(buf)
    );
    EXPECT_NE(code, OK);
    int32_t err_code = 0;
    std::memcpy(&err_code, buf, sizeof(err_code));
    EXPECT_EQ(err_code, code);
    EXPECT_STREQ(reinterpret_cast<const char *>(buf + 4), "sy.validation");
    EXPECT_STREQ(
        reinterpret_cast<const char *>(buf + 132),
        "null client, names, or out_keys"
    );
}

/// @brief it should reject a channel create with a null client.
TEST(ClibChannel, testCreateWithNullClientReturnsValidationError) {
    SynnaxError err;
    uint32_t keys[1] = {0};
    const uint8_t is_index[1] = {0};
    const uint32_t index[1] = {0};
    const uint8_t is_virtual[1] = {0};
    const int32_t code = synnax_channel_create(
        nullptr,
        "ch",
        "float32",
        is_index,
        index,
        is_virtual,
        1,
        keys,
        &err
    );
    EXPECT_NE(code, OK);
    EXPECT_STREQ(err.type, "sy.validation");
}

/// @brief it should reject a create whose count disagrees with the names string.
TEST(ClibChannel, testCreateRejectsCountMismatch) {
    SynnaxError err;
    SynnaxClient *client = open_test_client(&err);
    ASSERT_NE(client, nullptr) << err.message;

    uint32_t keys[2] = {0, 0};
    const uint8_t is_index[2] = {1, 1};
    const uint32_t index[2] = {0, 0};
    const uint8_t is_virtual[2] = {0, 0};
    const int32_t code = synnax_channel_create(
        client,
        "a\nb",
        "timestamp\ntimestamp",
        is_index,
        index,
        is_virtual,
        3,
        keys,
        &err
    );
    EXPECT_NE(code, OK);
    EXPECT_STREQ(err.type, "sy.validation");

    synnax_client_close(client);
}

/// @brief it should create a single index channel (count 1) and assign a non-zero key.
TEST(ClibChannel, testCreateSingleIndexChannel) {
    SynnaxError err;
    SynnaxClient *client = open_test_client(&err);
    ASSERT_NE(client, nullptr) << err.message;

    const std::string name = make_unique_channel_name("idx");
    uint32_t keys[1] = {0};
    const uint8_t is_index[1] = {1};
    const uint32_t index[1] = {0};
    const uint8_t is_virtual[1] = {0};
    const int32_t code = synnax_channel_create(
        client,
        name.c_str(),
        "timestamp",
        is_index,
        index,
        is_virtual,
        1,
        keys,
        &err
    );
    ASSERT_EQ(code, OK) << err.message;
    EXPECT_NE(keys[0], 0u);

    synnax_client_close(client);
}

/// @brief it should create multiple channels in a single call and assign each a key.
TEST(ClibChannel, testCreateManyIndexChannels) {
    SynnaxError err;
    SynnaxClient *client = open_test_client(&err);
    ASSERT_NE(client, nullptr) << err.message;

    const std::string names =
        make_unique_channel_name("idx") + "\n" + make_unique_channel_name("idx");
    uint32_t keys[2] = {0, 0};
    const uint8_t is_index[2] = {1, 1};
    const uint32_t index[2] = {0, 0};
    const uint8_t is_virtual[2] = {0, 0};
    const int32_t code = synnax_channel_create(
        client,
        names.c_str(),
        "timestamp\ntimestamp",
        is_index,
        index,
        is_virtual,
        2,
        keys,
        &err
    );
    ASSERT_EQ(code, OK) << err.message;
    EXPECT_NE(keys[0], 0u);
    EXPECT_NE(keys[1], 0u);

    synnax_client_close(client);
}

/// @brief it should create an indexed data channel referencing a created index.
TEST(ClibChannel, testCreateIndexedDataChannel) {
    SynnaxError err;
    SynnaxClient *client = open_test_client(&err);
    ASSERT_NE(client, nullptr) << err.message;

    const std::string idx_name = make_unique_channel_name("idx");
    uint32_t idx_key[1] = {0};
    const uint8_t idx_is_index[1] = {1};
    const uint32_t idx_index[1] = {0};
    const uint8_t idx_is_virtual[1] = {0};
    ASSERT_EQ(
        synnax_channel_create(
            client,
            idx_name.c_str(),
            "timestamp",
            idx_is_index,
            idx_index,
            idx_is_virtual,
            1,
            idx_key,
            &err
        ),
        OK
    ) << err.message;

    const std::string data_name = make_unique_channel_name("data");
    uint32_t data_key[1] = {0};
    const uint8_t data_is_index[1] = {0};
    const uint32_t data_index[1] = {idx_key[0]};
    const uint8_t data_is_virtual[1] = {0};
    const int32_t code = synnax_channel_create(
        client,
        data_name.c_str(),
        "float32",
        data_is_index,
        data_index,
        data_is_virtual,
        1,
        data_key,
        &err
    );
    ASSERT_EQ(code, OK) << err.message;
    EXPECT_NE(data_key[0], 0u);

    synnax_client_close(client);
}
}
