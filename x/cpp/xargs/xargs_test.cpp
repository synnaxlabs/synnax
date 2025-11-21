// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
    const auto name = parser.field<std::string>("--name");
    EXPECT_TRUE(parser.errors.empty());
    ASSERT_EQ(name, "test");
    cleanup(argc, argv);
}

TEST_F(XArgsTest, TestRequiredStringMissing) {
    auto [argc, argv] = make_args({"program"});
    parser = xargs::Parser(argc, argv);
    auto name = parser.field<std::string>("--name");
    EXPECT_FALSE(parser.errors.empty());
    ASSERT_EQ(parser.errors.at(0).message(), "[--name] Required argument not found");
    cleanup(argc, argv);
}

TEST_F(XArgsTest, TestRequiredIntegerHappyPath) {
    auto [argc, argv] = make_args({"program", "--count", "42"});
    parser = xargs::Parser(argc, argv);
    const auto count = parser.field<int>("--count");
    EXPECT_TRUE(parser.errors.empty());
    ASSERT_EQ(count, 42);
    cleanup(argc, argv);
}

TEST_F(XArgsTest, TestRequiredIntegerInvalidFormat) {
    auto [argc, argv] = make_args({"program", "--count", "not_a_number"});
    parser = xargs::Parser(argc, argv);
    const auto count = parser.field<int>("--count");
    ASSERT_EQ(count, 0);
    EXPECT_FALSE(parser.errors.empty());
    ASSERT_EQ(parser.errors.at(0).message(), "[--count] Invalid value");
    cleanup(argc, argv);
}

TEST_F(XArgsTest, TestOptionalWithDefault) {
    auto [argc, argv] = make_args({"program"});
    parser = xargs::Parser(argc, argv);
    const auto count = parser.field<int>("--count", 100);
    EXPECT_TRUE(parser.errors.empty());
    ASSERT_EQ(count, 100);
    cleanup(argc, argv);
}

TEST_F(XArgsTest, TestOptionalWithValue) {
    auto [argc, argv] = make_args({"program", "--count", "42"});
    parser = xargs::Parser(argc, argv);
    const auto count = parser.field<int>("--count", 100);
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
    auto [argc, argv] = make_args(
        {"program", "--name", "test", "--count", "42", "--verbose"}
    );
    parser = xargs::Parser(argc, argv);
    const auto name = parser.field<std::string>("--name");
    const auto count = parser.field<int>("--count");
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
    parser.field<std::string>("--name");
    ASSERT_NE(parser.error(), xerrors::NIL);
    cleanup(argc, argv);
}

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

TEST_F(XArgsTest, TestEqualsFormatString) {
    auto [argc, argv] = make_args({"program", "--name=test"});
    parser = xargs::Parser(argc, argv);
    const auto name = parser.field<std::string>("--name");
    EXPECT_TRUE(parser.errors.empty());
    ASSERT_EQ(name, "test");
    cleanup(argc, argv);
}

TEST_F(XArgsTest, TestEqualsFormatInteger) {
    auto [argc, argv] = make_args({"program", "--count=42"});
    parser = xargs::Parser(argc, argv);
    const auto count = parser.field<int>("--count");
    EXPECT_TRUE(parser.errors.empty());
    ASSERT_EQ(count, 42);
    cleanup(argc, argv);
}

TEST_F(XArgsTest, TestEqualsFormatOptional) {
    auto [argc, argv] = make_args({"program", "--value=123"});
    parser = xargs::Parser(argc, argv);
    const auto value = parser.field<int>("--value", 100);
    EXPECT_TRUE(parser.errors.empty());
    ASSERT_EQ(value, 123);
    cleanup(argc, argv);
}

TEST_F(XArgsTest, TestEqualsFormatInvalid) {
    auto [argc, argv] = make_args({"program", "--count=not_a_number"});
    parser = xargs::Parser(argc, argv);
    const auto count = parser.field<int>("--count");
    ASSERT_EQ(count, 0);
    EXPECT_FALSE(parser.errors.empty());
    ASSERT_EQ(parser.errors.at(0).message(), "[--count] Invalid value");
    cleanup(argc, argv);
}

TEST_F(XArgsTest, TestMixedFormatArguments) {
    auto [argc, argv] = make_args(
        {"program", "--name=test", "--count", "42", "--verbose", "--debug=true"}
    );
    parser = xargs::Parser(argc, argv);
    const auto name = parser.field<std::string>("--name");
    const auto count = parser.field<int>("--count");
    const auto verbose = parser.flag("--verbose");
    const auto debug = parser.flag("--debug");

    EXPECT_TRUE(parser.errors.empty());
    ASSERT_EQ(name, "test");
    ASSERT_EQ(count, 42);
    ASSERT_TRUE(verbose);
    ASSERT_TRUE(debug);
    cleanup(argc, argv);
}

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

    EXPECT_TRUE(parser.errors.empty());
    cleanup(argc, argv);
}

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

    EXPECT_TRUE(parser.errors.empty());
    cleanup(argc, argv);
}

TEST_F(XArgsTest, TestNullPointerHandling) {
    // Test with nullptr argv
    parser = xargs::Parser(0, nullptr);
    EXPECT_TRUE(parser.argv_.empty());
    EXPECT_TRUE(parser.errors.empty());

    // Verify behavior when trying to access values
    const auto required_str = parser.field<std::string>("test");
    EXPECT_TRUE(required_str.empty());
    EXPECT_FALSE(parser.errors.empty());
    EXPECT_EQ(parser.errors[0].message(), "[test] Required argument not found");

    // Clear errors for next test
    parser.errors.clear();

    // Test optional values
    const auto optional_str = parser.field<std::string>("test", "default");
    EXPECT_EQ(optional_str, "default");
    EXPECT_TRUE(parser.errors.empty());

    // Test flags
    EXPECT_FALSE(parser.flag("test"));
    EXPECT_TRUE(parser.errors.empty());

    // Test with zero argc but non-null argv
    char *dummy_argv[] = {nullptr};
    parser = xargs::Parser(0, dummy_argv);
    EXPECT_TRUE(parser.argv_.empty());
    EXPECT_TRUE(parser.errors.empty());
}

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

    EXPECT_TRUE(parser.errors.empty());
    ASSERT_EQ(name, "second");
    ASSERT_EQ(count, 20);
    ASSERT_TRUE(verbose); // Last --verbose flag wins
    cleanup(argc, argv);
}

TEST_F(XArgsTest, TestRegressionDash) {
    auto [argc, argv] = make_args({
        "program",
        "--correct-skew=true",
    });
    parser = xargs::Parser(argc, argv);

    const auto correct_skew = parser.field<bool>("correct_skew");
    EXPECT_TRUE(parser.errors.empty());
    ASSERT_TRUE(correct_skew);
    cleanup(argc, argv);
}

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
    EXPECT_TRUE(parser.errors.empty());
    ASSERT_EQ(strings.size(), 3);
    ASSERT_EQ(strings[0], "dog");
    ASSERT_EQ(strings[1], "cat");
    ASSERT_EQ(strings[2], "ferret");

    // Test integer vector
    auto numbers = parser.field<std::vector<int>>("numbers");
    EXPECT_TRUE(parser.errors.empty());
    ASSERT_EQ(numbers.size(), 5);
    ASSERT_EQ(numbers[0], 1);
    ASSERT_EQ(numbers[4], 5);

    // Test double vector
    auto doubles = parser.field<std::vector<double>>("doubles");
    EXPECT_TRUE(parser.errors.empty());
    ASSERT_EQ(doubles.size(), 3);
    ASSERT_DOUBLE_EQ(doubles[0], 1.5);
    ASSERT_DOUBLE_EQ(doubles[1], 2.7);
    ASSERT_DOUBLE_EQ(doubles[2], 3.14);

    // Test optional vector with default
    std::vector<int> default_vec = {1, 2, 3};
    auto optional_nums = parser.field<std::vector<int>>("missing", default_vec);
    EXPECT_TRUE(parser.errors.empty());
    ASSERT_EQ(optional_nums, default_vec);

    cleanup(argc, argv);
}
