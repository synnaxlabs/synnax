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

/// local
#include "x/cpp/xargs/xargs.h"

class XArgsTest : public ::testing::Test {
protected:
    static std::pair<int, char **> make_args(const std::vector<std::string> &args) {
        auto argv = new char *[args.size()];
        for (size_t i = 0; i < args.size(); i++) {
            argv[i] = new char[args[i].size() + 1];
            strcpy(argv[i], args[i].c_str());
        }
        return {static_cast<int>(args.size()), argv};
    }

    static void cleanup(const int argc, char **argv) {
        for (int i = 0; i < argc; i++)
            delete[] argv[i];
        delete[] argv;
    }

    xargs::Parser parser{0, nullptr}; // Default initialized, will be replaced in tests
};

TEST_F(XArgsTest, TestRequiredStringHappyPath) {
    auto [argc, argv] = make_args({"program", "--name", "test"});
    parser = xargs::Parser(argc, argv);
    const auto name = parser.required<std::string>("--name");
    EXPECT_TRUE(parser.errors.empty());
    ASSERT_EQ(name, "test");
    cleanup(argc, argv);
}

TEST_F(XArgsTest, TestRequiredStringMissing) {
    auto [argc, argv] = make_args({"program"});
    parser = xargs::Parser(argc, argv);
    auto name = parser.required<std::string>("--name");
    EXPECT_FALSE(parser.errors.empty());
    ASSERT_EQ(parser.errors.at(0).message(), "[--name] Required argument not found");
    cleanup(argc, argv);
}

TEST_F(XArgsTest, TestRequiredIntegerHappyPath) {
    auto [argc, argv] = make_args({"program", "--count", "42"});
    parser = xargs::Parser(argc, argv);
    const auto count = parser.required<int>("--count");
    EXPECT_TRUE(parser.errors.empty());
    ASSERT_EQ(count, 42);
    cleanup(argc, argv);
}

TEST_F(XArgsTest, TestRequiredIntegerInvalidFormat) {
    auto [argc, argv] = make_args({"program", "--count", "not_a_number"});
    parser = xargs::Parser(argc, argv);
    const auto count = parser.required<int>("--count");
    ASSERT_EQ(count, 0);
    EXPECT_FALSE(parser.errors.empty());
    ASSERT_EQ(parser.errors.at(0).message(), "[--count] Invalid value");
    cleanup(argc, argv);
}

TEST_F(XArgsTest, TestOptionalWithDefault) {
    auto [argc, argv] = make_args({"program"});
    parser = xargs::Parser(argc, argv);
    const auto count = parser.optional<int>("--count", 100);
    EXPECT_TRUE(parser.errors.empty());
    ASSERT_EQ(count, 100);
    cleanup(argc, argv);
}

TEST_F(XArgsTest, TestOptionalWithValue) {
    auto [argc, argv] = make_args({"program", "--count", "42"});
    parser = xargs::Parser(argc, argv);
    const auto count = parser.optional<int>("--count", 100);
    EXPECT_TRUE(parser.errors.empty());
    ASSERT_EQ(count, 42);
    cleanup(argc, argv);
}

TEST_F(XArgsTest, TestFlag) {
    auto [argc, argv] = make_args({"program", "--verbose"});
    parser = xargs::Parser(argc, argv);
    ASSERT_TRUE(parser.flag("--verbose"));
    ASSERT_FALSE(parser.flag("--quiet"));
    cleanup(argc, argv);
}

TEST_F(XArgsTest, TestMultipleArguments) {
    auto [argc, argv] = make_args({
        "program",
        "--name", "test",
        "--count", "42",
        "--verbose"
    });
    parser = xargs::Parser(argc, argv);
    const auto name = parser.required<std::string>("--name");
    const auto count = parser.required<int>("--count");
    const auto verbose = parser.flag("--verbose");
    EXPECT_TRUE(parser.errors.empty());
    ASSERT_EQ(name, "test");
    ASSERT_EQ(count, 42);
    ASSERT_TRUE(verbose);
    cleanup(argc, argv);
}

TEST_F(XArgsTest, TestError) {
    auto [argc, argv] = make_args({"program"});
    parser = xargs::Parser(argc, argv);
    ASSERT_EQ(parser.error(), xerrors::NIL);
    parser.required<std::string>("--name");
    ASSERT_NE(parser.error(), xerrors::NIL);
    cleanup(argc, argv);
}

TEST(XArgs, Regression) {
    auto parser = xargs::Parser(std::vector<std::string>{"program", "--state-file", "/tmp/rack-config-test/state.json"});
    const auto value = parser.optional<std::string>("--state-file", "");
    ASSERT_EQ(value, "/tmp/rack-config-test/state.json");
}