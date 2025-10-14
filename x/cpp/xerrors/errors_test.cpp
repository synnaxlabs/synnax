// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <sstream>

#include "gtest/gtest.h"

#include "x/cpp/xerrors/errors.h"

TEST(testXErrors, testErrorConstructionFromString) {
    const std::string error = "sy.validation---invalid key: 1000: validation error";
    auto err = xerrors::Error(error);
}

TEST(testXErrors, testErrorEqualsExactlyEqual) {
    const auto err1 = xerrors::Error("test", "");
    const auto err2 = xerrors::Error("test", "");
    ASSERT_EQ(err1, err2);
}

TEST(testXErrors, testErrorHequalHasPrefix) {
    const auto err1 = xerrors::Error("test", "");
    const auto err2 = xerrors::Error("test-specific", "");
    ASSERT_TRUE(err2.matches(err1));
}

TEST(testXErrors, testErrorMatchesVector) {
    const auto err = xerrors::Error("test.specific.error", "");
    const std::vector errors = {
        xerrors::Error("wrong", ""),
        xerrors::Error("test.specific", ""),
        xerrors::Error("another", "")
    };
    ASSERT_TRUE(err.matches(errors));
    const std::vector no_matches = {
        xerrors::Error("wrong", ""),
        xerrors::Error("other", ""),
        xerrors::Error("another", "")
    };
    ASSERT_FALSE(err.matches(no_matches));
}

TEST(testXErrors, testDefaultConstructor) {
    const auto err = xerrors::Error();
    ASSERT_EQ(err.type, xerrors::TYPE_NIL);
    ASSERT_TRUE(err.data.empty());
    ASSERT_TRUE(err.ok());
}

TEST(testXErrors, testConstructorWithErrorAndData) {
    const auto base_err = xerrors::Error("base.error", "base data");
    const auto err = xerrors::Error(base_err, "new data");
    ASSERT_EQ(err.type, "base.error");
    ASSERT_EQ(err.data, "new data");
}

TEST(testXErrors, testConstructorFromProtobuf) {
    errors::PBPayload pb_err;
    pb_err.set_type("protobuf.error");
    pb_err.set_data("protobuf data");
    const auto err = xerrors::Error(pb_err);
    ASSERT_EQ(err.type, "protobuf.error");
    ASSERT_EQ(err.data, "protobuf data");
}

TEST(testXErrors, testSubMethod) {
    const auto base_err = xerrors::Error("base", "");
    const auto sub_err = base_err.sub("child");
    ASSERT_EQ(sub_err.type, "base.child");
    ASSERT_TRUE(sub_err.data.empty());
}

TEST(testXErrors, testReparentMethod) {
    const auto child_err = xerrors::Error("old.parent.child", "child data");
    const auto new_parent = xerrors::Error("new.parent", "");
    const auto reparented = child_err.reparent(new_parent);
    ASSERT_EQ(reparented.type, "new.parent.child");
    ASSERT_EQ(reparented.data, "child data");

    const auto no_dot_err = xerrors::Error("nodot", "data");
    const auto unchanged = no_dot_err.reparent(new_parent);
    ASSERT_EQ(unchanged.type, "nodot");
    ASSERT_EQ(unchanged.data, "data");
}

TEST(testXErrors, testOkMethod) {
    const auto nil_err = xerrors::Error(xerrors::TYPE_NIL, "");
    ASSERT_TRUE(nil_err.ok());

    const auto non_nil_err = xerrors::Error("error", "");
    ASSERT_FALSE(non_nil_err.ok());
}

TEST(testXErrors, testMessageMethod) {
    const auto err = xerrors::Error("test.error", "error message");
    ASSERT_EQ(err.message(), "[test.error] error message");

    const auto no_data_err = xerrors::Error("test.error", "");
    ASSERT_EQ(no_data_err.message(), "[test.error] ");
}

TEST(testXErrors, testBoolOperator) {
    const auto nil_err = xerrors::Error(xerrors::TYPE_NIL, "");
    ASSERT_FALSE(static_cast<bool>(nil_err));

    const auto non_nil_err = xerrors::Error("error", "");
    ASSERT_TRUE(static_cast<bool>(non_nil_err));
}

TEST(testXErrors, testStreamOperator) {
    const auto err = xerrors::Error("test.error", "error message");
    std::stringstream ss;
    ss << err;
    ASSERT_EQ(ss.str(), "[test.error] error message");
}

TEST(testXErrors, testMatchesString) {
    const auto err = xerrors::Error("test.specific.error", "");
    ASSERT_TRUE(err.matches("test"));
    ASSERT_TRUE(err.matches("test.specific"));
    ASSERT_TRUE(err.matches("test.specific.error"));
    ASSERT_FALSE(err.matches("test.specific.error.more"));
    ASSERT_FALSE(err.matches("other"));
}

TEST(testXErrors, testNilMatchesBehavior) {
    const auto nil_err = xerrors::NIL;
    const auto specific_err = xerrors::Error("test.specific.error", "");

    ASSERT_FALSE(nil_err.matches(specific_err));
    ASSERT_FALSE(nil_err.matches("test"));
    ASSERT_FALSE(nil_err.matches("test.specific.error"));

    ASSERT_FALSE(specific_err.matches(nil_err));
    ASSERT_FALSE(specific_err.matches(xerrors::TYPE_NIL));

    ASSERT_TRUE(nil_err.matches(xerrors::NIL));
    ASSERT_TRUE(nil_err.matches(xerrors::TYPE_NIL));
}

TEST(testXErrors, testMatchesVectorStrings) {
    const auto err = xerrors::Error("test.specific.error", "");
    const std::vector<std::string> types = {"wrong", "test.specific", "another"};
    ASSERT_TRUE(err.matches(types));

    const std::vector<std::string> no_match_types = {"wrong", "other", "another"};
    ASSERT_FALSE(err.matches(no_match_types));
}

TEST(testXErrors, testSkipSingleError) {
    const auto err = xerrors::Error("test.error", "data");
    const auto skip_err = xerrors::Error("test", "");

    const auto skipped = err.skip(skip_err);
    ASSERT_EQ(skipped.type, xerrors::TYPE_NIL);
    ASSERT_TRUE(skipped.data.empty());

    const auto no_skip_err = xerrors::Error("other", "");
    const auto not_skipped = err.skip(no_skip_err);
    ASSERT_EQ(not_skipped.type, "test.error");
    ASSERT_EQ(not_skipped.data, "data");
}

TEST(testXErrors, testSkipVectorErrors) {
    const auto err = xerrors::Error("test.error", "data");
    const std::vector<xerrors::Error> skip_errors = {
        xerrors::Error("wrong", ""),
        xerrors::Error("test", ""),
        xerrors::Error("another", "")
    };

    const auto skipped = err.skip(skip_errors);
    ASSERT_EQ(skipped.type, xerrors::TYPE_NIL);
    ASSERT_TRUE(skipped.data.empty());

    const std::vector<xerrors::Error> no_skip_errors = {
        xerrors::Error("wrong", ""),
        xerrors::Error("other", ""),
        xerrors::Error("another", "")
    };
    const auto not_skipped = err.skip(no_skip_errors);
    ASSERT_EQ(not_skipped.type, "test.error");
    ASSERT_EQ(not_skipped.data, "data");
}

TEST(testXErrors, testSkipString) {
    const auto err = xerrors::Error("test.error", "data");

    const auto skipped = err.skip("test");
    ASSERT_EQ(skipped.type, xerrors::TYPE_NIL);
    ASSERT_TRUE(skipped.data.empty());

    const auto not_skipped = err.skip("other");
    ASSERT_EQ(not_skipped.type, "test.error");
    ASSERT_EQ(not_skipped.data, "data");
}

TEST(testXErrors, testNotEqualsOperator) {
    const auto err1 = xerrors::Error("test1", "");
    const auto err2 = xerrors::Error("test2", "");
    const auto err3 = xerrors::Error("test1", "");

    ASSERT_TRUE(err1 != err2);
    ASSERT_FALSE(err1 != err3);
}

TEST(testXErrors, testEqualsStringOperator) {
    const auto err = xerrors::Error("test", "data");
    ASSERT_TRUE(err == "test");
    ASSERT_FALSE(err == "other");
}

TEST(testXErrors, testNotEqualsStringOperator) {
    const auto err = xerrors::Error("test", "data");
    ASSERT_FALSE(err != "test");
    ASSERT_TRUE(err != "other");
}

TEST(testXErrors, testPredefinedErrors) {
    ASSERT_EQ(xerrors::NIL.type, xerrors::TYPE_NIL);
    ASSERT_EQ(xerrors::UNKNOWN.type, xerrors::TYPE_UNKNOWN);
    ASSERT_EQ(xerrors::SY.type, "sy");
    ASSERT_EQ(xerrors::VALIDATION.type, "sy.validation");
    ASSERT_EQ(xerrors::QUERY.type, "sy.query");
    ASSERT_EQ(xerrors::MULTIPLE_RESULTS.type, "sy.query.multiple_results");
    ASSERT_EQ(xerrors::NOT_FOUND.type, "sy.query.not_found");
    ASSERT_EQ(xerrors::NOT_SUPPORTED.type, "sy.not_supported");
    ASSERT_EQ(xerrors::INTERNAL.type, "sy.internal");
    ASSERT_EQ(xerrors::UNEXPECTED.type, "sy.unexpected");
    ASSERT_EQ(xerrors::CONTROL.type, "sy.control");
    ASSERT_EQ(xerrors::UNAUTHORIZED.type, "sy.control.unauthorized");
}

TEST(testXErrors, testStringConstructorWithoutDelimiter) {
    const auto err = xerrors::Error("simple.error");
    ASSERT_EQ(err.type, "simple.error");
    ASSERT_TRUE(err.data.empty());
}

TEST(testXErrors, testStringConstructorWithDelimiter) {
    const auto err = xerrors::Error("error.type---error data");
    ASSERT_EQ(err.type, "error.type");
    ASSERT_EQ(err.data, "error data");

    const auto multiple_delimiters = xerrors::Error("error---data---more");
    ASSERT_EQ(multiple_delimiters.type, "error");
    ASSERT_EQ(multiple_delimiters.data, "data---more");
}
