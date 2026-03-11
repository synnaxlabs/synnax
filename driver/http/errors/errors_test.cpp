// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/http/errors/errors.h"

namespace driver::http::errors {

#define STATUS_TEST(code, expected_err)                                                \
    TEST(FromStatus, Status##code##Returns##expected_err) {                            \
        ASSERT_OCCURRED_AS(from_status(code), expected_err);                           \
    }

#define STATUS_TEST_NIL(code)                                                          \
    TEST(FromStatus, Status##code##ReturnsNil) {                                       \
        ASSERT_NIL(from_status(code));                                                 \
    }

STATUS_TEST(000, CRITICAL_ERROR)
STATUS_TEST(100, CRITICAL_ERROR)
STATUS_TEST(199, CRITICAL_ERROR)

STATUS_TEST_NIL(200)
STATUS_TEST_NIL(201)
STATUS_TEST_NIL(204)
STATUS_TEST_NIL(299)

STATUS_TEST(300, CRITICAL_ERROR)
STATUS_TEST(399, CRITICAL_ERROR)
STATUS_TEST(400, CRITICAL_ERROR)
STATUS_TEST(404, CRITICAL_ERROR)
STATUS_TEST(408, TEMPORARY_ERROR)
STATUS_TEST(429, TEMPORARY_ERROR)
STATUS_TEST(499, CRITICAL_ERROR)
STATUS_TEST(500, TEMPORARY_ERROR)
STATUS_TEST(503, TEMPORARY_ERROR)
STATUS_TEST(599, TEMPORARY_ERROR)
STATUS_TEST(600, CRITICAL_ERROR)
STATUS_TEST(700, CRITICAL_ERROR)
STATUS_TEST(800, CRITICAL_ERROR)
STATUS_TEST(900, CRITICAL_ERROR)

}
