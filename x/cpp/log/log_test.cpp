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

#include "absl/log/globals.h"
#include "absl/log/initialize.h"
#include "absl/log/log.h"
#include "absl/log/log_sink_registry.h"

namespace x::log {
namespace {
constexpr auto YELLOW_CODE = "\033[0;33m";
constexpr auto RED_CODE = "\033[0;31m";
constexpr auto RESET_CODE = "\033[m\n";

/// Silences absl's default stderr output so registered sinks own everything the tests
/// capture. init() is deliberately never called in this process: it can only run once,
/// and these tests need sinks with both color settings.
void silence_default_stderr() {
    static const bool once = [] {
        absl::InitializeLog();
        absl::SetStderrThreshold(absl::LogSeverityAtLeast::kInfinity);
        return true;
    }();
    (void) once;
}

/// @brief registers a sink for the enclosing scope.
struct Registered {
    StderrSink sink;

    explicit Registered(const bool color): sink(color) { absl::AddLogSink(&sink); }
    ~Registered() { absl::RemoveLogSink(&sink); }
};

std::string capture(const bool color, const std::function<void()> &fn) {
    silence_default_stderr();
    const Registered reg(color);
    testing::internal::CaptureStderr();
    fn();
    return testing::internal::GetCapturedStderr();
}
}

/// @brief it should color the entire WARNING line yellow.
TEST(Log, testWarningYellow) {
    const auto out = capture(true, [] { LOG(WARNING) << "warning line"; });
    ASSERT_TRUE(out.starts_with(YELLOW_CODE));
    ASSERT_TRUE(out.ends_with(RESET_CODE));
    ASSERT_NE(out.find("warning line"), std::string::npos);
}

/// @brief it should color the entire ERROR line red.
TEST(Log, testErrorRed) {
    const auto out = capture(true, [] { LOG(ERROR) << "error line"; });
    ASSERT_TRUE(out.starts_with(RED_CODE));
    ASSERT_TRUE(out.ends_with(RESET_CODE));
    ASSERT_NE(out.find("error line"), std::string::npos);
}

/// @brief it should write INFO lines without color codes even when color is on.
TEST(Log, testInfoUncolored) {
    const auto out = capture(true, [] { LOG(INFO) << "plain info line"; });
    ASSERT_NE(out.find("plain info line"), std::string::npos);
    ASSERT_EQ(out.find("\033["), std::string::npos);
}

/// @brief it should write all lines uncolored when color is off.
TEST(Log, testColorDisabled) {
    const auto out = capture(false, [] {
        LOG(WARNING) << "warning line";
        LOG(ERROR) << "error line";
    });
    ASSERT_NE(out.find("warning line"), std::string::npos);
    ASSERT_NE(out.find("error line"), std::string::npos);
    ASSERT_EQ(out.find("\033["), std::string::npos);
}

/// @brief it should write each line exactly once, with the default stderr sink
/// silenced.
TEST(Log, testNoDuplicateLines) {
    const auto out = capture(true, [] { LOG(INFO) << "singular line"; });
    const auto first = out.find("singular line");
    ASSERT_NE(first, std::string::npos);
    ASSERT_EQ(out.find("singular line", first + 1), std::string::npos);
}

/// @brief it should abort on FATAL with the message on stderr.
TEST(Log, testFatalAborts) {
    silence_default_stderr();
    ASSERT_DEATH(
        {
            const Registered reg(true);
            LOG(FATAL) << "fatal line";
        },
        "fatal line"
    );
}

/// @brief it should keep color disabled and helpers empty before init.
TEST(Log, testHelpersBeforeInit) {
    ASSERT_FALSE(color_enabled());
    ASSERT_EQ(get_color("\033[1;31m"), "");
    ASSERT_EQ(RED(), "");
    ASSERT_EQ(RESET(), "");
}

/// @brief it should enable color and helper codes after init(true). Runs in a
/// threadsafe death-test child: init is once-per-process, and the child must re-execute
/// the binary so absl is not already initialized.
TEST(Log, testInitEnablesColor) {
    GTEST_FLAG_SET(death_test_style, "threadsafe");
    ASSERT_EXIT(
        {
            init(true);
            const bool ok = color_enabled() && RED() == "\033[1;31m" &&
                            get_color("x") == "x";
            std::exit(ok ? 0 : 1);
        },
        testing::ExitedWithCode(0),
        ""
    );
}

/// @brief it should leave color disabled after init(false).
TEST(Log, testInitWithoutColor) {
    GTEST_FLAG_SET(death_test_style, "threadsafe");
    ASSERT_EXIT(
        {
            init(false);
            const bool ok = !color_enabled() && RED() == "" && get_color("x") == "";
            std::exit(ok ? 0 : 1);
        },
        testing::ExitedWithCode(0),
        ""
    );
}

/// @brief it should ignore calls to init after the first: no abort from re-initializing
/// absl, no color change, and logging keeps flowing.
TEST(Log, testInitTwiceIsNoOp) {
    GTEST_FLAG_SET(death_test_style, "threadsafe");
    ASSERT_EXIT(
        {
            init(true);
            init(false);
            LOG(WARNING) << "still logging";
            std::exit(color_enabled() ? 0 : 1);
        },
        testing::ExitedWithCode(0),
        "still logging"
    );
}

/// @brief it should route LOG lines through the installed sink after init.
TEST(Log, testInitInstallsSink) {
    GTEST_FLAG_SET(death_test_style, "threadsafe");
    ASSERT_EXIT(
        {
            init(true);
            LOG(WARNING) << "wired through sink";
            std::exit(0);
        },
        testing::ExitedWithCode(0),
        "wired through sink"
    );
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
