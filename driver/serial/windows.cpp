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

#include <windows.h>

#include "driver/serial/native.h"
#include "driver/serial/scan.h"

namespace driver::serial {
namespace native {
namespace {
std::error_code last_error() {
    return {static_cast<int>(::GetLastError()), std::system_category()};
}

template<typename Edit>
std::error_code edit_dcb(const Handle handle, Edit &&edit) {
    const auto h = reinterpret_cast<HANDLE>(handle);
    DCB dcb{};
    dcb.DCBlength = sizeof(DCB);
    if (!::GetCommState(h, &dcb)) return last_error();
    edit(dcb);
    if (!::SetCommState(h, &dcb)) return last_error();
    return {};
}
}

std::error_code set_flow_control(const Handle handle, const FlowControl mode) {
    return edit_dcb(handle, [mode](DCB &dcb) {
        dcb.fOutxDsrFlow = FALSE;
        dcb.fDsrSensitivity = FALSE;
        dcb.fDtrControl = DTR_CONTROL_ENABLE;
        dcb.fTXContinueOnXoff = TRUE;
        dcb.fOutX = mode == FlowControl::SOFTWARE;
        dcb.fInX = mode == FlowControl::SOFTWARE;
        dcb.fOutxCtsFlow = mode == FlowControl::HARDWARE;
        dcb.fRtsControl = mode == FlowControl::HARDWARE ? RTS_CONTROL_HANDSHAKE
                                                        : RTS_CONTROL_ENABLE;
    });
}

std::error_code set_baud_rate(const Handle handle, const std::uint32_t rate) {
    return edit_dcb(handle, [rate](DCB &dcb) { dcb.BaudRate = rate; });
}

std::error_code set_mark_space_parity(const Handle handle, const bool mark) {
    return edit_dcb(handle, [mark](DCB &dcb) {
        dcb.fParity = TRUE;
        dcb.Parity = mark ? MARKPARITY : SPACEPARITY;
    });
}

std::error_code enable_rs485(const Handle handle) {
    return edit_dcb(handle, [](DCB &dcb) { dcb.fRtsControl = RTS_CONTROL_TOGGLE; });
}

std::pair<std::size_t, std::error_code> queued_output(const Handle handle) {
    DWORD errors = 0;
    COMSTAT stat{};
    if (!::ClearCommError(reinterpret_cast<HANDLE>(handle), &errors, &stat))
        return {0, last_error()};
    return {stat.cbOutQue, {}};
}
}

namespace {
/// @brief orders COM2 before COM10.
bool natural_less(const Info &a, const Info &b) {
    if (a.path.size() != b.path.size()) return a.path.size() < b.path.size();
    return a.path < b.path;
}
}

std::pair<std::vector<Info>, x::errors::Error> scan(const std::filesystem::path &) {
    HKEY key;
    const auto opened = ::RegOpenKeyExA(
        HKEY_LOCAL_MACHINE,
        "HARDWARE\\DEVICEMAP\\SERIALCOMM",
        0,
        KEY_READ,
        &key
    );
    if (opened == ERROR_FILE_NOT_FOUND) return {{}, x::errors::NIL};
    if (opened != ERROR_SUCCESS)
        return {
            {},
            x::errors::Error(
                SCAN_ERROR,
                "failed to open the SERIALCOMM registry key: " +
                    std::system_category().message(static_cast<int>(opened))
            )
        };
    std::vector<Info> ports;
    for (DWORD i = 0;; i++) {
        char name[256];
        DWORD name_size = sizeof(name);
        BYTE data[256];
        DWORD data_size = sizeof(data);
        DWORD type = 0;
        const auto status = ::RegEnumValueA(
            key,
            i,
            name,
            &name_size,
            nullptr,
            &type,
            data,
            &data_size
        );
        if (status == ERROR_NO_MORE_ITEMS) break;
        if (status != ERROR_SUCCESS) {
            ::RegCloseKey(key);
            return {
                {},
                x::errors::Error(
                    SCAN_ERROR,
                    "failed to read the SERIALCOMM registry key: " +
                        std::system_category().message(static_cast<int>(status))
                )
            };
        }
        if (type != REG_SZ) continue;
        std::string path(reinterpret_cast<const char *>(data), data_size);
        path.erase(path.find_last_not_of('\0') + 1);
        ports.push_back({.path = path, .name = std::string(name, name_size)});
    }
    ::RegCloseKey(key);
    std::ranges::sort(ports, natural_less);
    return {ports, x::errors::NIL};
}
}
