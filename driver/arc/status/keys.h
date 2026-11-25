// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <cstdint>
#include <filesystem>
#include <list>
#include <memory>
#include <mutex>
#include <string>
#include <unordered_map>
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"

#include "driver/journal/journal.h"

namespace driver::arc::status {
/// @brief the magic of a status keys file.
constexpr journal::Magic KEYS_MAGIC = {'S', 'Y', 'S', 'T', 'K', 'E', 'Y', 'S'};
/// @brief the record type of a name and its key in a status keys file.
constexpr std::uint8_t KEY_TYPE = 1;
/// @brief the record type of the key of the Core cluster that gave the keys.
constexpr std::uint8_t CLUSTER_TYPE = 2;
/// @brief the default number of names whose key the Driver keeps.
constexpr std::size_t DEFAULT_MAX_KEYS = 1024;
/// @brief the file in the data directory that holds the status keys.
const std::string KEYS_FILE = "status_keys";

/// @brief the keys that the Core gave to status names, kept in a file across
/// restarts. Holds the max most recently used names. Safe for concurrent use.
class Keys {
    std::filesystem::path path;
    std::string cluster;
    std::size_t max;
    journal::OnCreate on_create;
    /// @brief held for a whole save, so saves reach the disk in order. Taken before
    /// mu.
    std::mutex save_mu;
    std::mutex mu;
    /// @brief the names and keys, the most recently used first.
    std::list<std::pair<std::string, std::string>> entries;
    std::unordered_map<std::string, decltype(entries)::iterator> index;

    Keys(
        std::filesystem::path path,
        std::string cluster,
        std::size_t max,
        journal::OnCreate on_create
    );
    /// @brief puts name first with key, and forgets the least recently used name past
    /// max. Requires mu.
    void put(const std::string &name, const std::string &key);
    /// @brief replaces the file with entries, the least recently used first.
    /// Requires save_mu.
    [[nodiscard]] x::errors::Error
    save(const std::vector<std::pair<std::string, std::string>> &entries) const;

public:
    /// @brief loads the keys in the file at path that the Core cluster with the key
    /// cluster gave, or none when the file does not exist or another cluster gave
    /// them, and keeps the max most recently used. Removes a temp file that a crash
    /// during a save left. Each save calls on_create with the temp file before it
    /// takes the place of the old file.
    /// @returns journal::CORRUPT when the file is not whole. UNKNOWN_MAGIC,
    /// UNKNOWN_VERSION, or UNKNOWN_TYPE when it is not a keys file this Driver
    /// reads. An error from the file.
    static std::pair<std::shared_ptr<Keys>, x::errors::Error> open(
        const std::filesystem::path &path,
        const std::string &cluster,
        std::size_t max = DEFAULT_MAX_KEYS,
        journal::OnCreate on_create = journal::no_create_hook
    );

    /// @returns the key of name, or "" when the Core has not given name a key.
    [[nodiscard]] std::string get(const std::string &name);

    /// @brief records key as the key of name. When it changes, replaces the file,
    /// so a crash leaves the old file or the new one, and returns after the new file
    /// is on the disk. get does not wait for the disk.
    /// @returns an error from the file, and then the keys still hold key in memory.
    x::errors::Error set(const std::string &name, const std::string &key);
};
}
