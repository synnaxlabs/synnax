// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>

#include "driver/can/can.h"
#include "driver/can/nixnet/prod.h"
#include "driver/errors/errors.h"

namespace driver::can::nixnet {
namespace {
const LibraryInfo LIBRARY_INFO = {
    "NI-XNET",
    "https://www.ni.com/en/support/downloads/drivers/download.ni-xnet.html"
};

#ifdef _WIN32
const std::string LIBRARY_NAME = "nixnet.dll";
#elif defined(__linux__)
const std::string LIBRARY_NAME = "libnixnet.so";
#else
const std::string LIBRARY_NAME;
#endif
}

std::pair<std::shared_ptr<API>, x::errors::Error> ProdAPI::load() {
    if (LIBRARY_NAME.empty())
        return {
            nullptr,
            x::errors::Error(UNSUPPORTED_ERROR, "NI-XNET runs on Windows and Linux")
        };
    auto lib = std::make_unique<x::lib::Shared>(LIBRARY_NAME);
    if (!lib->load()) return {nullptr, driver::errors::missing_lib(LIBRARY_INFO)};
    auto api = std::make_shared<ProdAPI>(std::move(lib));
    x::lib::Symbols symbols(*api->lib);
    symbols.resolve("nxCreateSession", api->create_session);
    symbols.resolve("nxClear", api->clear);
    symbols.resolve("nxSetProperty", api->set_property);
    symbols.resolve("nxGetProperty", api->get_property);
    symbols.resolve("nxGetPropertySize", api->get_property_size);
    symbols.resolve("nxReadFrame", api->read_frame);
    symbols.resolve("nxWriteFrame", api->write_frame);
    symbols.resolve("nxStart", api->start);
    symbols.resolve("nxStatusToString", api->status_to_string);
    symbols.resolve("nxSystemOpen", api->system_open);
    symbols.resolve("nxSystemClose", api->system_close);
    if (auto err = symbols.error(CRITICAL_HARDWARE_ERROR, LIBRARY_INFO.name))
        return {nullptr, err};
    return {api, x::errors::NIL};
}

nxStatus_t ProdAPI::CreateSession(
    const char *database,
    const char *cluster,
    const char *list,
    const char *interface,
    const u32 mode,
    nxSessionRef_t *session
) {
    return this->create_session(database, cluster, list, interface, mode, session);
}

nxStatus_t ProdAPI::Clear(const nxSessionRef_t session) {
    return this->clear(session);
}

nxStatus_t ProdAPI::SetProperty(
    const nxSessionRef_t session,
    const u32 id,
    const u32 size,
    void *value
) {
    return this->set_property(session, id, size, value);
}

nxStatus_t ProdAPI::GetProperty(
    const nxSessionRef_t session,
    const u32 id,
    const u32 size,
    void *value
) {
    return this->get_property(session, id, size, value);
}

nxStatus_t
ProdAPI::GetPropertySize(const nxSessionRef_t session, const u32 id, u32 *size) {
    return this->get_property_size(session, id, size);
}

nxStatus_t ProdAPI::ReadFrame(
    const nxSessionRef_t session,
    void *buffer,
    const u32 size,
    const f64 timeout,
    u32 *returned
) {
    return this->read_frame(session, buffer, size, timeout, returned);
}

nxStatus_t ProdAPI::WriteFrame(
    const nxSessionRef_t session,
    void *buffer,
    const u32 size,
    const f64 timeout
) {
    return this->write_frame(session, buffer, size, timeout);
}

nxStatus_t ProdAPI::Start(const nxSessionRef_t session, const u32 scope) {
    return this->start(session, scope);
}

void ProdAPI::StatusToString(
    const nxStatus_t status,
    const u32 size,
    char *description
) {
    this->status_to_string(status, size, description);
}

nxStatus_t ProdAPI::SystemOpen(nxSessionRef_t *system) {
    return this->system_open(system);
}

nxStatus_t ProdAPI::SystemClose(const nxSessionRef_t system) {
    return this->system_close(system);
}
}
