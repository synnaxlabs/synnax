// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <functional>
#include <memory>
#include <mutex>
#include <utility>

#include "x/cpp/errors/errors.h"

namespace x::lib {
/// @brief a value that loads on first use, such as a vendor library. A failed load runs
/// again on the next use, and the first value that loads is kept. Safe for concurrent
/// use.
template<typename T>
class Lazy {
public:
    /// @brief returns the loaded value, or nullptr and the reason the load failed.
    using Loader = std::function<std::pair<std::shared_ptr<T>, x::errors::Error>()>;

    explicit Lazy(Loader load): load(std::move(load)) {}

    /// @brief returns the value, loading it when no load has succeeded yet. Blocks
    /// while another caller loads.
    std::pair<std::shared_ptr<T>, x::errors::Error> get() {
        std::lock_guard lock(this->mu);
        if (this->value == nullptr) {
            auto [value, err] = this->load();
            if (err) return {nullptr, err};
            this->value = std::move(value);
        }
        return {this->value, x::errors::NIL};
    }

private:
    const Loader load;
    std::mutex mu;
    std::shared_ptr<T> value;
};
}
