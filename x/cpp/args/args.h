// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <type_traits>
#include <vector>

#include "x/cpp/caseconv/caseconv.h"
#include "x/cpp/errors/errors.h"

namespace x::args {
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
///     args::Parser parser(argc, argv);
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
    struct ArgVariants {
        std::string single;
        std::string double_;
    };

    static ArgVariants normalize_arg_name(const std::string &name) {
        if (name.empty()) return {"", ""};
        std::string stripped = name;
        if (name[0] == '-')
            stripped = name.substr(name[0] == '-' && name[1] == '-' ? 2 : 1);

        const std::string kebab = caseconv::snake_to_kebab(stripped);
        return {"-" + kebab, "--" + kebab};
    }

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

    template<typename... Args>
    std::pair<std::string, bool> find_arg(const Args &...names) {
        std::pair<std::string, bool> last_found = {"", false};
        for (size_t i = 0; i < this->argv.size(); i++) {
            const std::string &arg = this->argv[i];
            for (const auto &name: {names...}) {
                auto norm = normalize_arg_name(name);
                std::string prefix = norm.double_ + "=";
                if (arg.compare(0, prefix.length(), prefix) == 0) {
                    last_found = {arg.substr(arg.find('=') + 1), true};
                    continue;
                }
                if (matches_arg(arg, norm, false) && i + 1 < this->argv.size()) {
                    last_found = {this->argv[i + 1], true};
                }
            }
        }
        return last_found;
    }

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

            this->field_err(name, "Unsupported type");
            return T();
        } catch (const std::exception &) {
            this->field_err(name, error_msg);
            return T();
        }
    }

    template<typename T>
    T handle_required(const std::string &name, const char *error_msg) {
        const auto [value, found] = find_arg(name);
        if (!found) {
            this->field_err(name, "required argument not found");
            return T();
        }
        return parse_value<T>(value, name, error_msg);
    }

    template<typename... Args>
    bool has_arg(const Args &...names) {
        for (const auto &arg: this->argv) {
            for (const auto &name: {names...}) {
                if (matches_arg(arg, normalize_arg_name(name))) return true;
            }
        }
        return false;
    }

public:
    /// @brief the command line arguments.
    std::vector<std::string> argv;
    /// @brief any errors encountered during parsing.
    std::vector<errors::Error> errors;

    Parser() = default;

    /// @brief constructs a parser from argc and argv.
    explicit Parser(const int argc, char *argv[]): argv(argv, argv + argc) {}

    /// @brief constructs a parser from a vector of strings.
    explicit Parser(std::vector<std::string> argv): argv(std::move(argv)) {}

    /// @brief parses a required argument.
    template<typename T>
    T field(const std::string &name) {
        return handle_required<T>(name, "Invalid value");
    }

    /// @brief parses an optional argument with a default value.
    template<typename T>
    T field(const std::string &name, const T &default_value) {
        const auto [value, found] = find_arg(name);
        if (!found) return default_value;
        return parse_value<T>(value, name, "Invalid value");
    }

    /// @brief convenience overload for string optional arguments with const char*
    /// defaults.
    std::string field(const std::string &name, const char *default_value) {
        return field<std::string>(name, default_value);
    }

    /// @brief binds an error to the given field name.
    void field_err(const std::string &name, const std::string &message) {
        this->errors.emplace_back(errors::VALIDATION, name + ": " + message);
    }

    /// @brief binds an error to the given field name from an existing error.
    void field_err(const std::string &name, const errors::Error &err) {
        this->field_err(name, err.data);
    }

    /// @returns true if no errors have been accumulated.
    [[nodiscard]] bool ok() const { return this->errors.empty(); }

    /// @brief checks if a flag is present in the command line arguments.
    template<typename... Args>
    [[nodiscard]] bool flag(const Args &...names) {
        return has_arg(names...);
    }

    /// @brief returns the first error encountered during parsing, or NIL.
    [[nodiscard]] errors::Error error() const {
        if (this->errors.empty()) return errors::NIL;
        return this->errors.at(0);
    }

    /// @brief gets the argument at the specified index.
    std::string at(const int index, const std::string &error_msg) {
        if (static_cast<size_t>(index) >= this->argv.size()) {
            this->field_err("index", error_msg);
            return "";
        }
        return this->argv[index];
    }
};
}
