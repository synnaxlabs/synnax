// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <csignal>
#include <cstdlib>
#include <exception>
#include <stdexcept>
#include <thread>
#include <vector>

#ifdef _WIN32
#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#include <windows.h>
#endif

#include "gmock/gmock.h"
#include "gtest/gtest.h"

#include "x/cpp/crash/crash.h"

using ::testing::AllOf;
using ::testing::HasSubstr;
using ::testing::Not;

#if defined(_MSC_VER)
#define CRASH_TEST_NOINLINE __declspec(noinline)
#else
#define CRASH_TEST_NOINLINE __attribute__((noinline, visibility("default")))
#endif

/// @brief throws unconditionally. Kept separate from the noexcept function below
/// because a throw-expression placed directly inside a noexcept function is rejected by
/// -Werror=exceptions, whereas a call to a throwing function is not.
CRASH_TEST_NOINLINE void crash_test_thrower() {
    throw std::runtime_error("boom-message");
}

/// @brief calls a throwing function across a noexcept boundary, forcing std::terminate
/// immediately. This is what lets the death-test child reach our terminate handler: a
/// plain throw in the death-test statement is caught by gtest's own harness before
/// terminate runs. The named, external-linkage frame also gives the trace a symbol to
/// resolve.
CRASH_TEST_NOINLINE void crash_test_unhandled_throw() noexcept {
    crash_test_thrower();
}

/// @brief it should dump a banner, the exception message, and a stack trace when a C++
/// exception goes unhandled.
TEST(CrashDeathTest, UnhandledExceptionDumpsTrace) {
    EXPECT_DEATH(
        {
            x::crash::install("test-driver");
            crash_test_unhandled_throw();
        },
        AllOf(
            HasSubstr("test-driver terminated: unhandled exception"),
            HasSubstr("boom-message"),
            HasSubstr("0x")
        )
    );
}

/// @brief it should capture a symbolized stack trace of the calling thread on demand,
/// without a crash. Exercises the trace engine (DbgHelp on Windows, backtrace on POSIX)
/// deterministically on every platform; CI's crash death tests cannot, since they only
/// assert an address is present.
TEST(CrashTest, CaptureTraceProducesFrames) {
    const std::string trace = x::crash::capture_trace();
    EXPECT_THAT(trace, HasSubstr("0x"));
    EXPECT_GE(std::count(trace.begin(), trace.end(), '\n'), 2);
}

#ifndef _WIN32
using ::testing::KilledBySignal;

/// @brief it should resolve and demangle symbol names in the captured trace where
/// symbols are available. capture_trace is reached through testing::Test::Run, whose
/// mangled name only reads as "testing::Test::Run" once demangled. (Windows needs a
/// .pdb, absent in fastbuild, so POSIX-only.)
TEST(CrashTest, CaptureTraceSymbolizesFrames) {
    EXPECT_THAT(x::crash::capture_trace(), HasSubstr("testing::Test::Run"));
}

/// @brief it should produce a demangled, human-readable trace rather than raw mangled
/// symbols. The gtest frame that invokes the test body is always on the death-test
/// stack, and its mangled name (_ZN7testing4Test3RunEv) only reads as
/// "testing::Test::Run" once demangled — so matching it proves the demangler ran. This
/// frame resolves on both libc++ (macOS) and libstdc++ (Linux), unlike the C++
/// runtime's own terminate frames, which differ by standard library. (Windows resolves
/// names via .pdb, which fastbuild does not produce, so this is POSIX-only.)
TEST(CrashDeathTest, UnhandledExceptionTraceIsDemangled) {
    EXPECT_DEATH(
        {
            x::crash::install("test-driver");
            crash_test_unhandled_throw();
        },
        HasSubstr("testing::Test::Run")
    );
}

/// @brief it should register an alternate signal stack so the handler can run when the
/// normal stack is exhausted (stack overflow). Deterministic and in-process: it checks
/// the registration rather than triggering a real overflow, which behaves differently
/// under sanitizers.
TEST(CrashTest, InstallsAlternateSignalStack) {
    x::crash::install("test-driver");
    stack_t ss = {};
    ASSERT_EQ(sigaltstack(nullptr, &ss), 0);
    EXPECT_EQ(ss.ss_flags & SS_DISABLE, 0);
    EXPECT_NE(ss.ss_sp, nullptr);
    EXPECT_GT(ss.ss_size, 0u);
}

/// @brief it should register a handler for every fatal signal. Deterministic and
/// in-process: it inspects the installed disposition rather than raising each signal.
/// Raising the hardware-fault signals (SIGSEGV/SIGILL/SIGFPE/SIGBUS) to their default
/// action invokes slow, flaky OS crash reporting under the test sandbox, so the runtime
/// behavior is exercised once via SIGABRT below — all fatal signals share one handler.
TEST(CrashTest, InstallsHandlersForFatalSignals) {
    x::crash::install("test-driver");
    for (const int sig: {SIGSEGV, SIGABRT, SIGILL, SIGFPE, SIGBUS}) {
        SCOPED_TRACE(sig);
        struct sigaction old = {};
        ASSERT_EQ(sigaction(sig, nullptr, &old), 0);
        EXPECT_NE(old.sa_handler, SIG_DFL);
        EXPECT_NE(old.sa_handler, SIG_IGN);
    }
}

/// @brief it should print a banner naming the signal, emit a trace, and re-raise so the
/// process still dies with the signal's default disposition. SIGABRT stands in for the
/// fatal-signal set (see InstallsHandlersForFatalSignals): it shares the handler and,
/// unlike the hardware faults, terminates promptly under the sandbox.
TEST(CrashDeathTest, FatalSignalDumpsTraceAndReraises) {
    EXPECT_EXIT(
        {
            x::crash::install("test-driver");
            std::raise(SIGABRT);
        },
        KilledBySignal(SIGABRT),
        AllOf(HasSubstr("SIGABRT"), HasSubstr("0x"))
    );
}

/// @brief it should terminate cleanly when many threads crash at once, rather than
/// interleaving a flood of traces or deadlocking. The driver is multi-threaded, so a
/// fault (e.g. a use-after-free) routinely hits several threads simultaneously; the
/// handler's one-shot guard must let only the first through and take the process down.
TEST(CrashDeathTest, ConcurrentFatalSignalsTerminateCleanly) {
    EXPECT_EXIT(
        {
            x::crash::install("test-driver");
            std::vector<std::thread> threads;
            for (int i = 0; i < 8; i++)
                threads.emplace_back([] { std::raise(SIGABRT); });
            for (auto &t: threads)
                t.join();
        },
        KilledBySignal(SIGABRT),
        HasSubstr("SIGABRT")
    );
}

/// @brief it should leave SIGTERM to its default disposition. x::shutdown owns SIGINT
/// and SIGTERM for graceful shutdown, so the crash handler must not intercept them or
/// print a crash banner for them.
TEST(CrashDeathTest, DoesNotInterceptSigterm) {
    EXPECT_EXIT(
        {
            x::crash::install("test-driver");
            std::raise(SIGTERM);
        },
        KilledBySignal(SIGTERM),
        Not(HasSubstr("crashed"))
    );
}
#endif

#ifdef _WIN32
/// @brief it should install both the unhandled-exception filter and the terminate
/// handler. Inspects the registered handlers rather than triggering a crash, mirroring
/// the POSIX InstallsHandlersForFatalSignals test.
TEST(CrashTest, InstallsWindowsHandlers) {
    std::set_terminate(+[] { std::abort(); });
    const std::terminate_handler sentinel = std::get_terminate();
    x::crash::install("test-driver");
    EXPECT_NE(std::get_terminate(), sentinel);
    const LPTOP_LEVEL_EXCEPTION_FILTER prev = SetUnhandledExceptionFilter(nullptr);
    SetUnhandledExceptionFilter(prev);
    EXPECT_NE(prev, nullptr);
}
#endif
