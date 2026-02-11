// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "nlohmann/json.hpp"

#include "x/cpp/errors/errors.h"

using json = nlohmann::json;

namespace x::kv {
class KV {
public:
    virtual ~KV() = default;

    virtual errors::Error set(const std::string &key, const std::string &value) = 0;

    virtual errors::Error get(const std::string &key, std::string &value) = 0;

    virtual errors::Error del(const std::string &key) = 0;
};

/// @brief the configuration for a JSONFileKV.
struct JSONFileConfig {
    /// @brief the path to the file.
    std::filesystem::path path;
    /// @brief the mode to set the directory to if it doesn't exist.
    std::filesystem::perms dir_mode;
    /// @brief the mode to set the file to if it doesn't exist.
    std::filesystem::perms file_mode;
};

/// @brief a KV implementation backed by a JSOn file on-disk.
class JSONFile final : public KV {
    std::filesystem::path path;
    json data_{};

public:
    /// @brief opens the key-value store using the provided configuration. If the
    /// file does not exist, it will be created.
    static std::pair<std::shared_ptr<KV>, errors::Error>
    open(const JSONFileConfig &config);

    /// @brief implements KV.
    errors::Error set(const std::string &key, const std::string &value) override;

    /// @brief implements KV.
    errors::Error get(const std::string &key, std::string &value) override;

    /// @brief implements KV.
    errors::Error del(const std::string &key) override;

    JSONFile(const std::string &path, const json &data);
};
}
