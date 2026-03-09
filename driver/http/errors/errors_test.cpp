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

TEST(FromStatus, Status000ReturnsCriticalError) {
    ASSERT_OCCURRED_AS(from_status(000), CRITICAL_ERROR);
}

TEST(FromStatus, Status100ReturnsCriticalError) {
    ASSERT_OCCURRED_AS(from_status(100), CRITICAL_ERROR);
}

TEST(FromStatus, Status199ReturnsCriticalError) {
    ASSERT_OCCURRED_AS(from_status(199), CRITICAL_ERROR);
}

TEST(FromStatus, Status200ReturnsNil) {
    ASSERT_NIL(from_status(200));
}

TEST(FromStatus, Status201ReturnsNil) {
    ASSERT_NIL(from_status(201));
}

TEST(FromStatus, Status204ReturnsNil) {
    ASSERT_NIL(from_status(204));
}

TEST(FromStatus, Status299ReturnsNil) {
    ASSERT_NIL(from_status(299));
}

TEST(FromStatus, Status300ReturnsCriticalError) {
    ASSERT_OCCURRED_AS(from_status(300), CRITICAL_ERROR);
}

TEST(FromStatus, Status399ReturnsCriticalError) {
    ASSERT_OCCURRED_AS(from_status(399), CRITICAL_ERROR);
}

TEST(FromStatus, Status400ReturnsCriticalError) {
    ASSERT_OCCURRED_AS(from_status(400), CRITICAL_ERROR);
}

TEST(FromStatus, Status404ReturnsTemporaryError) {
    ASSERT_OCCURRED_AS(from_status(404), TEMPORARY_ERROR);
}

TEST(FromStatus, Status408ReturnsTemporaryError) {
    ASSERT_OCCURRED_AS(from_status(408), TEMPORARY_ERROR);
}

TEST(FromStatus, Status429ReturnsTemporaryError) {
    ASSERT_OCCURRED_AS(from_status(429), TEMPORARY_ERROR);
}

TEST(FromStatus, Status499ReturnsCriticalError) {
    ASSERT_OCCURRED_AS(from_status(499), CRITICAL_ERROR);
}

TEST(FromStatus, Status500ReturnsTemporaryError) {
    ASSERT_OCCURRED_AS(from_status(500), TEMPORARY_ERROR);
}

TEST(FromStatus, Status503ReturnsTemporaryError) {
    ASSERT_OCCURRED_AS(from_status(503), TEMPORARY_ERROR);
}

TEST(FromStatus, Status599ReturnsTemporaryError) {
    ASSERT_OCCURRED_AS(from_status(599), TEMPORARY_ERROR);
}

TEST(FromStatus, Status600ReturnsCriticalError) {
    ASSERT_OCCURRED_AS(from_status(600), CRITICAL_ERROR);
}

TEST(FromStatus, Status700ReturnsCriticalError) {
    ASSERT_OCCURRED_AS(from_status(700), CRITICAL_ERROR);
}

TEST(FromStatus, Status800ReturnsCriticalError) {
    ASSERT_OCCURRED_AS(from_status(800), CRITICAL_ERROR);
}

TEST(FromStatus, Status900ReturnsCriticalError) {
    ASSERT_OCCURRED_AS(from_status(900), CRITICAL_ERROR);
}

}
