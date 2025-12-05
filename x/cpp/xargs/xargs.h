// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <string>
#include <type_traits>
#include <vector>

#include "x/cpp/caseconv/caseconv.h"
#include "x/cpp/xerrors/errors.h"

namespace xargs {
/// @brief Parser provides a simple command-line argument parsing utility that
/// supports required arguments, optional arguments with default values, and flags.
///
/// The Parser supports three main types of arguments:
/// 1. Required arguments: Must be provided in the command line
/// 2. Optional arguments: Can have default values if not provided
/// 3. Flags: Boolean values that are true if present, false if not
///
/// Arguments can be specified in three formats:
/// - Long form: --argument-name=value or --argument-name value
/// - Short form: -a=value or -a value
/// - Snake case is automatically converted to kebab case: my_arg -> --my-arg
///
/// You're required to specify both the short and long form for an argument. So you
/// need to do p.flag("--arm", "-a") in order to match "-a" as well as "--arm".
/// "--arm" won't auto-match "-a".
///
/// Example usage:
/// @code
/// int main(int argc, char* argv[]) {
///     xargs::Parser parser(argc, argv);
///
///     // Required argument
///     auto name = parser.field<std::string>("name");
///
///     // Optional argument with default
///     auto count = parser.field<int>("count", 10);
///
///     // Flag
///     auto verbose = parser.flag("verbose", "v");
///
///     if (parser.error()) {
///         std::cerr << parser.error().message() << std::endl;
///         return 1;
///     }
/// }
/// @endcode
class Parser {
    // Helper struct to convert argument names to their standard forms that would
    // be used in the command line i.e. --arg or -arg
    struct ArgVariants {
        std::string single; // Single dash form i.e. -arg
        std::string double_; // Double dash form i.e. --arg
    };

    static ArgVariants normalize_arg_name(const std::string &name) {
        if (name.empty()) return {"", ""};
        std::string stripped = name;
        if (name[0] == '-')
            stripped = name.substr(name[0] == '-' && name[1] == '-' ? 2 : 1);

        const std::string kebab = caseconv::snake_to_kebab(stripped);
        return {"-" + kebab, "--" + kebab};
    }

    // Helper to check if an argument matches any of its normalized forms
    static bool matches_arg(
        const std::string &arg,
        const ArgVariants &norm,
        const bool check_equals = true
    ) {
        if (arg == norm.single || arg == norm.double_) return true;
        if (check_equals) {
            return arg.compare(0, norm.double_.length() + 1, norm.double_ + "=") == 0 ||
                   arg.compare(0, norm.single.length() + 1, norm.single + "=") == 0;
        }
        return false;
    }

    /// @brief Searches for an argument in the command line arguments
    /// @tparam Args Variadic template for multiple possible argument names
    /// @param names The possible names of the argument to search for
    /// @return A pair containing the argument value and a boolean indicating if
    /// found
    template<typename... Args>
    std::pair<std::string, bool> find_arg(const Args &...names) {
        std::pair<std::string, bool> last_found = {"", false};
        for (size_t i = 0; i < argv_.size(); i++) {
            const std::string &arg = argv_[i];
            for (const auto &name: {names...}) {
                auto norm = normalize_arg_name(name);
                std::string prefix = norm.double_ + "=";
                if (arg.compare(0, prefix.length(), prefix) == 0) {
                    last_found = {arg.substr(arg.find('=') + 1), true};
                    continue;
                }
                if (matches_arg(arg, norm, false) && i + 1 < argv_.size()) {
                    last_found = {argv_[i + 1], true};
                }
            }
        }
        return last_found;
    }

    /// @brief Adds an error to the parser's error list
    /// @param name The name of the argument that caused the error
    /// @param msg The error message
    void add_error(const std::string &name, const char *msg) {
        errors.emplace_back(name, msg);
    }

    /// @brief Splits a string by comma delimiter
    /// @param str The string to split
    /// @return Vector of substrings
    static std::vector<std::string> split_by_comma(const std::string &str) {
        std::vector<std::string> result;
        std::string current;
        for (const char c: str) {
            if (c == ',') {
                if (!current.empty()) {
                    result.push_back(current);
                    current.clear();
                }
            } else
                current += c;
        }
        if (!current.empty()) result.push_back(current);
        return result;
    }

    /// @brief Parses a string value into the specified type
    /// @tparam T The target type to parse into
    /// @param value The string value to parse
    /// @param name The name of the argument (for error reporting)
    /// @param error_msg The error message to use if parsing fails
    /// @return The parsed value of type T
    template<typename T>
    T parse_value(
        const std::string &value,
        const std::string &name,
        const char *error_msg
    ) {
        try {
            if constexpr (std::is_same_v<T, std::string>)
                return value;
            else if constexpr (std::is_same_v<T, std::vector<std::string>>)
                return split_by_comma(value);
            else if constexpr (std::is_floating_point_v<T>)
                return static_cast<T>(std::stold(value));
            else if constexpr (std::is_integral_v<T> && !std::is_same_v<T, bool>)
                return static_cast<T>(std::stoll(value));
            else if constexpr (std::is_same_v<T, bool>)
                return value == "true" || value == "1";
            else if constexpr (std::is_same_v<T, const char *>)
                return value.c_str();
            else if constexpr (std::is_same_v<T, std::vector<int>>) {
                const auto strings = split_by_comma(value);
                std::vector<int> result;
                result.reserve(strings.size());
                for (const auto &s: strings)
                    result.push_back(std::stoi(s));
                return result;
            } else if constexpr (std::is_same_v<T, std::vector<double>>) {
                const auto strings = split_by_comma(value);
                std::vector<double> result;
                result.reserve(strings.size());
                for (const auto &s: strings)
                    result.push_back(std::stod(s));
                return result;
            }

            add_error(name, "Unsupported type");
            return T();
        } catch (const std::exception &) {
            add_error(name, error_msg);
            return T();
        }
    }

    /// @brief Handles required argument parsing and error checking
    /// @tparam T The type to parse the argument into
    /// @param name The name of the required argument
    /// @param error_msg The error message to use if parsing fails
    /// @return The parsed value of type T
    template<typename T>
    T handle_required(const std::string &name, const char *error_msg) {
        const auto [value, found] = find_arg(name);
        if (!found) {
            errors.emplace_back(xerrors::VALIDATION, name + ": required argument not found");
            return T();
        }
        return parse_value<T>(value, name, error_msg);
    }

    /// @brief Checks if an argument exists in the command line arguments
    /// @tparam Args Variadic template for multiple possible argument names
    /// @param names The possible names of the argument to check for
    /// @return true if the argument exists, false otherwise
    template<typename... Args>
    bool has_arg(const Args &...names) {
        for (const auto &arg: argv_) {
            for (const auto &name: {names...}) {
                if (matches_arg(arg, normalize_arg_name(name))) return true;
            }
        }
        return false;
    }

public:
    std::vector<std::string> argv_; ///< The command line arguments
    std::vector<xerrors::Error> errors; ///< Any errors encountered during parsing

    Parser() = default;

    /// @brief Constructs a parser from argc and argv
    /// @param argc The argument count
    /// @param argv The argument values array
    explicit Parser(const int argc, char *argv[]): argv_(argv, argv + argc) {}

    /// @brief Constructs a parser from a vector of strings
    /// @param argv The vector of argument strings
    explicit Parser(std::vector<std::string> argv): argv_(std::move(argv)) {}

    /// @brief Parses a required argument
    /// @tparam T The type to parse the argument into
    /// @param name The name of the required argument
    /// @return The parsed value of type T
    /// @throws Adds an error if the argument is missing or invalid
    template<typename T>
    T field(const std::string &name) {
        return handle_required<T>(name, "Invalid value");
    }

    /// @brief Parses an optional argument with a default value
    /// @tparam T The type to parse the argument into
    /// @param name The name of the optional argument
    /// @param default_value The default value to use if the argument is not
    /// provided
    /// @return The parsed value or the default value
    template<typename T>
    T field(const std::string &name, const T &default_value) {
        const auto [value, found] = find_arg(name);
        if (!found) return default_value;
        return parse_value<T>(value, name, "Invalid value");
    }

    /// @brief Convenience overload for string optional arguments with const char*
    /// defaults
    /// @param name The name of the optional argument
    /// @param default_value The default string value
    /// @return The parsed string or the default value
    std::string field(const std::string &name, const char *default_value) {
        return field<std::string>(name, default_value);
    }

    /// @brief Checks if a flag is present in the command line arguments
    /// @tparam Args Variadic template for multiple possible flag names
    /// @param names The possible names of the flag to check for
    /// @return true if the flag is present, false otherwise
    template<typename... Args>
    [[nodiscard]] bool flag(const Args &...names) {
        // Just check if the flag exists, don't look for a value after it
        return has_arg(names...);
    }

    /// @brief Returns the first error encountered during parsing
    /// @return The first error or xerrors::NIL if no errors
    [[nodiscard]] xerrors::Error error() const {
        if (errors.empty()) return xerrors::NIL;
        return errors.at(0);
    }

    /// @brief Gets the argument at the specified index
    /// @param index The index of the argument to retrieve
    /// @param error_msg The error message to use if the index is out of bounds
    /// @return The argument at the specified index or empty string if out of bounds
    std::string at(const int index, const std::string &error_msg) {
        if (static_cast<size_t>(index) >= argv_.size()) {
            errors.emplace_back("index", error_msg);
            return "";
        }
        return argv_[index];
    }
};
}
