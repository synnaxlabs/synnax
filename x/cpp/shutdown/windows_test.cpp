// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#include <cstdlib>

#include "gtest/gtest.h"
#include <windows.h>

#include "x/cpp/shutdown/shutdown.h"

namespace x::shutdown {
/// @brief replaces stdin with a new pipe and returns the pipe's write end.
HANDLE pipe_stdin() {
    HANDLE read_end = nullptr;
    HANDLE write_end = nullptr;
    if (!CreatePipe(&read_end, &write_end, nullptr, 0)) std::abort();
    if (!SetStdHandle(STD_INPUT_HANDLE, read_end)) std::abort();
    return write_end;
}

/// @brief it should end the process when the stdin pipe breaks.
TEST(ListenDeathTest, BrokenStdinPipeEndsProcess) {
    EXPECT_EXIT(
        {
            CloseHandle(pipe_stdin());
            listen(false, true);
        },
        ::testing::ExitedWithCode(1),
        ""
    );
}

/// @brief it should return on a STOP that arrives before the pipe breaks.
TEST(ListenDeathTest, StopBeforeBrokenPipeReturns) {
    EXPECT_EXIT(
        {
            const HANDLE write_end = pipe_stdin();
            DWORD written = 0;
            WriteFile(write_end, "STOP\n", 5, &written, nullptr);
            CloseHandle(write_end);
            listen(false, true);
            std::exit(0);
        },
        ::testing::ExitedWithCode(0),
        ""
    );
}
}
