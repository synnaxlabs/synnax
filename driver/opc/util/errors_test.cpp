// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"
#include "open62541/types.h"

#include "x/cpp/xtest/xtest.h"

#include "driver/opc/util/util.h"

TEST(ErrorTest, testParseErrorGood) {
    auto err = util::parse_error(UA_STATUSCODE_GOOD);
    EXPECT_FALSE(err);
}

TEST(ErrorTest, testParseErrorConnectionTimeout) {
    auto err = util::parse_error(UA_STATUSCODE_BADTIMEOUT);
    EXPECT_TRUE(err);
    EXPECT_TRUE(err.matches(util::UNREACHABLE_ERROR));
    EXPECT_TRUE(err.matches(util::CRITICAL_ERROR));
}

TEST(ErrorTest, testParseErrorConnectionNotConnected) {
    auto err = util::parse_error(UA_STATUSCODE_BADNOTCONNECTED);
    EXPECT_TRUE(err);
    EXPECT_TRUE(err.matches(util::UNREACHABLE_ERROR));
}

TEST(ErrorTest, testParseErrorConnectionSecureChannelClosed) {
    auto err = util::parse_error(UA_STATUSCODE_BADSECURECHANNELCLOSED);
    EXPECT_TRUE(err);
    EXPECT_TRUE(err.matches(util::UNREACHABLE_ERROR));
}

TEST(ErrorTest, testParseErrorConnectionSessionInvalid) {
    auto err = util::parse_error(UA_STATUSCODE_BADSESSIONIDINVALID);
    EXPECT_TRUE(err);
    EXPECT_TRUE(err.matches(util::UNREACHABLE_ERROR));
}

TEST(ErrorTest, testParseErrorConnectionSessionClosed) {
    auto err = util::parse_error(UA_STATUSCODE_BADSESSIONCLOSED);
    EXPECT_TRUE(err);
    EXPECT_TRUE(err.matches(util::UNREACHABLE_ERROR));
}

TEST(ErrorTest, testParseErrorConnectionSessionNotActivated) {
    auto err = util::parse_error(UA_STATUSCODE_BADSESSIONNOTACTIVATED);
    EXPECT_TRUE(err);
    EXPECT_TRUE(err.matches(util::UNREACHABLE_ERROR));
}

TEST(ErrorTest, testParseErrorConnectionRejected) {
    auto err = util::parse_error(UA_STATUSCODE_BADCONNECTIONREJECTED);
    EXPECT_TRUE(err);
    EXPECT_TRUE(err.matches(util::UNREACHABLE_ERROR));
}

TEST(ErrorTest, testParseErrorDisconnect) {
    auto err = util::parse_error(UA_STATUSCODE_BADDISCONNECT);
    EXPECT_TRUE(err);
    EXPECT_TRUE(err.matches(util::UNREACHABLE_ERROR));
}

TEST(ErrorTest, testParseErrorConnectionClosed) {
    auto err = util::parse_error(UA_STATUSCODE_BADCONNECTIONCLOSED);
    EXPECT_TRUE(err);
    EXPECT_TRUE(err.matches(util::UNREACHABLE_ERROR));
}

TEST(ErrorTest, testParseErrorNonConnectionError) {
    auto err = util::parse_error(UA_STATUSCODE_BADOUTOFRANGE);
    EXPECT_TRUE(err);
    EXPECT_FALSE(err.matches(util::UNREACHABLE_ERROR));
    EXPECT_TRUE(err.matches(util::CRITICAL_ERROR));
}

TEST(ErrorTest, testParseErrorInvalidNodeId) {
    auto err = util::parse_error(UA_STATUSCODE_BADNODEIDUNKNOWN);
    EXPECT_TRUE(err);
    EXPECT_FALSE(err.matches(util::UNREACHABLE_ERROR));
    EXPECT_TRUE(err.matches(util::CRITICAL_ERROR));
}

TEST(ErrorTest, testParseErrorAccessDenied) {
    auto err = util::parse_error(UA_STATUSCODE_BADUSERACCESSDENIED);
    EXPECT_TRUE(err);
    EXPECT_FALSE(err.matches(util::UNREACHABLE_ERROR));
    EXPECT_TRUE(err.matches(util::CRITICAL_ERROR));
}
