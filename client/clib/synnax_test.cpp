// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstddef>
#include <cstring>

#include "gtest/gtest.h"

#include "client/clib/synnax.h"

namespace synnax::clib {
constexpr int32_t OK = 0;

/// @brief it should keep the SynnaxError byte layout the LabVIEW Array Data Pointer
/// parse depends on: 644 bytes, code at 0, type at 4, message at 132.
TEST(ClibError, testSynnaxErrorLayoutMatchesLabVIEWContract) {
    EXPECT_EQ(sizeof(SynnaxError), 644u);
    EXPECT_EQ(offsetof(SynnaxError, code), 0u);
    EXPECT_EQ(offsetof(SynnaxError, type), 4u);
    EXPECT_EQ(offsetof(SynnaxError, message), 132u);
}

/// @brief it should return a non-empty version string.
TEST(ClibClient, testReturnsNonEmptyVersion) {
    const char *version = synnax_client_version();
    ASSERT_NE(version, nullptr);
    EXPECT_GT(std::strlen(version), 0u);
}

/// @brief it should reject a client open with a null out_client pointer.
TEST(ClibClient, testOpenWithNullOutClientReturnsValidationError) {
    SynnaxError err;
    const int32_t code = synnax_client_open(
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
        nullptr,
        &err
    );
    EXPECT_NE(code, OK);
    EXPECT_EQ(err.code, code);
    EXPECT_STREQ(err.type, "sy.validation");
}

/// @brief it should reject a secure client open that omits the CA certificate.
TEST(ClibClient, testOpenSecureWithoutCACertReturnsValidationError) {
    SynnaxError err;
    SynnaxClient *client = nullptr;
    const int32_t code = synnax_client_open(
        "localhost",
        9090,
        "synnax",
        "seldon",
        1,
        nullptr,
        nullptr,
        nullptr,
        0,
        0,
        &client,
        &err
    );
    EXPECT_NE(code, OK);
    EXPECT_EQ(client, nullptr);
    EXPECT_STREQ(err.type, "sy.validation");
    EXPECT_NE(std::strstr(err.message, "ca_cert_file"), nullptr);
}

/// @brief it should tolerate a null error pointer on a failing client open.
TEST(ClibClient, testOpenWithNullErrorDoesNotCrash) {
    EXPECT_NE(
        synnax_client_open(
            "localhost",
            9090,
            "synnax",
            "seldon",
            1,
            nullptr,
            nullptr,
            nullptr,
            0,
            0,
            nullptr,
            nullptr
        ),
        OK
    );
}

/// @brief it should safely ignore a close on a null client.
TEST(ClibClient, testCloseClientOnNullIsSafe) {
    synnax_client_close(nullptr);
}

/// @brief it should connect to the local test cluster and return a handle.
TEST(ClibClient, testConnectsToLocalCluster) {
    SynnaxError err;
    SynnaxClient *client = nullptr;
    const int32_t code = synnax_client_open(
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
        &err
    );
    ASSERT_EQ(code, OK) << err.message;
    ASSERT_NE(client, nullptr);
    synnax_client_close(client);
}

/// @brief it should report a connection error and leave out_client null (without
/// leaking) when the target cluster is unreachable.
TEST(ClibClient, testOpenAgainstUnreachableClusterReturnsError) {
    SynnaxError err;
    SynnaxClient *client = nullptr;
    const int32_t code = synnax_client_open(
        "localhost",
        59999,
        "synnax",
        "seldon",
        0,
        nullptr,
        nullptr,
        nullptr,
        1,
        0,
        &client,
        &err
    );
    EXPECT_NE(code, OK);
    EXPECT_EQ(client, nullptr);
}
}
