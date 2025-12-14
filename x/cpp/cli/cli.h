// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#ifdef _WIN32
#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#include <windows.h>
#else
#include <termios.h>
#include <unistd.h>
#endif

#include <cstdint>
#include <iostream>
#include <optional>
#include <string>

/// @brief provides utilities for building interactive CLI applications.
namespace cli {
/// @brief prompts the user to enter a value.
/// @param message the message to display to the user.
/// @param default_value optional default value to use if input is empty
/// @param hide_input whether to hide the input (for passwords)
inline std::string prompt(
    const std::string &message,
    std::optional<std::string> default_value = std::nullopt,
    bool hide_input = false
) {
    while (true) {
        std::string prompt_text = message;
        if (default_value.has_value()) prompt_text += " [" + *default_value + "]";
        prompt_text += ": ";

#ifdef _WIN32
        HANDLE h_stdin = GetStdHandle(STD_INPUT_HANDLE);
        DWORD mode;
        GetConsoleMode(h_stdin, &mode);
        if (hide_input) SetConsoleMode(h_stdin, mode & (~ENABLE_ECHO_INPUT));
#else
        struct termios oldt{};
        if (hide_input) {
            if (tcgetattr(STDIN_FILENO, &oldt) == 0) {
                struct termios newt = oldt;
                newt.c_lflag &= ~ECHO;
                tcsetattr(STDIN_FILENO, TCSANOW, &newt);
            } else {
                std::cerr << "warning: failed to hide input" << '\n';
            }
        }
#endif

        std::string input;
        std::cout << prompt_text;
        std::getline(std::cin, input);

        if (hide_input) {
            std::cout << '\n';
#ifdef _WIN32
            SetConsoleMode(h_stdin, mode);
#else
            tcsetattr(STDIN_FILENO, TCSANOW, &oldt);
#endif
        }

        if (!input.empty() || default_value.has_value())
            return input.empty() ? *default_value : input;
    }
}

/// @brief prompts the user to confirm an action with a yes/no question.
/// @param message the confirmation message to display to the user.
/// @param default_value optional default value to use if input is empty
/// @return true if the user confirms (Y/y), false if denied (N/n).
inline bool
confirm(const std::string &message, std::optional<bool> default_value = std::nullopt) {
    while (true) {
        std::string input = prompt(
            message + " (Y/N)",
            default_value.has_value()
                ? std::optional<std::string>(default_value.value() ? "Y" : "N")
                : std::nullopt
        );
        if (input.empty() || input.size() > 1) continue;
        const char response = static_cast<char>(
            std::toupper(static_cast<unsigned char>(input[0]))
        );
        if (response == 'Y') return true;
        if (response == 'N') return false;
        std::cout << "Please enter Y or N \n";
    }
}

/// @brief prompts the user to enter a numeric value.
/// @tparam T the numeric type (int, float, double, etc.)
/// @param message the message to display to the user.
/// @param default_value optional default value to use if input is empty
template<typename T>
inline T
prompt(const std::string &message, std::optional<T> default_value = std::nullopt) {
    static_assert(
        std::is_arithmetic_v<T>,
        "Template parameter T must be an arithmetic type"
    );
    while (true) {
        const std::string &prompt_text = message;
        std::string default_str;
        if (default_value.has_value()) default_str = std::to_string(*default_value);
        const std::string input = prompt(prompt_text, default_str);
        try {
            if constexpr (std::is_same_v<T, int>)
                return std::stoi(input);
            else if constexpr (std::is_same_v<T, float>)
                return std::stof(input);
            else if constexpr (std::is_same_v<T, double>)
                return std::stod(input);
            else if constexpr (std::is_same_v<T, std::int64_t>)
                return std::stol(input);
            else if constexpr (std::is_same_v<T, std::uint16_t> ||
                               std::is_same_v<
                                   T,
                                   unsigned short>) // NOLINT(*-runtime-int)
                return static_cast<T>(std::stoul(input));
            else
                static_assert(sizeof(T) == 0, "Unsupported numeric type");
        } catch (const std::exception &) {
            std::cout << "Invalid input: please enter a valid number \n";
        }
    }
}
}
