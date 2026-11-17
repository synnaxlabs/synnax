// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <string>
#include <system_error>

#include <windows.h>

#include "driver/journal/file.h"
#include "driver/journal/journal.h"

namespace driver::journal {
namespace {
x::errors::Error last_error(const std::string &msg) {
    const auto code = static_cast<int>(GetLastError());
    return x::errors::Error(ERROR, msg + ": " + std::system_category().message(code));
}

HANDLE to_handle(const std::intptr_t handle) {
    return reinterpret_cast<HANDLE>(handle);
}
}

File::File(const std::intptr_t handle): handle(handle) {}

File::~File() {
    CloseHandle(to_handle(this->handle));
}

std::pair<std::unique_ptr<File>, x::errors::Error>
File::open(const std::filesystem::path &path) {
    // FILE_SHARE_DELETE lets other handles rename or remove the file while it is open.
    const HANDLE h = CreateFileW(
        path.c_str(),
        GENERIC_WRITE,
        FILE_SHARE_READ | FILE_SHARE_WRITE | FILE_SHARE_DELETE,
        nullptr,
        OPEN_ALWAYS,
        FILE_ATTRIBUTE_NORMAL,
        nullptr
    );
    if (h == INVALID_HANDLE_VALUE)
        return {nullptr, last_error("failed to open " + path.string())};
    if (!SetFilePointerEx(h, LARGE_INTEGER{}, nullptr, FILE_END)) {
        auto err = last_error("failed to seek " + path.string());
        CloseHandle(h);
        return {nullptr, err};
    }
    return {
        std::unique_ptr<File>(new File(reinterpret_cast<std::intptr_t>(h))),
        x::errors::NIL
    };
}

x::errors::Error File::write(std::string_view data) {
    while (!data.empty()) {
        DWORD n = 0;
        const auto len = static_cast<DWORD>(
            std::min<std::size_t>(data.size(), MAXDWORD)
        );
        if (!WriteFile(to_handle(this->handle), data.data(), len, &n, nullptr))
            return last_error("failed to write");
        data.remove_prefix(n);
    }
    return x::errors::NIL;
}

x::errors::Error File::sync() {
    if (!FlushFileBuffers(to_handle(this->handle))) return last_error("failed to sync");
    return x::errors::NIL;
}

x::errors::Error File::lock() {
    OVERLAPPED overlapped{};
    if (LockFileEx(
            to_handle(this->handle),
            LOCKFILE_EXCLUSIVE_LOCK | LOCKFILE_FAIL_IMMEDIATELY,
            0,
            MAXDWORD,
            MAXDWORD,
            &overlapped
        ))
        return x::errors::NIL;
    if (GetLastError() == ERROR_LOCK_VIOLATION) return LOCKED;
    return last_error("failed to lock");
}

/// @brief Windows has no directory sync. replace writes through instead.
x::errors::Error sync_dir(const std::filesystem::path &) {
    return x::errors::NIL;
}

x::errors::Error
replace(const std::filesystem::path &from, const std::filesystem::path &to) {
    if (!MoveFileExW(
            from.c_str(),
            to.c_str(),
            MOVEFILE_REPLACE_EXISTING | MOVEFILE_WRITE_THROUGH
        ))
        return last_error("failed to rename " + from.string());
    return x::errors::NIL;
}
}
