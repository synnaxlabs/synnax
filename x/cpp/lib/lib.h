// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

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
/// SharedLib is a shared library loader and lifecycle manager implemented for
/// Windows.
class SharedLib {
    const std::string name;
    LibraryHandle handle = nullptr;

public:
    explicit SharedLib(const std::string &name): name(name) {}

    ~SharedLib() { this->unload(); }

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
class SharedLib {
    const std::string name;
    LibraryHandle handle = nullptr;

public:
    explicit SharedLib(std::string name): name(std::move(name)) {}

    ~SharedLib() { this->unload(); }

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

    [[nodiscard]] const void *get_func_ptr(const std::string &name) const {
        if (this->handle == nullptr) return nullptr;
        return ::dlsym(this->handle, name.c_str());
    }
};
#endif
}
