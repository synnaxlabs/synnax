// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <sstream>

#include "gtest/gtest.h"

#include "x/cpp/errors/errors.h"

namespace x::errors {
/// @brief it should correctly parse an error from a string with type and data.
TEST(testerrors, testErrorConstructionFromString) {
    const std::string error = "sy.validation---invalid key: 1000: validation error";
    const auto err = Error(error);
    EXPECT_EQ(err.type, VALIDATION.type);
    EXPECT_EQ(err.data, "invalid key: 1000: validation error");
}

/// @brief it should correctly compare two equal errors.
TEST(testerrors, testErrorEqualsExactlyEqual) {
    const auto err1 = Error("test", "");
    const auto err2 = Error("test", "");
    ASSERT_EQ(err1, err2);
}

/// @brief it should match errors with the same type prefix.
TEST(testerrors, testErrorHequalHasPrefix) {
    const auto err1 = Error("test", "");
    const auto err2 = Error("test-specific", "");
    ASSERT_TRUE(err2.matches(err1));
}

/// @brief it should match errors against a vector of possible matches.
TEST(testerrors, testErrorMatchesVector) {
    const auto err = Error("test.specific.error", "");
    const std::vector errors = {
        Error("wrong", ""),
        Error("test.specific", ""),
        Error("another", "")
    };
    ASSERT_TRUE(err.matches(errors));
    const std::vector no_matches = {
        Error("wrong", ""),
        Error("other", ""),
        Error("another", "")
    };
    ASSERT_FALSE(err.matches(no_matches));
}

/// @brief it should create a nil error with default constructor.
TEST(testerrors, testDefaultConstructor) {
    const auto err = Error();
    ASSERT_EQ(err.type, TYPE_NIL);
    ASSERT_TRUE(err.data.empty());
    ASSERT_TRUE(err.ok());
}

/// @brief it should create an error with new data from an existing error.
TEST(testerrors, testConstructorWithErrorAndData) {
    const auto base_err = Error("base.error", "base data");
    const auto err = Error(base_err, "new data");
    ASSERT_EQ(err.type, "base.error");
    ASSERT_EQ(err.data, "new data");
}

/// @brief it should correctly construct an error from a protobuf message.
TEST(testerrors, testConstructorFromProtobuf) {
    ::errors::PBPayload pb_err;
    pb_err.set_type("protobuf.error");
    pb_err.set_data("protobuf data");
    const auto err = Error(pb_err);
    ASSERT_EQ(err.type, "protobuf.error");
    ASSERT_EQ(err.data, "protobuf data");
}

/// @brief it should create a sub-error with appended type.
TEST(testerrors, testSubMethod) {
    const auto base_err = Error("base", "");
    const auto sub_err = base_err.sub("child");
    ASSERT_EQ(sub_err.type, "base.child");
    ASSERT_TRUE(sub_err.data.empty());
}

/// @brief it should reparent an error to a new parent type.
TEST(testerrors, testReparentMethod) {
    const auto child_err = Error("old.parent.child", "child data");
    const auto new_parent = Error("new.parent", "");
    const auto reparented = child_err.reparent(new_parent);
    ASSERT_EQ(reparented.type, "new.parent.child");
    ASSERT_EQ(reparented.data, "child data");

    const auto no_dot_err = Error("nodot", "data");
    const auto unchanged = no_dot_err.reparent(new_parent);
    ASSERT_EQ(unchanged.type, "nodot");
    ASSERT_EQ(unchanged.data, "data");
}

/// @brief it should return true for ok() on nil errors and false otherwise.
TEST(testerrors, testOkMethod) {
    const auto nil_err = Error(TYPE_NIL, "");
    ASSERT_TRUE(nil_err.ok());

    const auto non_nil_err = Error("error", "");
    ASSERT_FALSE(non_nil_err.ok());
}

/// @brief it should format the error message correctly.
TEST(testerrors, testMessageMethod) {
    const auto err = Error("test.error", "error message");
    ASSERT_EQ(err.message(), "[test.error] error message");

    const auto no_data_err = Error("test.error", "");
    ASSERT_EQ(no_data_err.message(), "[test.error] ");
}

/// @brief it should convert to false for nil errors and true for non-nil errors.
TEST(testerrors, testBoolOperator) {
    const auto nil_err = Error(TYPE_NIL, "");
    ASSERT_FALSE(static_cast<bool>(nil_err));

    const auto non_nil_err = Error("error", "");
    ASSERT_TRUE(static_cast<bool>(non_nil_err));
}

/// @brief it should correctly stream the error to an output stream.
TEST(testerrors, testStreamOperator) {
    const auto err = Error("test.error", "error message");
    std::stringstream ss;
    ss << err;
    ASSERT_EQ(ss.str(), "[test.error] error message");
}

/// @brief it should match errors against string type prefixes.
TEST(testerrors, testMatchesString) {
    const auto err = Error("test.specific.error", "");
    ASSERT_TRUE(err.matches("test"));
    ASSERT_TRUE(err.matches("test.specific"));
    ASSERT_TRUE(err.matches("test.specific.error"));
    ASSERT_FALSE(err.matches("test.specific.error.more"));
    ASSERT_FALSE(err.matches("other"));
}

/// @brief it should correctly handle nil error matching behavior.
TEST(testerrors, testNilMatchesBehavior) {
    const auto nil_err = NIL;
    const auto specific_err = Error("test.specific.error", "");

    ASSERT_FALSE(nil_err.matches(specific_err));
    ASSERT_FALSE(nil_err.matches("test"));
    ASSERT_FALSE(nil_err.matches("test.specific.error"));

    ASSERT_FALSE(specific_err.matches(nil_err));
    ASSERT_FALSE(specific_err.matches(TYPE_NIL));

    ASSERT_TRUE(nil_err.matches(NIL));
    ASSERT_TRUE(nil_err.matches(TYPE_NIL));
}

/// @brief it should match errors against a vector of string types.
TEST(testerrors, testMatchesVectorStrings) {
    const auto err = Error("test.specific.error", "");
    const std::vector<std::string> types = {"wrong", "test.specific", "another"};
    ASSERT_TRUE(err.matches(types));

    const std::vector<std::string> no_match_types = {"wrong", "other", "another"};
    ASSERT_FALSE(err.matches(no_match_types));
}

/// @brief it should skip matching errors and return nil.
TEST(testerrors, testSkipSingleError) {
    const auto err = Error("test.error", "data");
    const auto skip_err = Error("test", "");

    const auto skipped = err.skip(skip_err);
    ASSERT_EQ(skipped.type, TYPE_NIL);
    ASSERT_TRUE(skipped.data.empty());

    const auto no_skip_err = Error("other", "");
    const auto not_skipped = err.skip(no_skip_err);
    ASSERT_EQ(not_skipped.type, "test.error");
    ASSERT_EQ(not_skipped.data, "data");
}

/// @brief it should skip errors matching any in a vector.
TEST(testerrors, testSkipVectorErrors) {
    const auto err = Error("test.error", "data");
    const std::vector<Error> skip_errors = {
        Error("wrong", ""),
        Error("test", ""),
        Error("another", "")
    };

    const auto skipped = err.skip(skip_errors);
    ASSERT_EQ(skipped.type, TYPE_NIL);
    ASSERT_TRUE(skipped.data.empty());

    const std::vector<Error> no_skip_errors = {
        Error("wrong", ""),
        Error("other", ""),
        Error("another", "")
    };
    const auto not_skipped = err.skip(no_skip_errors);
    ASSERT_EQ(not_skipped.type, "test.error");
    ASSERT_EQ(not_skipped.data, "data");
}

/// @brief it should skip errors matching a string type.
TEST(testerrors, testSkipString) {
    const auto err = Error("test.error", "data");

    const auto skipped = err.skip("test");
    ASSERT_EQ(skipped.type, TYPE_NIL);
    ASSERT_TRUE(skipped.data.empty());

    const auto not_skipped = err.skip("other");
    ASSERT_EQ(not_skipped.type, "test.error");
    ASSERT_EQ(not_skipped.data, "data");
}

/// @brief it should correctly compare errors for inequality.
TEST(testerrors, testNotEqualsOperator) {
    const auto err1 = Error("test1", "");
    const auto err2 = Error("test2", "");
    const auto err3 = Error("test1", "");

    ASSERT_TRUE(err1 != err2);
    ASSERT_FALSE(err1 != err3);
}

/// @brief it should correctly compare an error type to a string.
TEST(testerrors, testEqualsStringOperator) {
    const auto err = Error("test", "data");
    ASSERT_TRUE(err == "test");
    ASSERT_FALSE(err == "other");
}

/// @brief it should correctly compare an error type inequality to a string.
TEST(testerrors, testNotEqualsStringOperator) {
    const auto err = Error("test", "data");
    ASSERT_FALSE(err != "test");
    ASSERT_TRUE(err != "other");
}

/// @brief it should define all predefined error types correctly.
TEST(testerrors, testPredefinedErrors) {
    ASSERT_EQ(NIL.type, TYPE_NIL);
    ASSERT_EQ(UNKNOWN.type, TYPE_UNKNOWN);
    ASSERT_EQ(SY.type, "sy");
    ASSERT_EQ(VALIDATION.type, "sy.validation");
    ASSERT_EQ(QUERY.type, "sy.query");
    ASSERT_EQ(MULTIPLE_RESULTS.type, "sy.query.multiple_results");
    ASSERT_EQ(NOT_FOUND.type, "sy.query.not_found");
    ASSERT_EQ(NOT_SUPPORTED.type, "sy.not_supported");
    ASSERT_EQ(INTERNAL.type, "sy.internal");
    ASSERT_EQ(UNEXPECTED.type, "sy.unexpected");
    ASSERT_EQ(CONTROL.type, "sy.control");
    ASSERT_EQ(UNAUTHORIZED.type, "sy.control.unauthorized");
}

/// @brief it should parse a string without delimiter as type only.
TEST(testerrors, testStringConstructorWithoutDelimiter) {
    const auto err = Error("simple.error");
    ASSERT_EQ(err.type, "simple.error");
    ASSERT_TRUE(err.data.empty());
}

/// @brief it should correctly parse a string with delimiter into type and data.
TEST(testerrors, testStringConstructorWithDelimiter) {
    const auto err = Error("error.type---error data");
    ASSERT_EQ(err.type, "error.type");
    ASSERT_EQ(err.data, "error data");

    const auto multiple_delimiters = Error("error---data---more");
    ASSERT_EQ(multiple_delimiters.type, "error");
    ASSERT_EQ(multiple_delimiters.data, "data---more");
}
}
