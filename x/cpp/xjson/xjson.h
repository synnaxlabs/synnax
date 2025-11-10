// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <fstream>
#include <sstream>
#include <string>
#include <type_traits>

#include "nlohmann/json.hpp"

#include "x/cpp/xerrors/errors.h"

using json = nlohmann::json;

/// @brief general utilities for parsing configurations.
namespace xjson {
/// @brief a utility class for improving the experience of parsing JSON-based
/// configurations.
class Parser {
    /// @brief the JSON configuration being parsed.
    json config;
    /// @brief noop means the parser should fail fast.
    bool noop = false;

    Parser(
        json config,
        std::shared_ptr<std::vector<json>> errors,
        std::string path_prefix
    ):
        config(std::move(config)),
        path_prefix(std::move(path_prefix)),
        errors(std::move(errors)) {}

    template<typename T>
    T get(const std::string &path, const nlohmann::basic_json<>::iterator &iter) {
        try {
            if constexpr (std::is_constructible_v<T, xjson::Parser &>()) {
                return T(this->child("path"));
            } else if constexpr (std::is_arithmetic_v<T>) {
                if (iter->is_string()) {
                    T value;
                    std::istringstream iss(iter->get<std::string>());
                    if (!(iss >> value)) {
                        this->field_err(
                            path,
                            "expected a number, got '" + iter->get<std::string>() + "'"
                        );
                        return T();
                    }
                    return value;
                }
            }
            return iter->get<T>();
        } catch (const nlohmann::json::type_error &e) {
            this->field_err(path, e.what() + 32);
        }
        return T();
    }

    /// @brief Helper method to parse JSON and handle errors
    void parse_with_err_handling(const std::function<json()> &parser) {
        try {
            config = parser();
        } catch (const json::parse_error &e) {
            field_err("", e.what());
            noop = true;
        }
    }

public:
    /// @brief used for tracking the path of a child parser.
    const std::string path_prefix;
    /// @brief the current list of accumulated errors.
    std::shared_ptr<std::vector<json>> errors;

    /// @brief constructs a parser for accessing values on the given JSON
    /// configuration.
    explicit Parser(json config):
        config(std::move(config)), errors(std::make_shared<std::vector<json>>()) {}

    /// @brief constructs a parser for accessing values on the given stringified
    /// JSON configuration. If the string is not valid JSON, immediately binds an
    /// error to the parser.
    explicit Parser(const std::string &encoded):
        errors(std::make_shared<std::vector<json>>()) {
        parse_with_err_handling([&encoded] {
            if (encoded.empty()) return json::object();
            return json::parse(encoded);
        });
    }

    /// @brief constructs a parser from an input stream (e.g., file stream).
    /// If the stream content is not valid JSON, immediately binds an error
    /// to the parser.
    explicit Parser(std::istream &stream):
        errors(std::make_shared<std::vector<json>>()) {
        parse_with_err_handling([&stream] { return json::parse(stream); });
    }

    /// @brief default constructor constructs a parser that will fail fast.
    Parser(): noop(true), errors(std::make_shared<std::vector<json>>()) {}

    /// @brief constructs a valid, empty parser {
    explicit Parser(const bool noop):
        noop(noop), errors(std::make_shared<std::vector<json>>()) {}

    /// @brief gets the field at the given path. If the field is not found,
    /// accumulates an error in the builder.
    template<typename T>
    T required(const std::string &path) {
        if (noop) return T();
        const auto iter = config.find(path);
        if (iter == config.end()) {
            field_err(path, "This field is required");
            return T();
        }

        return get<T>(path, iter);
    }

    template<typename T, typename... Paths>
    T required(const std::string &path, const Paths &...alts) {
        if (noop) return T();
        const auto iter = config.find(path);
        if (iter != config.end()) return get<T>(path, iter);
        bool found = false;
        T result{};
        ((found = found ||
                  [&](const std::string &alt_path) {
                      const auto it = config.find(alt_path);
                      if (it != config.end()) {
                          result = get<T>(alt_path, it);
                          return true;
                      }
                      return false;
                  }(alts)),
         ...);
        if (found) return result;
        field_err(path, "this field is required");
        return T();
    }

    /// @brief gets the array field at the given path and returns a vector. If the
    /// field is not found, accumulates an error in the builder.
    /// @param path The JSON path to the vector.
    template<typename T>
    std::vector<T> required_vec(const std::string &path) {
        if (noop) return std::vector<T>();
        const auto iter = config.find(path);
        if (iter == config.end()) {
            field_err(path, "This field is required");
            return std::vector<T>();
        }
        if (!iter->is_array()) {
            field_err(path, "Expected an array");
            return std::vector<T>();
        }
        std::vector<T> values;
        for (size_t i = 0; i < iter->size(); ++i) {
            const auto child_path = path_prefix + path + "." + std::to_string(i) + ".";
            values.push_back(get<T>(child_path, iter->begin() + i));
        }
        return values;
    }

    /// @brief attempts to pull the value at the provided path. If that path is not
    /// found, returns the default. Note that this function will still accumulate an
    /// error if the path is found but the value is not of the expected type.
    /// @param path The JSON path to the value.
    /// @param default_value The default value to return if the path is not found.
    template<typename T>
    std::vector<T> optional_vec(const std::string &path, std::vector<T> default_value) {
        if (noop) return default_value;
        const auto iter = config.find(path);
        if (iter == config.end()) return default_value;
        if (!iter->is_array()) {
            field_err(path, "Expected an array");
            return default_value;
        }
        std::vector<T> values;
        for (size_t i = 0; i < iter->size(); ++i) {
            const auto child_path = path_prefix + path + "." + std::to_string(i) + ".";
            values.push_back(get<T>(child_path, iter->begin() + i));
        }
        return values;
    }

    /// @brief attempts to pull the value at the provided path. If that path is not
    /// found, returns the default. Note that this function will still accumulate an
    /// error if the path is found but the value is not of the expected type.
    /// @param path The JSON path to the value.
    /// @param default_value The default value to return if the path is not found.
    template<typename T>
    T optional(const std::string &path, T default_value) {
        if (noop) return default_value;
        const auto iter = config.find(path);
        if (iter == config.end()) return default_value;
        return get<T>(path, iter);
    }

    /// @brief gets the field at the given path and creates a new parser just for
    /// that field. The field must be an object or an array. If the field is not of
    /// the expected type, or if the field is not found, accumulates an error in the
    /// parser.
    /// @param path The JSON path to the field.
    [[nodiscard]] Parser child(const std::string &path) const {
        if (noop) return {};
        const auto iter = config.find(path);
        if (iter == config.end()) {
            field_err(path, "This field is required");
            return {};
        }
        if (!iter->is_object() && !iter->is_array()) {
            field_err(path, "Expected an object or array");
            return {};
        }
        return {*iter, errors, path_prefix + path + "."};
    }

    [[nodiscard]] Parser optional_child(const std::string &path) const {
        if (noop) return {};
        const auto iter = config.find(path);
        if (iter == config.end()) return {};
        if (!iter->is_object() && !iter->is_array()) {
            field_err(path, "Expected an object or array");
            return {};
        }
        return {*iter, errors, path_prefix + path + "."};
    }

    /// @brief Iterates over an array at the given path, executing a function for
    /// each element. If the path does not point to an array, logs an error.
    /// @param path The JSON path to the array.
    /// @param func The function to execute for each element of the array. It should
    /// take a Parser as its argument.
    void
    iter(const std::string &path, const std::function<void(Parser &)> &func) const {
        if (noop) return;
        const auto iter = config.find(path);
        if (iter == config.end()) return field_err(path, "This field is required");
        if (!iter->is_array()) return field_err(path, "Expected an array");
        for (size_t i = 0; i < iter->size(); ++i) {
            const auto child_path = path_prefix + path + "." + std::to_string(i) + ".";
            Parser childParser((*iter)[i], errors, child_path);
            func(childParser);
        }
    }

    /// @brief Maps over an array at the given path, executing a function for each
    /// element and collecting the results into a vector. If the path does not point
    /// to an array, logs an error and returns an empty vector.
    /// @param path The JSON path to the array.
    /// @param func The function to execute for each element of the array. It should
    /// take a Parser as its argument and return a value of type T.
    /// @return A vector containing the results of applying func to each element.
    template<typename T>
    [[nodiscard]] std::vector<T>
    map(const std::string &path,
        const std::function<std::pair<T, bool>(Parser &)> &func) const {
        if (noop) return {};
        const auto iter = config.find(path);
        if (iter == config.end()) {
            field_err(path, "This field is required");
            return {};
        }
        if (!iter->is_array()) {
            field_err(path, "Expected an array");
            return {};
        }
        std::vector<T> results;
        results.reserve(iter->size());
        for (size_t i = 0; i < iter->size(); ++i) {
            const auto child_path = path_prefix + path + "." + std::to_string(i) + ".";
            Parser childParser((*iter)[i], errors, child_path);
            auto [res, ok] = func(childParser);
            if (ok) results.push_back(std::move(res));
        }
        return results;
    }

    void field_err(const std::string &path, const xerrors::Error &err) const {
        this->field_err(path, err.message());
    }

    /// @brief binds a new error to the field at the given path.
    /// @param path The JSON path to the field.
    /// @param message The error message to bind.
    void field_err(const std::string &path, const std::string &message) const {
        if (this->noop || this->errors == nullptr) return;
        this->errors->push_back({{"path", path_prefix + path}, {"message", message}});
    }

    /// @returns true if the parser has accumulated no errors, false otherwise.
    [[nodiscard]] bool ok() const {
        if (this->noop) return false;
        return this->errors == nullptr || this->errors->empty();
    }

    /// @returns the parser's errors as a JSON object of the form {"errors":
    /// [ACCUMULATED_ERRORS]}.
    [[nodiscard]] json error_json() const {
        json err;
        err["errors"] = *errors;
        return err;
    }

    [[nodiscard]] xerrors::Error error() const {
        if (this->errors->empty()) return xerrors::Error{};
        if (this->errors->size() == 1) {
            const auto &err = this->errors->at(0);
            if (err["path"].get<std::string>().empty())
                return xerrors::Error{
                    xerrors::VALIDATION,
                    err["message"].get<std::string>()
                };
        }
        return xerrors::Error{xerrors::VALIDATION, error_json().dump()};
    }

    /// @returns the parser's errors as a JSON object of the form {"errors":
    /// [ACCUMULATED_ERRORS]}.
    [[nodiscard]] json get_json() const { return config; }

    /// @brief creates a parser from a file at the given path
    /// @param path The path to the JSON configuration file
    /// @return A parser for the configuration file
    static Parser from_file_path(const std::string &path) {
        std::ifstream file(path);
        if (!file.is_open()) {
            Parser p(false);
            p.field_err("", "failed to open file: " + path);
            return p;
        }
        return Parser(file);
    }
};
}
