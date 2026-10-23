// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/can/can.h"
#include "driver/can/nixnet/prod.h"

namespace driver::can::nixnet {
std::pair<std::shared_ptr<API>, x::errors::Error> ProdAPI::load() {
    const auto library = nixnet::library();
    auto [lib, err] = vendor::load(library);
    if (err) return {nullptr, err};
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
    err = symbols.error(CRITICAL_HARDWARE_ERROR, library.info.name);
    if (err) return {nullptr, err};
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
