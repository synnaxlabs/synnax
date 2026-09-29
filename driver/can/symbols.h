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
#include <vector>

#include "x/cpp/errors/errors.h"
#include "x/cpp/lib/lib.h"

#include "driver/can/can.h"

namespace driver::can {
/// @brief resolves functions from a loaded vendor library and records the ones it
/// lacks, so a backend can refuse an outdated library instead of calling a null
/// pointer.
class Symbols {
    const x::lib::Shared &lib;
    std::vector<std::string> missing;

public:
    explicit Symbols(const x::lib::Shared &lib): lib(lib) {}

    /// @brief sets fn to the library's function with the given name, or to nullptr
    /// when the library lacks it.
    template<typename Fn>
    void resolve(const std::string &name, Fn &fn) {
        fn = reinterpret_cast<Fn>(const_cast<void *>(this->lib.get_func_ptr(name)));
        if (fn == nullptr) this->missing.push_back(name);
    }

    /// @returns CRITICAL_HARDWARE_ERROR naming every function resolve could not find,
    /// or NIL when it found them all.
    /// @param library the library's name for the message.
    [[nodiscard]] x::errors::Error error(const std::string &library) const {
        if (this->missing.empty()) return x::errors::NIL;
        std::string names;
        for (const auto &name: this->missing)
            names += (names.empty() ? "" : ", ") + name;
        return {
            CRITICAL_HARDWARE_ERROR,
            "the installed " + library + " library lacks " + names +
                ". Install a newer version"
        };
    }
};
}
