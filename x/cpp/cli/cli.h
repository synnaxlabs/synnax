// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <iostream>
#include <string>

/// @brief provides utilities for building interactive CLI applications.
namespace cli {
/// @brief prompts the user to enter a value.
/// @param prompt the message to display to the user.
/// @param hide_input
inline std::string prompt(const std::string &prompt, bool hide_input = false) {
    std::string input;
#ifdef _WIN32
    HANDLE h_stdin = GetStdHandle(STD_INPUT_HANDLE);
    DWORD mode;
    GetConsoleMode(h_stdin, &mode);
    if (hide_input)
        SetConsoleMode(h_stdin, mode & (~ENABLE_ECHO_INPUT));
#else
    if (hide_input)
        system("stty -echo");
#endif

    std::cout << prompt;
    std::getline(std::cin, input);

    if (hide_input) {
        std::cout << std::endl;
#ifdef _WIN32
        SetConsoleMode(h_stdin, mode);
#else
        system("stty echo");
#endif
    }
    return input;
}
}
