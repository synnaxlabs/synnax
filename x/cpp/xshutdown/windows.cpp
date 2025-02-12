// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//
// Created by Emiliano Bonilla on 2/11/25.
//

#include "xshutdown.h"
#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#include <windows.h>

namespace xshutdown {

static Listen* active_listener = nullptr;

static BOOL WINAPI console_ctrl_handler(DWORD ctrl_type) {
    if (ctrl_type == CTRL_C_EVENT && active_listener != nullptr) {
        active_listener->signal_shutdown();
        return TRUE;
    }
    return FALSE;
}

void Listen::listen_signal() {
    active_listener = this;
    SetConsoleCtrlHandler(console_ctrl_handler, TRUE);
}

void Listen::listen_stdin() {
    HANDLE hStdin = GetStdHandle(STD_INPUT_HANDLE);
    DWORD mode;
    GetConsoleMode(hStdin, &mode);
    SetConsoleMode(hStdin, mode & ~ENABLE_MOUSE_INPUT & ~ENABLE_WINDOW_INPUT);
    
    char buffer[256];
    std::string input;
    
    while (true) {
        if (should_shutdown()) break;
        
        // Check for input
        DWORD available;
        INPUT_RECORD record;
        PeekConsoleInput(hStdin, &record, 1, &available);
        
        if (available > 0) {
            DWORD read;
            if (ReadConsole(hStdin, buffer, sizeof(buffer) - 1, &read, nullptr)) {
                buffer[read] = '\0';
                input += buffer;
                
                // Process complete lines
                size_t pos;
                while ((pos = input.find('\n')) != std::string::npos) {
                    std::string line = input.substr(0, pos);
                    if (!line.empty() && line[line.length()-1] == '\r') {
                        line = line.substr(0, line.length()-1);
                    }
                    input.erase(0, pos + 1);
                    
                    if (line == "STOP") {
                        signal_shutdown();
                        return;
                    }
                }
            }
        }
        
        Sleep(100); // Prevent busy waiting
    }
}

void Listen::listen() {
    Listen listener;
    listener.listen_signal();
    listener.listen_stdin();
}

bool Listen::should_shutdown() const {
    std::lock_guard<std::mutex> lock(mu);
    return should_stop;
}

void Listen::signal_shutdown() {
    {
        std::lock_guard<std::mutex> lock(mu);
        should_stop = true;
    }
    cv.notify_all();
}

} // namespace shutdown
