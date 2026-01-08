// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#ifdef _WIN32
#include <cstdlib> // For std::mbstowcs, std::wcstombs

#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN // Exclude winsock.h from windows.h
#endif
#include <windows.h>
#else
#include <pthread.h>
#endif

namespace x::thread {
/// @brief max length for thread names on POSIX systems.
constexpr size_t MAX_NAME_LEN = 16;

/// @brief sets the name of the current thread. This name will be visible in debuggers
/// (CLion, Visual Studio, lldb, gdb) and system tools. Thread names are limited to
/// 15-16 characters on most platforms.
/// @param name the name to assign to the current thread.
inline void set_name(const char *name) {
#ifdef _WIN32
    wchar_t wname[64];
    std::mbstowcs(wname, name, 64);
    SetThreadDescription(GetCurrentThread(), wname);
#elif defined(__APPLE__)
    pthread_setname_np(name);
#else
    pthread_setname_np(pthread_self(), name);
#endif
}

/// @brief gets the name of the current thread.
/// @param buf buffer to store the thread name.
/// @param len length of the buffer.
/// @return true if the name was retrieved successfully, false otherwise.
inline bool get_name(char *buf, size_t len) {
#ifdef _WIN32
    PWSTR wname = nullptr;
    if (SUCCEEDED(GetThreadDescription(GetCurrentThread(), &wname))) {
        std::wcstombs(buf, wname, len);
        // Ensure null termination for truncated names on Windows
        if (len > 0) buf[len - 1] = '\0';
        LocalFree(wname);
        return true;
    }
    return false;
#else
    return pthread_getname_np(pthread_self(), buf, len) == 0;
#endif
}
}
