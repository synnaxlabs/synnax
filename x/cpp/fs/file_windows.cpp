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

#include "x/cpp/fs/file.h"

namespace x::fs {
namespace {
x::errors::Error last_error(const std::string &msg) {
    const auto code = static_cast<int>(GetLastError());
    return x::errors::Error(
        FS_ERROR,
        msg + ": " + std::system_category().message(code)
    );
}

HANDLE to_handle(const std::intptr_t handle) {
    return reinterpret_cast<HANDLE>(handle);
}

/// @brief the lock covers one byte far past the data, so it does not block reads.
OVERLAPPED lock_range() {
    OVERLAPPED overlapped{};
    overlapped.Offset = MAXDWORD;
    overlapped.OffsetHigh = 0x7FFFFFFF;
    return overlapped;
}

bool seek_end(const HANDLE h) {
    return SetFilePointerEx(h, LARGE_INTEGER{}, nullptr, FILE_END) != 0;
}
}

File::File(const std::intptr_t handle): handle(handle) {}

File::~File() {
    // Windows releases the lock of a closed handle only after a delay.
    auto overlapped = lock_range();
    UnlockFileEx(to_handle(this->handle), 0, 1, 0, &overlapped);
    CloseHandle(to_handle(this->handle));
}

std::pair<std::unique_ptr<File>, x::errors::Error>
File::open(const std::filesystem::path &path) {
    // FILE_SHARE_DELETE lets other handles rename or remove the file while it is open.
    const HANDLE h = CreateFileW(
        path.c_str(),
        GENERIC_WRITE | FILE_READ_ATTRIBUTES,
        FILE_SHARE_READ | FILE_SHARE_WRITE | FILE_SHARE_DELETE,
        nullptr,
        OPEN_ALWAYS,
        FILE_ATTRIBUTE_NORMAL,
        nullptr
    );
    if (h == INVALID_HANDLE_VALUE)
        return {nullptr, last_error("failed to open " + path.string())};
    if (!seek_end(h)) {
        auto err = last_error("failed to seek " + path.string());
        CloseHandle(h);
        return {nullptr, err};
    }
    return {
        std::unique_ptr<File>(new File(reinterpret_cast<std::intptr_t>(h))),
        x::errors::NIL
    };
}

std::pair<std::unique_ptr<File>, x::errors::Error>
File::open_read(const std::filesystem::path &path) {
    const HANDLE h = CreateFileW(
        path.c_str(),
        GENERIC_READ,
        FILE_SHARE_READ | FILE_SHARE_WRITE | FILE_SHARE_DELETE,
        nullptr,
        OPEN_EXISTING,
        FILE_ATTRIBUTE_NORMAL,
        nullptr
    );
    if (h == INVALID_HANDLE_VALUE)
        return {nullptr, last_error("failed to open " + path.string())};
    return {
        std::unique_ptr<File>(new File(reinterpret_cast<std::intptr_t>(h))),
        x::errors::NIL
    };
}

std::pair<std::size_t, x::errors::Error>
File::read(const std::uint64_t pos, char *dst, const std::size_t size) const {
    std::size_t done = 0;
    while (done < size) {
        OVERLAPPED overlapped{};
        const auto at = pos + done;
        overlapped.Offset = static_cast<DWORD>(at & 0xFFFFFFFF);
        overlapped.OffsetHigh = static_cast<DWORD>(at >> 32);
        const auto len = static_cast<DWORD>(
            std::min<std::size_t>(size - done, MAXDWORD)
        );
        DWORD n = 0;
        if (!ReadFile(to_handle(this->handle), dst + done, len, &n, &overlapped)) {
            if (GetLastError() == ERROR_HANDLE_EOF) break;
            return {done, last_error("failed to read")};
        }
        if (n == 0) break;
        done += n;
    }
    return {done, x::errors::NIL};
}

std::pair<Info, x::errors::Error> File::stat() const {
    FILE_STANDARD_INFO info{};
    if (!GetFileInformationByHandleEx(
            to_handle(this->handle),
            FileStandardInfo,
            &info,
            sizeof(info)
        ))
        return {{}, last_error("failed to stat")};
    return {
        Info{
            .size = static_cast<std::uint64_t>(info.EndOfFile.QuadPart),
            .removed = info.NumberOfLinks == 0 || info.DeletePending != FALSE,
        },
        x::errors::NIL
    };
}

x::errors::Error File::write(std::string_view data) const {
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

x::errors::Error File::truncate(const std::uint64_t size) const {
    FILE_END_OF_FILE_INFO info{};
    info.EndOfFile.QuadPart = static_cast<LONGLONG>(size);
    const auto h = to_handle(this->handle);
    if (!SetFileInformationByHandle(h, FileEndOfFileInfo, &info, sizeof(info)) ||
        !seek_end(h))
        return last_error("failed to truncate");
    return x::errors::NIL;
}

x::errors::Error File::sync() const {
    if (!FlushFileBuffers(to_handle(this->handle))) return last_error("failed to sync");
    return x::errors::NIL;
}

x::errors::Error File::lock() const {
    auto overlapped = lock_range();
    if (LockFileEx(
            to_handle(this->handle),
            LOCKFILE_EXCLUSIVE_LOCK | LOCKFILE_FAIL_IMMEDIATELY,
            0,
            1,
            0,
            &overlapped
        ))
        return x::errors::NIL;
    if (GetLastError() == ERROR_LOCK_VIOLATION) return LOCKED;
    return last_error("failed to lock");
}

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
