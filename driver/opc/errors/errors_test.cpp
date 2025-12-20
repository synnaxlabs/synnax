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

#include "driver/opc/errors/errors.h"

/// @brief it should return nil error for good status code.
TEST(ErrorTest, testParseErrorGood) {
    ASSERT_NIL(opc::errors::parse(UA_STATUSCODE_GOOD));
}

/// @brief it should map timeout status to unreachable error.
TEST(ErrorTest, testParseErrorConnectionTimeout) {
    ASSERT_OCCURRED_AS(
        opc::errors::parse(UA_STATUSCODE_BADTIMEOUT),
        opc::errors::UNREACHABLE
    );
}

/// @brief it should map not connected status to unreachable error.
TEST(ErrorTest, testParseErrorConnectionNotConnected) {
    ASSERT_OCCURRED_AS(
        opc::errors::parse(UA_STATUSCODE_BADNOTCONNECTED),
        opc::errors::UNREACHABLE
    );
}

/// @brief it should map secure channel closed status to unreachable error.
TEST(ErrorTest, testParseErrorConnectionSecureChannelClosed) {
    ASSERT_OCCURRED_AS(
        opc::errors::parse(UA_STATUSCODE_BADSECURECHANNELCLOSED),
        opc::errors::UNREACHABLE
    );
}

/// @brief it should map session invalid status to unreachable error.
TEST(ErrorTest, testParseErrorConnectionSessionInvalid) {
    ASSERT_OCCURRED_AS(
        opc::errors::parse(UA_STATUSCODE_BADSESSIONIDINVALID),
        opc::errors::UNREACHABLE
    );
}

/// @brief it should map session closed status to unreachable error.
TEST(ErrorTest, testParseErrorConnectionSessionClosed) {
    ASSERT_OCCURRED_AS(
        opc::errors::parse(UA_STATUSCODE_BADSESSIONCLOSED),
        opc::errors::UNREACHABLE
    );
}

/// @brief it should map session not activated status to unreachable error.
TEST(ErrorTest, testParseErrorConnectionSessionNotActivated) {
    ASSERT_OCCURRED_AS(
        opc::errors::parse(UA_STATUSCODE_BADSESSIONNOTACTIVATED),
        opc::errors::UNREACHABLE
    );
}

/// @brief it should map connection rejected status to unreachable error.
TEST(ErrorTest, testParseErrorConnectionRejected) {
    ASSERT_OCCURRED_AS(
        opc::errors::parse(UA_STATUSCODE_BADCONNECTIONREJECTED),
        opc::errors::UNREACHABLE
    );
}

/// @brief it should map disconnect status to unreachable error.
TEST(ErrorTest, testParseErrorDisconnect) {
    ASSERT_OCCURRED_AS(
        opc::errors::parse(UA_STATUSCODE_BADDISCONNECT),
        opc::errors::UNREACHABLE
    );
}

/// @brief it should map connection closed status to unreachable error.
TEST(ErrorTest, testParseErrorConnectionClosed){ASSERT_OCCURRED_AS(
    opc::errors::parse(UA_STATUSCODE_BADCONNECTIONCLOSED),
    opc::errors::UNREACHABLE
)}

/// @brief it should map out of range status to critical error.
TEST(ErrorTest, testParseErrorNonConnectionError) {
    ASSERT_OCCURRED_AS(
        opc::errors::parse(UA_STATUSCODE_BADOUTOFRANGE),
        opc::errors::CRITICAL
    );
}

/// @brief it should map unknown node id status to critical error.
TEST(ErrorTest, testParseErrorInvalidNodeId) {
    ASSERT_OCCURRED_AS(
        opc::errors::parse(UA_STATUSCODE_BADNODEIDUNKNOWN),
        opc::errors::CRITICAL
    );
}

/// @brief it should map user access denied status to critical error.
TEST(ErrorTest, testParseErrorAccessDenied) {
    ASSERT_OCCURRED_AS(
        opc::errors::parse(UA_STATUSCODE_BADUSERACCESSDENIED),
        opc::errors::CRITICAL
    )
}
