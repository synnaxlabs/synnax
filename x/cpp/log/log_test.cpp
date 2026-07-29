// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <functional>
#include <string>

#include "gtest/gtest.h"

#include "x/cpp/log/log.h"

#include "absl/log/log.h"

namespace x::log {
namespace {
constexpr auto YELLOW_CODE = "\033[0;33m";
constexpr auto RED_CODE = "\033[0;31m";
constexpr auto RESET_CODE = "\033[m\n";

/// init installs a process-global sink and absl::InitializeLog can only run
/// once, so every test shares a single colored initialization.
void ensure_init() {
    static const bool once = [] {
        init(true);
        return true;
    }();
    (void) once;
}

std::string capture(const std::function<void()> &fn) {
    ensure_init();
    testing::internal::CaptureStderr();
    fn();
    return testing::internal::GetCapturedStderr();
}
}

/// @brief it should report color as enabled after init(true).
TEST(Log, testColorEnabled) {
    ensure_init();
    ASSERT_TRUE(color_enabled());
}

/// @brief it should pass color codes through when color is enabled.
TEST(Log, testGetColor) {
    ensure_init();
    ASSERT_EQ(get_color("\033[1;31m"), "\033[1;31m");
    ASSERT_EQ(RED(), "\033[1;31m");
    ASSERT_EQ(GREEN(), "\033[1;32m");
    ASSERT_EQ(RESET(), "\033[0m");
}

/// @brief it should write INFO lines without any color codes.
TEST(Log, testInfoUncolored) {
    const auto out = capture([] { LOG(INFO) << "plain info line"; });
    ASSERT_NE(out.find("plain info line"), std::string::npos);
    ASSERT_EQ(out.find("\033["), std::string::npos);
}

/// @brief it should color the entire WARNING line yellow.
TEST(Log, testWarningYellow) {
    const auto out = capture([] { LOG(WARNING) << "warning line"; });
    ASSERT_TRUE(out.starts_with(YELLOW_CODE));
    ASSERT_TRUE(out.ends_with(RESET_CODE));
    ASSERT_NE(out.find("warning line"), std::string::npos);
}

/// @brief it should color the entire ERROR line red.
TEST(Log, testErrorRed) {
    const auto out = capture([] { LOG(ERROR) << "error line"; });
    ASSERT_TRUE(out.starts_with(RED_CODE));
    ASSERT_TRUE(out.ends_with(RESET_CODE));
    ASSERT_NE(out.find("error line"), std::string::npos);
}

/// @brief it should write each line exactly once, with the default stderr sink
/// silenced.
TEST(Log, testNoDuplicateLines) {
    const auto out = capture([] { LOG(INFO) << "singular line"; });
    const auto first = out.find("singular line");
    ASSERT_NE(first, std::string::npos);
    ASSERT_EQ(out.find("singular line", first + 1), std::string::npos);
}

/// @brief it should abort on FATAL with the message on stderr.
TEST(Log, testFatalAborts) {
    ensure_init();
    ASSERT_DEATH(LOG(FATAL) << "fatal line", "fatal line");
}

/// @brief it should convert bools to their string names.
TEST(Log, testBoolToStr) {
    ASSERT_EQ(bool_to_str(true), "true");
    ASSERT_EQ(bool_to_str(false), "false");
}

/// @brief it should mask every character of a sensitive string.
TEST(Log, testSensitiveString) {
    ASSERT_EQ(sensitive_string("secret"), "******");
    ASSERT_EQ(sensitive_string(""), "");
}
}
