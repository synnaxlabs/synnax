// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <fstream>
#ifdef _WIN32
#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#include <windows.h>
#else
#include <sys/stat.h>
#endif
#include <filesystem>

#include "nlohmann/json.hpp"

#include "x/cpp/kv/kv.h"
#include "x/cpp/xerrors/errors.h"

using json = nlohmann::json;

namespace kv {

namespace {
xerrors::Error
check_minimum_permissions(const std::filesystem::path &path, const char *context) {
    auto perms = std::filesystem::status(path).permissions();
    if ((perms & std::filesystem::perms::owner_write) == std::filesystem::perms::none ||
        (perms & std::filesystem::perms::owner_read) == std::filesystem::perms::none) {
        return xerrors::Error("insufficient permissions on " + std::string(context));
    }
    return xerrors::NIL;
}
}

JSONFile::JSONFile(const std::string &path, const json &data):
    path(path), data_(data) {}

std::pair<std::shared_ptr<kv::KV>, xerrors::Error>
JSONFile::open(const JSONFileConfig &config) {
    try {
        const auto dir = config.path.parent_path();
        if (!std::filesystem::exists(dir)) {
            auto parent_dir = dir.parent_path();
            if (std::filesystem::exists(parent_dir)) {
                if (auto err = check_minimum_permissions(
                        parent_dir,
                        "parent directory"
                    ))
                    return {
                        std::make_shared<JSONFile>(
                            config.path.string(),
                            json::object()
                        ),
                        err
                    };
            }

            std::filesystem::create_directories(dir);
            std::filesystem::permissions(dir, config.dir_mode);
        }

        const auto exists = std::filesystem::exists(config.path);
        if (!exists) {
            std::ofstream file(config.path);
            if (!file.is_open())
                return {
                    std::make_shared<JSONFile>(config.path.string(), json::object()),
                    xerrors::Error("failed to open file")
                };
            file << "{}";
            file.close();

            std::filesystem::permissions(config.path, config.file_mode);
            if (auto err = check_minimum_permissions(config.path, "file"))
                return {
                    std::make_shared<JSONFile>(config.path.string(), json::object()),
                    err
                };

            return {
                std::make_shared<JSONFile>(config.path.string(), json::object()),
                xerrors::NIL
            };
        }

        // Read existing JSON data
        std::ifstream file(config.path);
        if (!file.is_open())
            return {
                std::make_shared<JSONFile>(config.path.string(), json::object()),
                xerrors::Error("failed to open file")
            };
        json data;
        try {
            file >> data;
        } catch (const json::exception &e) {
            return {
                std::make_shared<JSONFile>(config.path.string(), json::object()),
                xerrors::Error("failed to parse JSON: " + std::string(e.what()))
            };
        }
        file.close();
        return {std::make_shared<JSONFile>(config.path.string(), data), xerrors::NIL};
    } catch (const std::exception &e) {
        return {
            std::make_shared<JSONFile>(config.path.string(), json::object()),
            xerrors::Error("filesystem operation failed: " + std::string(e.what()))
        };
    }
}

xerrors::Error JSONFile::set(const std::string &key, const std::string &value) {
    data_[key] = value;

    std::ofstream file(path);
    if (!file.is_open()) { return xerrors::Error("failed to open file for writing"); }
    file << data_.dump(4);
    file.close();
    return xerrors::NIL;
}

xerrors::Error JSONFile::get(const std::string &key, std::string &value) {
    if (!data_.contains(key)) return xerrors::NOT_FOUND;
    try {
        value = data_[key].get<std::string>();
    } catch (const json::exception &e) {
        return xerrors::Error("failed to get value: " + std::string(e.what()));
    }
    return xerrors::NIL;
}

xerrors::Error JSONFile::del(const std::string &key) {
    if (!data_.contains(key)) return xerrors::NIL;
    data_.erase(key);
    std::ofstream file(path);
    if (!file.is_open()) { return xerrors::Error("failed to open file for writing"); }
    file << data_.dump(4);
    file.close();
    return xerrors::NIL;
}

}
