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
#include <memory>
#include <mutex>
#include <optional>
#include <string>
#include <unordered_map>
#include <utility>

#include "client/cpp/task/task.h"
#include "x/cpp/errors/errors.h"

namespace driver::bus {
/// @brief Holders gives each key to one holder at a time. It is safe for concurrent
/// use.
class Holders {
    /// @brief Holder is the holder of a key.
    struct Holder {
        /// @brief id is the key of the task or device that holds the key.
        std::string id;
        std::string name;
        /// @brief claims is the number of claims that the holder has on the key.
        std::size_t claims = 0;
    };

    std::mutex mu;
    std::unordered_map<std::string, Holder> held;

public:
    /// @brief Claim holds a key until it is destroyed.
    class Claim {
        std::shared_ptr<Holders> holders;
        std::string key;
        Claim(std::shared_ptr<Holders> holders, std::string key);
        friend class Holders;

    public:
        ~Claim();
        Claim(const Claim &) = delete;
        Claim &operator=(const Claim &) = delete;
    };

    /// @brief claims key in holders for the holder with id until the claim is
    /// destroyed. A holder can claim a key that it holds again.
    /// @returns the claim, or the name of the holder of key when another holder has it.
    static std::pair<std::unique_ptr<Claim>, std::optional<std::string>> take(
        const std::shared_ptr<Holders> &holders,
        const std::string &key,
        const std::string &id,
        const std::string &name
    );
};

/// @brief Readers gives each device of an integration one read task at a time, since
/// two tasks that read one stream would each get only part of it. It is safe for
/// concurrent use.
class Readers {
    std::string kind;
    std::shared_ptr<Holders> holders = std::make_shared<Holders>();

public:
    /// @param kind names the devices in errors, such as "serial".
    explicit Readers(std::string kind);

    /// @brief claims the device with key for task until the claim is destroyed. A task
    /// can claim a device that it holds again.
    /// @returns CONFIGURATION_ERROR naming the read task that holds the device.
    std::pair<std::unique_ptr<Holders::Claim>, x::errors::Error>
    claim(const std::string &device, const synnax::task::Task &task);
};
}
