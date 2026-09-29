// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <utility>

#include "x/cpp/errors/errors.h"
#include "x/cpp/lib/lib.h"

#include "driver/can/nixnet/api.h"

namespace driver::can::nixnet {
/// @brief NI's nixnet library, loaded at runtime.
class ProdAPI final : public API {
    std::unique_ptr<x::lib::Shared> lib;
    nxCreateSession_t create_session = nullptr;
    nxClear_t clear = nullptr;
    nxSetProperty_t set_property = nullptr;
    nxGetProperty_t get_property = nullptr;
    nxGetPropertySize_t get_property_size = nullptr;
    nxReadFrame_t read_frame = nullptr;
    nxWriteFrame_t write_frame = nullptr;
    nxStart_t start = nullptr;
    nxStatusToString_t status_to_string = nullptr;
    nxSystemOpen_t system_open = nullptr;
    nxSystemClose_t system_close = nullptr;

public:
    explicit ProdAPI(std::unique_ptr<x::lib::Shared> lib): lib(std::move(lib)) {}

    /// @brief loads NI-XNET.
    /// @returns UNSUPPORTED_ERROR on macOS, the missing library error when NI-XNET is
    /// not installed, and CRITICAL_HARDWARE_ERROR when the library lacks a function the
    /// backend calls.
    static std::pair<std::shared_ptr<API>, x::errors::Error> load();

    nxStatus_t CreateSession(
        const char *database,
        const char *cluster,
        const char *list,
        const char *interface,
        u32 mode,
        nxSessionRef_t *session
    ) override;
    nxStatus_t Clear(nxSessionRef_t session) override;
    nxStatus_t
    SetProperty(nxSessionRef_t session, u32 id, u32 size, void *value) override;
    nxStatus_t
    GetProperty(nxSessionRef_t session, u32 id, u32 size, void *value) override;
    nxStatus_t GetPropertySize(nxSessionRef_t session, u32 id, u32 *size) override;
    nxStatus_t ReadFrame(
        nxSessionRef_t session,
        void *buffer,
        u32 size,
        f64 timeout,
        u32 *returned
    ) override;
    nxStatus_t
    WriteFrame(nxSessionRef_t session, void *buffer, u32 size, f64 timeout) override;
    nxStatus_t Start(nxSessionRef_t session, u32 scope) override;
    void StatusToString(nxStatus_t status, u32 size, char *description) override;
    nxStatus_t SystemOpen(nxSessionRef_t *system) override;
    nxStatus_t SystemClose(nxSessionRef_t system) override;
};
}
