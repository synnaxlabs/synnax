// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cstddef>
#include <initializer_list>
#include <span>
#include <string>
#include <system_error>
#include <utility>

#include "asio/error.hpp"
#include <windows.h>

#include "driver/serial/errors.h"
#include "driver/serial/line.h"
#include "driver/serial/native.h"
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
    // Asio reports its own codes, such as a closed port, in its own category.
    if (reported == asio::error::eof || reported == asio::error::bad_descriptor)
        return transport::error(transport::UNREACHABLE_ERROR, context, reported);
    const auto ec = transport::from_asio(reported);
    // A USB adapter that goes away fails its open handle with access denied.
    if (ec == std::errc::timed_out || win32(
                                          ec,
                                          {ERROR_FILE_NOT_FOUND,
                                           ERROR_PATH_NOT_FOUND,
                                           ERROR_ACCESS_DENIED,
                                           ERROR_SHARING_VIOLATION,
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

x::errors::Error translate_open(const std::string &port, const std::error_code &ec) {
    const auto context = "failed to open serial port " + port;
    // Windows opens a port for one program at a time and denies access to the rest.
    if (win32(transport::from_asio(ec), {ERROR_ACCESS_DENIED, ERROR_SHARING_VIOLATION}))
        return x::errors::Error(
            transport::UNREACHABLE_ERROR,
            context + ": the port is in use by another program"
        );
    return translate(context, ec);
}

namespace line {
x::errors::Error apply(const synnax::serial::Properties &props, Settings &settings) {
    if (auto err = check(props)) return err;
    const bool five = props.data_bits == 5;
    if (props.stop_bits == synnax::serial::STOP_BITS_ONE_AND_HALF && !five)
        return {transport::CONFIG_ERROR, "Windows needs 5 data bits for 1.5 stop bits"};
    if (props.stop_bits == synnax::serial::STOP_BITS_TWO && five)
        return {
            transport::CONFIG_ERROR,
            "Windows does not support 2 stop bits with 5 data bits"
        };
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

void discard_output(const Handle handle) {
    ::PurgeComm(reinterpret_cast<HANDLE>(handle), PURGE_TXABORT | PURGE_TXCLEAR);
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
        return {{.size = data.size()}, last_error()};
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
}
