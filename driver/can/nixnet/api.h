// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "driver/can/nixnet/official/nixnet.h"

namespace driver::can::nixnet {
/// @brief the NI-XNET calls the backend makes. ProdAPI loads them from the vendor
/// library, and MockAPI simulates them in tests.
class API {
public:
    virtual ~API() = default;

    virtual nxStatus_t CreateSession(
        const char *database,
        const char *cluster,
        const char *list,
        const char *interface,
        u32 mode,
        nxSessionRef_t *session
    ) = 0;
    virtual nxStatus_t Clear(nxSessionRef_t session) = 0;
    virtual nxStatus_t
    SetProperty(nxSessionRef_t session, u32 id, u32 size, void *value) = 0;
    virtual nxStatus_t
    GetProperty(nxSessionRef_t session, u32 id, u32 size, void *value) = 0;
    virtual nxStatus_t GetPropertySize(nxSessionRef_t session, u32 id, u32 *size) = 0;
    virtual nxStatus_t ReadFrame(
        nxSessionRef_t session,
        void *buffer,
        u32 size,
        f64 timeout,
        u32 *returned
    ) = 0;
    virtual nxStatus_t
    WriteFrame(nxSessionRef_t session, void *buffer, u32 size, f64 timeout) = 0;
    virtual nxStatus_t Start(nxSessionRef_t session, u32 scope) = 0;
    virtual void StatusToString(nxStatus_t status, u32 size, char *description) = 0;
    virtual nxStatus_t SystemOpen(nxSessionRef_t *system) = 0;
    virtual nxStatus_t SystemClose(nxSessionRef_t system) = 0;
};
}
