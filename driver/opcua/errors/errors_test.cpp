// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"
#include "open62541/types.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/opcua/errors/errors.h"

namespace driver::opcua::errors {
/// @brief it should return nil error for good status code.
TEST(ErrorTest, testParseErrorGood) {
    ASSERT_NIL(parse(UA_STATUSCODE_GOOD));
}

/// @brief it should map timeout status to unreachable error.
TEST(ErrorTest, testParseErrorConnectionTimeout) {
    ASSERT_OCCURRED_AS(parse(UA_STATUSCODE_BADTIMEOUT), UNREACHABLE);
}

/// @brief it should map not connected status to unreachable error.
TEST(ErrorTest, testParseErrorConnectionNotConnected) {
    ASSERT_OCCURRED_AS(parse(UA_STATUSCODE_BADNOTCONNECTED), UNREACHABLE);
}

/// @brief it should map secure channel closed status to unreachable error.
TEST(ErrorTest, testParseErrorConnectionSecureChannelClosed) {
    ASSERT_OCCURRED_AS(parse(UA_STATUSCODE_BADSECURECHANNELCLOSED), UNREACHABLE);
}

/// @brief it should map session invalid status to unreachable error.
TEST(ErrorTest, testParseErrorConnectionSessionInvalid) {
    ASSERT_OCCURRED_AS(parse(UA_STATUSCODE_BADSESSIONIDINVALID), UNREACHABLE);
}

/// @brief it should map session closed status to unreachable error.
TEST(ErrorTest, testParseErrorConnectionSessionClosed) {
    ASSERT_OCCURRED_AS(parse(UA_STATUSCODE_BADSESSIONCLOSED), UNREACHABLE);
}

/// @brief it should map session not activated status to unreachable error.
TEST(ErrorTest, testParseErrorConnectionSessionNotActivated) {
    ASSERT_OCCURRED_AS(parse(UA_STATUSCODE_BADSESSIONNOTACTIVATED), UNREACHABLE);
}

/// @brief it should map connection rejected status to unreachable error.
TEST(ErrorTest, testParseErrorConnectionRejected) {
    ASSERT_OCCURRED_AS(parse(UA_STATUSCODE_BADCONNECTIONREJECTED), UNREACHABLE);
}

/// @brief it should map disconnect status to unreachable error.
TEST(ErrorTest, testParseErrorDisconnect) {
    ASSERT_OCCURRED_AS(parse(UA_STATUSCODE_BADDISCONNECT), UNREACHABLE);
}

/// @brief it should map connection closed status to unreachable error.
TEST(ErrorTest, testParseErrorConnectionClosed) {
    ASSERT_OCCURRED_AS(parse(UA_STATUSCODE_BADCONNECTIONCLOSED), UNREACHABLE);
}

/// @brief it should map a connection status with flag bits set to unreachable error.
TEST(ErrorTest, ParseErrorConnectionCodeWithFlags) {
    ASSERT_OCCURRED_AS(parse(UA_STATUSCODE_BADTIMEOUT | 0x0400), UNREACHABLE);
}

/// @brief it should map a lost connection to a temporary error.
TEST(ErrorTest, ParseConnectionLossIsTemporary) {
    ASSERT_MATCHES(
        parse(UA_STATUSCODE_BADCONNECTIONCLOSED),
        driver::errors::TEMPORARY_HARDWARE_ERROR
    );
}

/// @brief it should map a server shutdown to unreachable error.
TEST(ErrorTest, ParseServerShutdown) {
    ASSERT_OCCURRED_AS(parse(UA_STATUSCODE_BADSHUTDOWN), UNREACHABLE);
    ASSERT_OCCURRED_AS(parse(UA_STATUSCODE_BADSERVERHALTED), UNREACHABLE);
}

/// @brief it should map a request timeout to unreachable error.
TEST(ErrorTest, ParseRequestTimeout) {
    ASSERT_OCCURRED_AS(parse(UA_STATUSCODE_BADREQUESTTIMEOUT), UNREACHABLE);
}

/// @brief it should map a rejected login at connect to a session rejection.
TEST(ErrorTest, ParseConnectLoginRejected) {
    ASSERT_OCCURRED_AS(
        parse_connect(UA_STATUSCODE_BADUSERACCESSDENIED),
        SESSION_REJECTED
    );
    ASSERT_OCCURRED_AS(
        parse_connect(UA_STATUSCODE_BADIDENTITYTOKENREJECTED),
        SESSION_REJECTED
    );
    ASSERT_MATCHES(
        parse_connect(UA_STATUSCODE_BADIDENTITYTOKENREJECTED),
        driver::errors::TEMPORARY_HARDWARE_ERROR
    );
}

/// @brief it should map a full session table at connect to a session rejection.
TEST(ErrorTest, ParseConnectTooManySessions) {
    ASSERT_OCCURRED_AS(
        parse_connect(UA_STATUSCODE_BADTOOMANYSESSIONS),
        SESSION_REJECTED
    );
}

/// @brief it should map an invalid endpoint at connect to a critical error.
TEST(ErrorTest, ParseConnectInvalidEndpoint) {
    ASSERT_OCCURRED_AS(
        parse_connect(UA_STATUSCODE_BADTCPENDPOINTURLINVALID),
        INVALID_ENDPOINT
    );
}

/// @brief it should map out of range status to critical error.
TEST(ErrorTest, testParseErrorNonConnectionError) {
    ASSERT_OCCURRED_AS(parse(UA_STATUSCODE_BADOUTOFRANGE), CRITICAL);
}

/// @brief it should map unknown node id status to critical error.
TEST(ErrorTest, testParseErrorInvalidNodeId) {
    ASSERT_OCCURRED_AS(parse(UA_STATUSCODE_BADNODEIDUNKNOWN), CRITICAL);
}

/// @brief it should map user access denied status to critical error.
TEST(ErrorTest, testParseErrorAccessDenied) {
    ASSERT_OCCURRED_AS(parse(UA_STATUSCODE_BADUSERACCESSDENIED), CRITICAL);
}

/// @brief it should describe a status code by its condition when flag bits are set.
TEST(ErrorTest, ParseErrorDescribesCodeWithFlags) {
    EXPECT_EQ(
        parse(UA_STATUSCODE_BADTIMEOUT | 0x0400).data,
        "The operation timed out."
    );
}

/// @brief it should describe an unknown status code by its name and hex value.
TEST(ErrorTest, ParseErrorDescribesUnknownCode) {
    EXPECT_EQ(
        parse(0x80FF0000).data,
        "The server returned Unknown StatusCode (0x80FF0000)."
    );
}

/// @brief it should name the channel, the status, and the reason for a bad status.
TEST(ErrorTest, ChannelStatusBad) {
    EXPECT_EQ(
        channel_status("valve", UA_STATUSCODE_BADNOTWRITABLE),
        "valve returned BadNotWritable. The access level does not allow writing to the "
        "Node."
    );
}

/// @brief it should name the channel and the status without a reason for an uncertain
/// status.
TEST(ErrorTest, ChannelStatusUncertain) {
    EXPECT_EQ(
        channel_status("pressure", UA_STATUSCODE_UNCERTAINLASTUSABLEVALUE),
        "pressure returned UncertainLastUsableValue."
    );
}
}
