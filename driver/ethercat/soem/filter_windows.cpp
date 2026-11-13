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
/// @brief how long the user has to approve the change, and Windows to make it.
constexpr DWORD TIMEOUT_MS = 45000;
/// @brief the exit code of the script when Windows does not give administrator access.
constexpr DWORD NOT_APPROVED = ERROR_CANCELLED;
/// @brief the exit code of the script when Windows has no adapter for an interface.
constexpr DWORD NO_ADAPTER = 2;

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

/// @brief returns a script that turns off the filter on the adapter of each GUID, and
/// exits with NO_ADAPTER before it changes anything when Windows lacks one.
std::string disable_script(const std::vector<std::string> &guids) {
    std::string list;
    for (const auto &guid: guids)
        list += (list.empty() ? "'" : ",'") + guid + "'";
    return std::format(
        "$ErrorActionPreference = 'Stop'; $guids = @({}); "
        "$adapters = @(Get-NetAdapter | Where-Object {{ $guids -contains "
        "$_.InterfaceGuid }}); "
        "if ($adapters.Count -ne $guids.Count) {{ exit {} }}; "
        "foreach ($a in $adapters) {{ Disable-NetAdapterBinding -Name $a.Name "
        "-ComponentID DW_EthcatDAQ }}; exit 0",
        list,
        NO_ADAPTER
    );
}

/// @brief returns a script that runs script with administrator access, and exits with
/// its exit code, or with NOT_APPROVED when Windows does not give the access.
std::string elevate_script(const std::string &script) {
    return std::format(
        "try {{ $p = Start-Process -FilePath (Join-Path $PSHOME 'powershell.exe') "
        "-Verb RunAs -Wait -PassThru -WindowStyle Hidden -ArgumentList "
        "'-NoProfile -NonInteractive -EncodedCommand {}'; exit $p.ExitCode }} "
        "catch {{ exit {} }}",
        encode(script),
        NOT_APPROVED
    );
}

std::optional<std::wstring> powershell_path() {
    wchar_t dir[MAX_PATH];
    const UINT len = GetSystemDirectoryW(dir, MAX_PATH);
    if (len == 0 || len >= MAX_PATH) return std::nullopt;
    return std::wstring(dir, len) + L"\\WindowsPowerShell\\v1.0\\powershell.exe";
}

x::errors::Error run(const std::string &script) {
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
    DWORD code = 0;
    const bool exited = WaitForSingleObject(process.hProcess, TIMEOUT_MS) ==
                            WAIT_OBJECT_0 &&
                        GetExitCodeProcess(process.hProcess, &code) != 0;
    CloseHandle(process.hProcess);
    if (!exited)
        return x::errors::Error(
            errors::FILTER_ERROR,
            "Windows did not turn off the filter in 45 seconds. If Windows asks for "
            "administrator access, approve it, and Synnax finds the modules at a later "
            "scan."
        );
    if (code == 0) return x::errors::NIL;
    if (code == NOT_APPROVED)
        return x::errors::Error(
            errors::FILTER_ERROR,
            "Windows did not give the Driver administrator access. " + MANUAL_FIX
        );
    if (code == NO_ADAPTER)
        return x::errors::Error(
            errors::FILTER_ERROR,
            "Windows has no network adapter for the interface. " + MANUAL_FIX
        );
    return x::errors::Error(
        errors::FILTER_ERROR,
        std::format("Windows could not turn off the filter (exit code {}). ", code) +
            MANUAL_FIX
    );
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
    return run(elevated() ? script : elevate_script(script));
}
}
