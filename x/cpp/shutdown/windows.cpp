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
#include <windows.h>

#include "x/cpp/shutdown/shutdown.h"

namespace x::shutdown::priv {
std::mutex shutdown_mutex;
std::condition_variable shutdown_cv;
bool should_stop = false;

BOOL WINAPI console_ctrl_handler(DWORD ctrl_type) noexcept {
    if (ctrl_type == CTRL_C_EVENT) {
        signal_shutdown();
        return TRUE;
    }
    return FALSE;
}

void listen_signal() {
    SetConsoleCtrlHandler(console_ctrl_handler, TRUE);
}

void listen_stdin() {
    HANDLE hStdin = GetStdHandle(STD_INPUT_HANDLE);
    char buffer[256];
    std::string input;

    while (true) {
        if (should_shutdown()) break;

        DWORD bytesRead;
        if (ReadFile(hStdin, buffer, sizeof(buffer) - 1, &bytesRead, nullptr)) {
            if (bytesRead > 0) {
                buffer[bytesRead] = '\0';
                input += buffer;

                size_t pos;
                while ((pos = input.find('\n')) != std::string::npos) {
                    std::string line = input.substr(0, pos);
                    if (!line.empty() && line[line.length() - 1] == '\r') {
                        line = line.substr(0, line.length() - 1);
                    }
                    input.erase(0, pos + 1);

                    if (line == "STOP") {
                        signal_shutdown();
                        return;
                    }
                }
            }
        }

        Sleep(100);
    }
}
}
