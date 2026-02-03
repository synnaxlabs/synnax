// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/xargs/xargs.h"
#include "x/cpp/xtest/xtest.h"

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

    xargs::Parser parser{0, nullptr};
};

/// @brief it should correctly parse a required string argument.
TEST_F(XArgsTest, TestRequiredStringHappyPath) {
    auto [argc, argv] = make_args({"program", "--name", "test"});
    parser = xargs::Parser(argc, argv);
    const auto name = parser.field<std::string>("--name");
    ASSERT_NIL(parser.error());
    ASSERT_EQ(name, "test");
    cleanup(argc, argv);
}

/// @brief it should return an error when a required string argument is missing.
TEST_F(XArgsTest, TestRequiredStringMissing) {
    auto [argc, argv] = make_args({"program"});
    parser = xargs::Parser(argc, argv);
    auto name = parser.field<std::string>("--name");
    EXPECT_FALSE(parser.errors.empty());
    ASSERT_EQ(parser.errors.at(0).data, "--name: required argument not found");
    cleanup(argc, argv);
}

/// @brief it should correctly parse a required integer argument.
TEST_F(XArgsTest, TestRequiredIntegerHappyPath) {
    auto [argc, argv] = make_args({"program", "--count", "42"});
    parser = xargs::Parser(argc, argv);
    const auto count = parser.field<int>("--count");
    ASSERT_NIL(parser.error());
    ASSERT_EQ(count, 42);
    cleanup(argc, argv);
}

/// @brief it should return an error when an integer argument has an invalid format.
TEST_F(XArgsTest, TestRequiredIntegerInvalidFormat) {
    auto [argc, argv] = make_args({"program", "--count", "not_a_number"});
    parser = xargs::Parser(argc, argv);
    const auto count = parser.field<int>("--count");
    ASSERT_EQ(count, 0);
    EXPECT_FALSE(parser.errors.empty());
    ASSERT_EQ(parser.errors.at(0).message(), "[--count] Invalid value");
    cleanup(argc, argv);
}

/// @brief it should return the default value when an optional argument is missing.
TEST_F(XArgsTest, TestOptionalWithDefault) {
    auto [argc, argv] = make_args({"program"});
    parser = xargs::Parser(argc, argv);
    const auto count = parser.field<int>("--count", 100);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(count, 100);
    cleanup(argc, argv);
}

/// @brief it should use the provided value over the default for optional arguments.
TEST_F(XArgsTest, TestOptionalWithValue) {
    auto [argc, argv] = make_args({"program", "--count", "42"});
    parser = xargs::Parser(argc, argv);
    const auto count = parser.field<int>("--count", 100);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(count, 42);
    cleanup(argc, argv);
}

/// @brief it should correctly parse boolean flags.
TEST_F(XArgsTest, TestFlag) {
    auto [argc, argv] = make_args({"program", "--verbose"});
    parser = xargs::Parser(argc, argv);
    ASSERT_TRUE(parser.flag("--verbose"));
    ASSERT_FALSE(parser.flag("--quiet"));
    cleanup(argc, argv);
}

/// @brief it should correctly parse multiple arguments of different types.
TEST_F(XArgsTest, TestMultipleArguments) {
    auto [argc, argv] = make_args(
        {"program", "--name", "test", "--count", "42", "--verbose"}
    );
    parser = xargs::Parser(argc, argv);
    const auto name = parser.field<std::string>("--name");
    const auto count = parser.field<int>("--count");
    const auto verbose = parser.flag("--verbose");
    ASSERT_NIL(parser.error());
    ASSERT_EQ(name, "test");
    ASSERT_EQ(count, 42);
    ASSERT_TRUE(verbose);
    cleanup(argc, argv);
}

/// @brief it should return a validation error when a required argument is missing.
TEST_F(XArgsTest, TestError) {
    auto [argc, argv] = make_args({"program"});
    parser = xargs::Parser(argc, argv);
    ASSERT_NIL(parser.error());
    parser.field<std::string>("--name");
    ASSERT_OCCURRED_AS(parser.error(), xerrors::VALIDATION);
    cleanup(argc, argv);
}

/// @brief it should correctly parse file path arguments with dashes.
TEST(XArgs, Regression) {
    auto parser = xargs::Parser(
        std::vector<std::string>{
            "program",
            "--state-file",
            "/tmp/rack-config-test/state.json"
        }
    );
    const std::string value = parser.field("--state-file", "");
    ASSERT_EQ(value, "/tmp/rack-config-test/state.json");
}

/// @brief it should correctly parse string arguments using equals format.
TEST_F(XArgsTest, TestEqualsFormatString) {
    auto [argc, argv] = make_args({"program", "--name=test"});
    parser = xargs::Parser(argc, argv);
    const auto name = parser.field<std::string>("--name");
    ASSERT_NIL(parser.error());
    ASSERT_EQ(name, "test");
    cleanup(argc, argv);
}

/// @brief it should correctly parse integer arguments using equals format.
TEST_F(XArgsTest, TestEqualsFormatInteger) {
    auto [argc, argv] = make_args({"program", "--count=42"});
    parser = xargs::Parser(argc, argv);
    const auto count = parser.field<int>("--count");
    ASSERT_NIL(parser.error());
    ASSERT_EQ(count, 42);
    cleanup(argc, argv);
}

/// @brief it should correctly parse optional arguments using equals format.
TEST_F(XArgsTest, TestEqualsFormatOptional) {
    auto [argc, argv] = make_args({"program", "--value=123"});
    parser = xargs::Parser(argc, argv);
    const auto value = parser.field<int>("--value", 100);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(value, 123);
    cleanup(argc, argv);
}

/// @brief it should return an error for invalid integer values in equals format.
TEST_F(XArgsTest, TestEqualsFormatInvalid) {
    auto [argc, argv] = make_args({"program", "--count=not_a_number"});
    parser = xargs::Parser(argc, argv);
    const auto count = parser.field<int>("--count");
    ASSERT_EQ(count, 0);
    EXPECT_FALSE(parser.errors.empty());
    ASSERT_EQ(parser.errors.at(0).message(), "[--count] Invalid value");
    cleanup(argc, argv);
}

/// @brief it should correctly parse a mix of space and equals format arguments.
TEST_F(XArgsTest, TestMixedFormatArguments) {
    auto [argc, argv] = make_args(
        {"program", "--name=test", "--count", "42", "--verbose", "--debug=true"}
    );
    parser = xargs::Parser(argc, argv);
    const auto name = parser.field<std::string>("--name");
    const auto count = parser.field<int>("--count");
    const auto verbose = parser.flag("--verbose");
    const auto debug = parser.flag("--debug");

    ASSERT_NIL(parser.error());
    ASSERT_EQ(name, "test");
    ASSERT_EQ(count, 42);
    ASSERT_TRUE(verbose);
    ASSERT_TRUE(debug);
    cleanup(argc, argv);
}

/// @brief it should correctly handle different prefix styles for arguments.
TEST_F(XArgsTest, TestPrefixHandling) {
    auto [argc, argv] = make_args(
        {"program", "--long-flag", "-v", "--unprefixed=value", "-f", "file.txt"}
    );
    parser = xargs::Parser(argc, argv);

    // Test different prefix scenarios
    ASSERT_TRUE(parser.flag("--long-flag")); // Original --
    ASSERT_TRUE(parser.flag("long-flag")); // Auto-add --
    ASSERT_TRUE(parser.flag("-v")); // Preserve single -
    ASSERT_TRUE(parser.flag("v")); // Auto-add --
    ASSERT_EQ(parser.field<std::string>("-f"), "file.txt"); // Preserve single -
    ASSERT_EQ(parser.field<std::string>("unprefixed"), "value"); // Auto-add --

    ASSERT_NIL(parser.error());
    cleanup(argc, argv);
}

/// @brief it should correctly parse single letter flags with various prefixes.
TEST_F(XArgsTest, TestSingleLetterFlags) {
    auto [argc, argv] = make_args({"program", "-v", "--f", "-x=true"});
    parser = xargs::Parser(argc, argv);

    // Test single letter flags with different prefixes
    ASSERT_TRUE(parser.flag("v")); // Should match -v
    ASSERT_TRUE(parser.flag("-v")); // Should match -v
    ASSERT_TRUE(parser.flag("--v")); // Should match -v
    ASSERT_TRUE(parser.flag("f")); // Should match --f
    ASSERT_TRUE(parser.flag("-f")); // Should match --f
    ASSERT_TRUE(parser.flag("--f")); // Should match --f
    ASSERT_TRUE(parser.flag("x")); // Should match -x=true

    ASSERT_NIL(parser.error());
    cleanup(argc, argv);
}

/// @brief it should handle null pointer arguments gracefully.
TEST_F(XArgsTest, TestNullPointerHandling) {
    // Test with nullptr argv
    parser = xargs::Parser(0, nullptr);
    EXPECT_TRUE(parser.argv_.empty());
    ASSERT_NIL(parser.error());

    // Verify behavior when trying to access values
    const auto required_str = parser.field<std::string>("test");
    EXPECT_TRUE(required_str.empty());
    EXPECT_FALSE(parser.errors.empty());
    EXPECT_EQ(
        parser.errors[0].message(),
        "[sy.validation] test: required argument not found"
    );

    // Clear errors for next test
    parser.errors.clear();

    // Test optional values
    const auto optional_str = parser.field<std::string>("test", "default");
    EXPECT_EQ(optional_str, "default");
    ASSERT_NIL(parser.error());

    // Test flags
    EXPECT_FALSE(parser.flag("test"));
    ASSERT_NIL(parser.error());

    // Test with zero argc but non-null argv
    char *dummy_argv[] = {nullptr};
    parser = xargs::Parser(0, dummy_argv);
    EXPECT_TRUE(parser.argv_.empty());
    ASSERT_NIL(parser.error());
}

/// @brief it should handle unusual argument names and edge cases.
TEST_F(XArgsTest, TestWeirdArgumentNames) {
    auto [argc, argv] = make_args(
        {"program",
         "---triple-dash",
         "----quad-dash=value",
         "--weird@#$%chars",
         "--spaces in value",
         "--unicode-☺=smiley",
         "--empty=",
         "---=direct",
         "--chain=one--chain=two",
         "--duplicate=first",
         "--duplicate=second",
         "=standalone-equals",
         "--=empty-name",
         "--no-value=",
         "--==double-equals",
         "--missing-equals-dash--next-arg",
         "--space = value",
         "--=-",
         "----",
         "- -",
         "--"}
    );
    parser = xargs::Parser(argc, argv);

    ASSERT_FALSE(parser.flag("triple-dash"));
    ASSERT_TRUE(parser.flag("---triple-dash"));
    ASSERT_EQ(parser.field<std::string>("quad-dash"), "");
    ASSERT_EQ(parser.field<std::string>("----quad-dash"), "value");
    ASSERT_TRUE(parser.flag("weird@#$%chars"));
    ASSERT_EQ(parser.field<std::string>("spaces"), "");
    ASSERT_EQ(parser.field<std::string>("unicode-☺"), "smiley");
    ASSERT_EQ(parser.field<std::string>("empty"), "");
    ASSERT_EQ(parser.field<std::string>("---"), "direct");
    ASSERT_EQ(parser.field<std::string>("chain"), "one--chain=two");
    ASSERT_FALSE(parser.flag("=standalone-equals"));
    ASSERT_EQ(parser.field<std::string>(""), "standalone-equals");
    ASSERT_EQ(parser.field<std::string>("=double-equals"), "");
    ASSERT_TRUE(parser.flag("missing-equals-dash--next-arg"));
    ASSERT_EQ(parser.field<std::string>("space"), "");
    ASSERT_EQ(parser.field<std::string>("-"), "-");
    ASSERT_TRUE(parser.flag("----"));
    ASSERT_TRUE(parser.flag("- -"));
    ASSERT_TRUE(parser.flag("--"));

    EXPECT_FALSE(parser.errors.empty());
    cleanup(argc, argv);
}

/// @brief it should use the last value when duplicate arguments are provided.
TEST_F(XArgsTest, TestDuplicateArguments) {
    auto [argc, argv] = make_args(
        {"program",
         "--name",
         "first",
         "--name=second",
         "--count",
         "10",
         "--count",
         "20",
         "--verbose",
         "--verbose=false",
         "--verbose"}
    );
    parser = xargs::Parser(argc, argv);

    const auto name = parser.field<std::string>("name");
    const auto count = parser.field<int>("count");
    const auto verbose = parser.flag("verbose");

    ASSERT_NIL(parser.error());
    ASSERT_EQ(name, "second");
    ASSERT_EQ(count, 20);
    ASSERT_TRUE(verbose); // Last --verbose flag wins
    cleanup(argc, argv);
}

/// @brief it should normalize dashes and underscores in argument names.
TEST_F(XArgsTest, TestRegressionDash) {
    auto [argc, argv] = make_args({
        "program",
        "--correct-skew=true",
    });
    parser = xargs::Parser(argc, argv);

    const auto correct_skew = parser.field<bool>("correct_skew");
    ASSERT_NIL(parser.error());
    ASSERT_TRUE(correct_skew);
    cleanup(argc, argv);
}

/// @brief it should correctly parse comma-separated vector arguments.
TEST_F(XArgsTest, TestVectorArguments) {
    auto [argc, argv] = make_args(
        {"program",
         "--strings=dog,cat,ferret",
         "--numbers=1,2,3,4,5",
         "--doubles=1.5,2.7,3.14"}
    );
    parser = xargs::Parser(argc, argv);

    // Test string vector
    auto strings = parser.field<std::vector<std::string>>("strings");
    ASSERT_NIL(parser.error());
    ASSERT_EQ(strings.size(), 3);
    ASSERT_EQ(strings[0], "dog");
    ASSERT_EQ(strings[1], "cat");
    ASSERT_EQ(strings[2], "ferret");

    // Test integer vector
    auto numbers = parser.field<std::vector<int>>("numbers");
    ASSERT_NIL(parser.error());
    ASSERT_EQ(numbers.size(), 5);
    ASSERT_EQ(numbers[0], 1);
    ASSERT_EQ(numbers[4], 5);

    // Test double vector
    auto doubles = parser.field<std::vector<double>>("doubles");
    ASSERT_NIL(parser.error());
    ASSERT_EQ(doubles.size(), 3);
    ASSERT_DOUBLE_EQ(doubles[0], 1.5);
    ASSERT_DOUBLE_EQ(doubles[1], 2.7);
    ASSERT_DOUBLE_EQ(doubles[2], 3.14);

    // Test optional vector with default
    std::vector<int> default_vec = {1, 2, 3};
    auto optional_nums = parser.field<std::vector<int>>("missing", default_vec);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(optional_nums, default_vec);

    cleanup(argc, argv);
}
