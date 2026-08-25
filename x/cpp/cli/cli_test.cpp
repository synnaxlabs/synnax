// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>
#include <sstream>
#include <string>

#include "gtest/gtest.h"

#include "x/cpp/cli/cli.h"

namespace x::cli {
/// @brief it should return the entered value.
TEST(CLITest, PromptReturnsInput) {
    std::istringstream in("synnax.example.com\n");
    std::ostringstream out;
    EXPECT_EQ(prompt("Host", std::nullopt, false, in, out), "synnax.example.com");
    EXPECT_NE(out.str().find("Host: "), std::string::npos);
}

/// @brief it should return the default when the input is empty.
TEST(CLITest, PromptEmptyInputReturnsDefault) {
    std::istringstream in("\n");
    std::ostringstream out;
    EXPECT_EQ(prompt("Host", "localhost", false, in, out), "localhost");
}

/// @brief it should prefer the entered value over the default.
TEST(CLITest, PromptInputOverridesDefault) {
    std::istringstream in("remote\n");
    std::ostringstream out;
    EXPECT_EQ(prompt("Host", "localhost", false, in, out), "remote");
}

/// @brief it should display the default in the prompt.
TEST(CLITest, PromptDisplaysDefault) {
    std::istringstream in("\n");
    std::ostringstream out;
    prompt("Host", "localhost", false, in, out);
    EXPECT_NE(out.str().find("Host [localhost]: "), std::string::npos);
}

/// @brief it should display an empty default as [none].
TEST(CLITest, PromptDisplaysEmptyDefaultAsNone) {
    std::istringstream in("\n");
    std::ostringstream out;
    EXPECT_EQ(prompt("Path to CA certificate file", "", false, in, out), "");
    EXPECT_NE(
        out.str().find("Path to CA certificate file [none]: "),
        std::string::npos
    );
}

/// @brief it should re-prompt on empty input when there is no default.
TEST(CLITest, PromptRetriesUntilNonEmpty) {
    std::istringstream in("\n\nnico\n");
    std::ostringstream out;
    EXPECT_EQ(prompt("Username", std::nullopt, false, in, out), "nico");
}

/// @brief it should return the default when the stream ends before a value is entered.
TEST(CLITest, PromptExhaustedStreamReturnsDefault) {
    std::istringstream in("");
    std::ostringstream out;
    EXPECT_EQ(prompt("Host", "localhost", false, in, out), "localhost");
}

/// @brief it should return empty when the stream ends and there is no default.
TEST(CLITest, PromptExhaustedStreamWithoutDefaultReturnsEmpty) {
    std::istringstream in("");
    std::ostringstream out;
    EXPECT_EQ(prompt("Username", std::nullopt, false, in, out), "");
}

/// @brief it should confirm on Y and deny on N, ignoring case.
TEST(CLITest, ConfirmYesNo) {
    for (const auto &yes: {"Y\n", "y\n"}) {
        std::istringstream in(yes);
        std::ostringstream out;
        EXPECT_TRUE(confirm("Secure", std::nullopt, in, out));
    }
    for (const auto &no: {"N\n", "n\n"}) {
        std::istringstream in(no);
        std::ostringstream out;
        EXPECT_FALSE(confirm("Secure", std::nullopt, in, out));
    }
}

/// @brief it should return the default when the input is empty.
TEST(CLITest, ConfirmEmptyInputReturnsDefault) {
    std::istringstream yes_in("\n");
    std::ostringstream yes_out;
    EXPECT_TRUE(confirm("Secure", true, yes_in, yes_out));
    EXPECT_NE(yes_out.str().find("Secure (Y/N) [Y]: "), std::string::npos);

    std::istringstream no_in("\n");
    std::ostringstream no_out;
    EXPECT_FALSE(confirm("Secure", false, no_in, no_out));
    EXPECT_NE(no_out.str().find("Secure (Y/N) [N]: "), std::string::npos);
}

/// @brief it should re-prompt until the response is a single Y or N.
TEST(CLITest, ConfirmRetriesOnInvalidInput) {
    std::istringstream in("x\nyes\nN\n");
    std::ostringstream out;
    EXPECT_FALSE(confirm("Secure", std::nullopt, in, out));
    EXPECT_NE(out.str().find("Please enter Y or N"), std::string::npos);
}

/// @brief it should fall back to the default when the stream ends.
TEST(CLITest, ConfirmExhaustedStreamReturnsDefault) {
    std::istringstream in("");
    std::ostringstream out;
    EXPECT_TRUE(confirm("Secure", true, in, out));
}

/// @brief it should deny when the stream ends and there is no default.
TEST(CLITest, ConfirmExhaustedStreamWithoutDefaultDenies) {
    std::istringstream in("");
    std::ostringstream out;
    EXPECT_FALSE(confirm("Secure", std::nullopt, in, out));
}

/// @brief it should parse the entered number.
TEST(CLITest, PromptNumericParsesInput) {
    std::istringstream in("8080\n");
    std::ostringstream out;
    EXPECT_EQ(prompt<std::uint16_t>("Port", std::nullopt, in, out), 8080);
}

/// @brief it should return the numeric default when the input is empty.
TEST(CLITest, PromptNumericEmptyInputReturnsDefault) {
    std::istringstream in("\n");
    std::ostringstream out;
    const auto port = prompt<std::uint16_t>(
        "Port",
        std::optional<std::uint16_t>(9090),
        in,
        out
    );
    EXPECT_EQ(port, 9090);
    EXPECT_NE(out.str().find("Port [9090]: "), std::string::npos);
}

/// @brief it should re-prompt until the input parses as a number.
TEST(CLITest, PromptNumericRetriesOnInvalidInput) {
    std::istringstream in("abc\n42\n");
    std::ostringstream out;
    EXPECT_EQ(prompt<int>("Count", std::nullopt, in, out), 42);
    EXPECT_NE(out.str().find("Invalid input"), std::string::npos);
}

/// @brief it should parse floating point types.
TEST(CLITest, PromptNumericFloatingPoint) {
    std::istringstream double_in("1.5\n");
    std::ostringstream double_out;
    EXPECT_DOUBLE_EQ(prompt<double>("Scale", std::nullopt, double_in, double_out), 1.5);

    std::istringstream float_in("2.5\n");
    std::ostringstream float_out;
    EXPECT_FLOAT_EQ(prompt<float>("Scale", std::nullopt, float_in, float_out), 2.5f);
}

/// @brief it should return the default when the stream ends before a value is entered.
TEST(CLITest, PromptNumericExhaustedStreamReturnsDefault) {
    std::istringstream in("");
    std::ostringstream out;
    const auto port = prompt<std::uint16_t>(
        "Port",
        std::optional<std::uint16_t>(9090),
        in,
        out
    );
    EXPECT_EQ(port, 9090);
}

/// @brief it should return zero when the stream ends and there is no default.
TEST(CLITest, PromptNumericExhaustedStreamWithoutDefaultReturnsZero) {
    std::istringstream in("");
    std::ostringstream out;
    EXPECT_EQ(prompt<int>("Count", std::nullopt, in, out), 0);
}
}
