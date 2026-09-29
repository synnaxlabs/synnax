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
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"

#ifdef _WIN32
#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#include <windows.h>
typedef HMODULE LibraryHandle;
#else
#include <dlfcn.h>
typedef void *LibraryHandle;
#endif

namespace x::lib {
const errors::Error BASE_ERROR = errors::SY.sub("shared");
const errors::Error LOAD_ERROR = BASE_ERROR.sub("load");

#ifdef _WIN32
/// Shared is a shared library loader and lifecycle manager implemented for
/// Windows.
class Shared {
    const std::string name;
    LibraryHandle handle = nullptr;

public:
    explicit Shared(const std::string &name): name(name) {}

    ~Shared() { this->unload(); }

    bool load() {
        if (this->handle != nullptr || name.empty()) return false;
        this->handle = ::LoadLibraryA(name.c_str());
        return this->handle != nullptr;
    }

    void unload() {
        if (this->handle == nullptr) return;
        ::FreeLibrary(this->handle);
    }

    const void *get_func_ptr(const std::string &name) const {
        if (this->handle == nullptr) return nullptr;
        return ::GetProcAddress(this->handle, name.c_str());
    }
};
#else

/// Lib is a shared library loader and lifecycle manager implemented for POSIX
/// compliant systems.
class Shared {
    const std::string name;
    LibraryHandle handle = nullptr;

public:
    explicit Shared(std::string name): name(std::move(name)) {}

    ~Shared() { this->unload(); }

    bool load() {
        if (this->handle != nullptr || name.empty()) return false;
        this->handle = ::dlopen(name.c_str(), RTLD_NOW | RTLD_GLOBAL);
        // Don't log error if specific hardware driver libraries are
        // not installed. Downstream code handles this gracefully.
        return this->handle != nullptr;
    }

    void unload() {
        if (this->handle == nullptr) return;
        ::dlclose(this->handle);
        this->handle = nullptr;
    }

    [[nodiscard]] const void *get_func_ptr(const std::string &lib_name) const {
        if (this->handle == nullptr) return nullptr;
        return ::dlsym(this->handle, lib_name.c_str());
    }
};
#endif

/// @brief resolves functions from a loaded library and records the ones it lacks, so a
/// caller can refuse an outdated library instead of calling a null pointer.
class Symbols {
    const Shared &lib;
    std::vector<std::string> missing;

public:
    explicit Symbols(const Shared &lib): lib(lib) {}

    /// @brief sets fn to the library's function with the given name, or to nullptr when
    /// the library lacks it.
    template<typename Fn>
    void resolve(const std::string &name, Fn &fn) {
        fn = reinterpret_cast<Fn>(const_cast<void *>(this->lib.get_func_ptr(name)));
        if (fn == nullptr) this->missing.push_back(name);
    }

    /// @returns an error of the given type naming every function resolve could not
    /// find, or NIL when it found them all.
    /// @param type the type of the returned error.
    /// @param library the library's name for the message.
    [[nodiscard]] errors::Error
    error(const errors::Error &type, const std::string &library) const {
        if (this->missing.empty()) return errors::NIL;
        std::string names;
        for (const auto &name: this->missing)
            names += (names.empty() ? "" : ", ") + name;
        return {
            type,
            "the installed " + library + " library lacks " + names +
                ". Install a newer version"
        };
    }
};
}
