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

TEST(ClassifyStatus, Status000ReturnsCriticalError) {
    ASSERT_OCCURRED_AS(classify_status(000), CRITICAL_ERROR);
}

TEST(ClassifyStatus, Status100ReturnsCriticalError) {
    ASSERT_OCCURRED_AS(classify_status(100), CRITICAL_ERROR);
}

TEST(ClassifyStatus, Status199ReturnsCriticalError) {
    ASSERT_OCCURRED_AS(classify_status(199), CRITICAL_ERROR);
}

TEST(ClassifyStatus, Status200ReturnsNil) {
    ASSERT_NIL(classify_status(200));
}

TEST(ClassifyStatus, Status201ReturnsNil) {
    ASSERT_NIL(classify_status(201));
}

TEST(ClassifyStatus, Status204ReturnsNil) {
    ASSERT_NIL(classify_status(204));
}

TEST(ClassifyStatus, Status299ReturnsNil) {
    ASSERT_NIL(classify_status(299));
}

TEST(ClassifyStatus, Status300ReturnsCriticalError) {
    ASSERT_OCCURRED_AS(classify_status(300), CRITICAL_ERROR);
}

TEST(ClassifyStatus, Status399ReturnsCriticalError) {
    ASSERT_OCCURRED_AS(classify_status(399), CRITICAL_ERROR);
}

TEST(ClassifyStatus, Status400ReturnsCriticalError) {
    ASSERT_OCCURRED_AS(classify_status(400), CRITICAL_ERROR);
}

TEST(ClassifyStatus, Status404ReturnsTemporaryError) {
    ASSERT_OCCURRED_AS(classify_status(404), TEMPORARY_ERROR);
}

TEST(ClassifyStatus, Status408ReturnsTemporaryError) {
    ASSERT_OCCURRED_AS(classify_status(408), TEMPORARY_ERROR);
}

TEST(ClassifyStatus, Status429ReturnsTemporaryError) {
    ASSERT_OCCURRED_AS(classify_status(429), TEMPORARY_ERROR);
}

TEST(ClassifyStatus, Status499ReturnsCriticalError) {
    ASSERT_OCCURRED_AS(classify_status(499), CRITICAL_ERROR);
}

TEST(ClassifyStatus, Status500ReturnsTemporaryError) {
    ASSERT_OCCURRED_AS(classify_status(500), TEMPORARY_ERROR);
}

TEST(ClassifyStatus, Status503ReturnsTemporaryError) {
    ASSERT_OCCURRED_AS(classify_status(503), TEMPORARY_ERROR);
}

TEST(ClassifyStatus, Status599ReturnsTemporaryError) {
    ASSERT_OCCURRED_AS(classify_status(599), TEMPORARY_ERROR);
}

TEST(ClassifyStatus, Status600ReturnsCriticalError) {
    ASSERT_OCCURRED_AS(classify_status(600), CRITICAL_ERROR);
}

TEST(ClassifyStatus, Status700ReturnsCriticalError) {
    ASSERT_OCCURRED_AS(classify_status(700), CRITICAL_ERROR);
}

TEST(ClassifyStatus, Status800ReturnsCriticalError) {
    ASSERT_OCCURRED_AS(classify_status(800), CRITICAL_ERROR);
}

TEST(ClassifyStatus, Status900ReturnsCriticalError) {
    ASSERT_OCCURRED_AS(classify_status(900), CRITICAL_ERROR);
}

}
