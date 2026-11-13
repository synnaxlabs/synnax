// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif

#include <format>
#include <optional>
#include <string>
#include <vector>

#include <windows.h>

#include "x/cpp/base64/base64.h"

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/soem/filter.h"
#include "driver/ethercat/soem/util.h"

namespace driver::ethercat::soem {
namespace {
const std::string
    MANUAL_FIX = "To turn it off by hand, clear \"EthercatDAQ Filter\" in "
                 "the properties of the adapter.";

bool elevated() {
    HANDLE token = nullptr;
    if (OpenProcessToken(GetCurrentProcess(), TOKEN_QUERY, &token) == 0) return false;
    TOKEN_ELEVATION elevation{};
    DWORD size = 0;
    const bool read = GetTokenInformation(
                          token,
                          TokenElevation,
                          &elevation,
                          sizeof(elevation),
                          &size
                      ) != 0;
    CloseHandle(token);
    return read && elevation.TokenIsElevated != 0;
}

/// @brief returns the ASCII script in the form that powershell -EncodedCommand takes:
/// the base64 of its UTF-16LE bytes.
std::string encode(const std::string &script) {
    std::string utf16;
    for (const char c: script) {
        utf16.push_back(c);
        utf16.push_back('\0');
    }
    return x::base64::encode(utf16);
}

/// @brief returns a script that turns off the filter on the adapter of each GUID.
std::string disable_script(const std::vector<std::string> &guids) {
    std::string list;
    for (const auto &guid: guids)
        list += (list.empty() ? "'" : ",'") + guid + "'";
    return std::format(
        "$guids = @({}); Get-NetAdapter | Where-Object {{ $guids -contains "
        "$_.InterfaceGuid }} | ForEach-Object {{ Disable-NetAdapterBinding -Name "
        "$_.Name -ComponentID DW_EthcatDAQ }}",
        list
    );
}

/// @brief returns a script that runs script with administrator access.
std::string elevate_script(const std::string &script) {
    return std::format(
        "Start-Process -FilePath (Join-Path $PSHOME 'powershell.exe') -Verb RunAs "
        "-WindowStyle Hidden -ArgumentList '-NoProfile -NonInteractive "
        "-EncodedCommand {}'",
        encode(script)
    );
}

std::optional<std::wstring> powershell_path() {
    wchar_t dir[MAX_PATH];
    const UINT len = GetSystemDirectoryW(dir, MAX_PATH);
    if (len == 0 || len >= MAX_PATH) return std::nullopt;
    return std::wstring(dir, len) + L"\\WindowsPowerShell\\v1.0\\powershell.exe";
}

/// @brief starts PowerShell on script, and returns without waiting for it to exit.
x::errors::Error start(const std::string &script) {
    const auto path = powershell_path();
    if (!path)
        return x::errors::Error(
            errors::FILTER_ERROR,
            "Windows did not give the path of PowerShell. " + MANUAL_FIX
        );
    const auto encoded = encode(script);
    std::wstring command = L"\"" + *path +
                           L"\" -NoProfile -NonInteractive -EncodedCommand " +
                           std::wstring(encoded.begin(), encoded.end());
    STARTUPINFOW startup{};
    startup.cb = sizeof(startup);
    PROCESS_INFORMATION process{};
    if (CreateProcessW(
            path->c_str(),
            command.data(),
            nullptr,
            nullptr,
            FALSE,
            CREATE_NO_WINDOW,
            nullptr,
            nullptr,
            &startup,
            &process
        ) == 0)
        return x::errors::Error(
            errors::FILTER_ERROR,
            std::format(
                "Windows could not start PowerShell (error {}). ",
                GetLastError()
            ) + MANUAL_FIX
        );
    CloseHandle(process.hThread);
    CloseHandle(process.hProcess);
    return x::errors::NIL;
}
}

x::errors::Error disable_dewesoft_filter(const std::vector<std::string> &interfaces) {
    std::vector<std::string> guids;
    for (const auto &iface: interfaces) {
        const auto guid = adapter_guid(iface);
        if (!guid)
            return x::errors::Error(
                errors::FILTER_ERROR,
                iface + " is not a Windows network adapter."
            );
        guids.push_back(*guid);
    }
    const auto script = disable_script(guids);
    return start(elevated() ? script : elevate_script(script));
}
}
