// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <initializer_list>
#include <string>

#include "asio/error.hpp"
#include <windows.h>

#include "driver/serial/errors.h"
#include "driver/serial/line.h"
#include "driver/serial/native.h"
#include "driver/serial/scan.h"
#include "driver/transport/errors.h"

namespace driver::serial {
namespace {
bool win32(const std::error_code &ec, const std::initializer_list<DWORD> codes) {
    if (ec.category() != std::system_category()) return false;
    return std::ranges::any_of(codes, [&ec](const DWORD code) {
        return ec.value() == static_cast<int>(code);
    });
}

std::error_code last_error() {
    return {static_cast<int>(::GetLastError()), std::system_category()};
}
}

x::errors::Error
translate(const std::string &context, const std::error_code &reported) {
    const auto ec = transport::from_asio(reported);
    // Windows opens a port for one program at a time and denies access to the rest.
    if (win32(ec, {ERROR_ACCESS_DENIED, ERROR_SHARING_VIOLATION}))
        return x::errors::Error(
            transport::UNREACHABLE_ERROR,
            context + ": the port is in use by another program"
        );
    if (ec == asio::error::eof || ec == asio::error::bad_descriptor ||
        ec == std::errc::timed_out ||
        win32(
            ec,
            {ERROR_FILE_NOT_FOUND,
             ERROR_PATH_NOT_FOUND,
             ERROR_INVALID_HANDLE,
             ERROR_BAD_COMMAND,
             ERROR_GEN_FAILURE,
             ERROR_DEV_NOT_EXIST,
             ERROR_SEM_TIMEOUT,
             ERROR_NO_SUCH_DEVICE,
             ERROR_OPERATION_ABORTED,
             ERROR_IO_DEVICE,
             ERROR_COUNTER_TIMEOUT,
             ERROR_DEVICE_NOT_CONNECTED,
             ERROR_DEVICE_REMOVED}
        ))
        return transport::error(transport::UNREACHABLE_ERROR, context, ec);
    if (win32(
            ec,
            {ERROR_INVALID_FUNCTION,
             ERROR_NOT_SUPPORTED,
             ERROR_INVALID_PARAMETER,
             ERROR_INVALID_NAME,
             ERROR_BAD_PATHNAME}
        ))
        return transport::error(transport::CONFIG_ERROR, context, ec);
    return transport::error(transport::CRITICAL_ERROR, context, ec);
}

namespace line {
x::errors::Error apply(const synnax::serial::Properties &props, Settings &settings) {
    settings.BaudRate = props.baud_rate;
    settings.ByteSize = props.data_bits;
    const auto &parity = props.parity;
    settings.fParity = parity != synnax::serial::PARITY_NONE_;
    if (parity == synnax::serial::PARITY_NONE_) settings.Parity = NOPARITY;
    if (parity == synnax::serial::PARITY_EVEN_) settings.Parity = EVENPARITY;
    if (parity == synnax::serial::PARITY_ODD_) settings.Parity = ODDPARITY;
    if (parity == synnax::serial::PARITY_MARK_) settings.Parity = MARKPARITY;
    if (parity == synnax::serial::PARITY_SPACE_) settings.Parity = SPACEPARITY;
    if (props.stop_bits == synnax::serial::STOP_BITS_ONE)
        settings.StopBits = ONESTOPBIT;
    if (props.stop_bits == synnax::serial::STOP_BITS_ONE_AND_HALF)
        settings.StopBits = ONE5STOPBITS;
    if (props.stop_bits == synnax::serial::STOP_BITS_TWO)
        settings.StopBits = TWOSTOPBITS;
    const auto &flow = props.flow_control;
    settings.fOutxDsrFlow = FALSE;
    settings.fDsrSensitivity = FALSE;
    settings.fDtrControl = DTR_CONTROL_ENABLE;
    settings.fTXContinueOnXoff = TRUE;
    settings.fOutX = flow == synnax::serial::FLOW_CONTROL_SOFTWARE;
    settings.fInX = flow == synnax::serial::FLOW_CONTROL_SOFTWARE;
    settings.fOutxCtsFlow = flow == synnax::serial::FLOW_CONTROL_HARDWARE;
    settings.fRtsControl = flow == synnax::serial::FLOW_CONTROL_HARDWARE
                             ? RTS_CONTROL_HANDSHAKE
                             : RTS_CONTROL_ENABLE;
    if (props.rs485) settings.fRtsControl = RTS_CONTROL_TOGGLE;
    return x::errors::NIL;
}
}

namespace native {
namespace {
x::errors::Error failed(const std::string &port) {
    return translate("failed to set the line settings on " + port, last_error());
}
}

// CreateFile opens the port with no sharing, so Windows already keeps other programs
// out.
x::errors::Error lock(Handle, const std::string &) {
    return x::errors::NIL;
}

void unlock(Handle) {}

x::errors::Error
configure(const Handle handle, const synnax::serial::Properties &props) {
    const auto h = reinterpret_cast<HANDLE>(handle);
    line::Settings settings{};
    settings.DCBlength = sizeof(DCB);
    if (!::GetCommState(h, &settings)) return failed(props.port);
    if (auto err = line::apply(props, settings)) return err;
    if (!::SetCommState(h, &settings)) return failed(props.port);
    return x::errors::NIL;
}

std::pair<std::size_t, std::error_code> queued_output(const Handle handle) {
    DWORD errors = 0;
    COMSTAT stat{};
    if (!::ClearCommError(reinterpret_cast<HANDLE>(handle), &errors, &stat))
        return {0, last_error()};
    return {stat.cbOutQue, {}};
}

std::error_code watch(const Handle handle, Line &) {
    DWORD errors = 0;
    if (!::ClearCommError(reinterpret_cast<HANDLE>(handle), &errors, nullptr))
        return last_error();
    return {};
}

// Windows reports which line errors occurred since the last call, not which bytes they
// hit, so each reported error counts once and the bytes stay in data.
std::pair<Received, std::error_code>
receive(const Handle handle, Line &, const std::span<std::uint8_t> data) {
    DWORD errors = 0;
    if (!::ClearCommError(reinterpret_cast<HANDLE>(handle), &errors, nullptr))
        return {{}, last_error()};
    const auto count = [errors](const std::initializer_list<DWORD> flags) {
        return static_cast<std::size_t>(std::ranges::count_if(flags, [errors](DWORD f) {
            return (errors & f) != 0;
        }));
    };
    return {
        {.size = data.size(),
         .errors = count({CE_RXPARITY, CE_FRAME, CE_BREAK}),
         .overruns = count({CE_OVERRUN, CE_RXOVER})},
        {}
    };
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
