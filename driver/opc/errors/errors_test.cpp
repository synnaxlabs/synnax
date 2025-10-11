// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// external
#include "gtest/gtest.h"
#include "open62541/types.h"

/// module
#include "x/cpp/xtest/xtest.h"

/// internal
#include "driver/opc/errors/errors.h"

TEST(ErrorTest, testParseErrorGood) {
    ASSERT_NIL(opc::errors::parse(UA_STATUSCODE_GOOD));
}

TEST(ErrorTest, testParseErrorConnectionTimeout) {
    ASSERT_OCCURRED_AS(
        opc::errors::parse(UA_STATUSCODE_BADTIMEOUT),
        opc::errors::UNREACHABLE
    );
}

TEST(ErrorTest, testParseErrorConnectionNotConnected) {
    ASSERT_OCCURRED_AS(
        opc::errors::parse(UA_STATUSCODE_BADNOTCONNECTED),
        opc::errors::UNREACHABLE
    );
}

TEST(ErrorTest, testParseErrorConnectionSecureChannelClosed) {
    ASSERT_OCCURRED_AS(
        opc::errors::parse(UA_STATUSCODE_BADSECURECHANNELCLOSED),
        opc::errors::UNREACHABLE
    );
}

TEST(ErrorTest, testParseErrorConnectionSessionInvalid) {
    ASSERT_OCCURRED_AS(
        opc::errors::parse(UA_STATUSCODE_BADSESSIONIDINVALID),
        opc::errors::UNREACHABLE
    );
}

TEST(ErrorTest, testParseErrorConnectionSessionClosed) {
    ASSERT_OCCURRED_AS(
        opc::errors::parse(UA_STATUSCODE_BADSESSIONCLOSED),
        opc::errors::UNREACHABLE
    );
}

TEST(ErrorTest, testParseErrorConnectionSessionNotActivated) {
    ASSERT_OCCURRED_AS(
        opc::errors::parse(UA_STATUSCODE_BADSESSIONNOTACTIVATED),
        opc::errors::UNREACHABLE
    );
}

TEST(ErrorTest, testParseErrorConnectionRejected) {
    ASSERT_OCCURRED_AS(
        opc::errors::parse(UA_STATUSCODE_BADCONNECTIONREJECTED),
        opc::errors::UNREACHABLE
    );
}

TEST(ErrorTest, testParseErrorDisconnect) {
    ASSERT_OCCURRED_AS(
        opc::errors::parse(UA_STATUSCODE_BADDISCONNECT),
        opc::errors::UNREACHABLE
    );
}

TEST(ErrorTest, testParseErrorConnectionClosed){ASSERT_OCCURRED_AS(
    opc::errors::parse(UA_STATUSCODE_BADCONNECTIONCLOSED),
    opc::errors::UNREACHABLE
)}

TEST(ErrorTest, testParseErrorNonConnectionError) {
    ASSERT_OCCURRED_AS(
        opc::errors::parse(UA_STATUSCODE_BADOUTOFRANGE),
        opc::errors::CRITICAL
    );
}

TEST(ErrorTest, testParseErrorInvalidNodeId) {
    ASSERT_OCCURRED_AS(
        opc::errors::parse(UA_STATUSCODE_BADNODEIDUNKNOWN),
        opc::errors::CRITICAL
    );
}

TEST(ErrorTest, testParseErrorAccessDenied) {
    ASSERT_OCCURRED_AS(
        opc::errors::parse(UA_STATUSCODE_BADUSERACCESSDENIED),
        opc::errors::CRITICAL
    )
}
