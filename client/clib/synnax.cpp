// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <exception>
#include <string>

#include "client/clib/internal.h"
#include "client/clib/synnax.h"

using namespace synnax::clib;

int32_t synnax_client_open(
    const char *host,
    const uint16_t port,
    const char *username,
    const char *password,
    const int32_t secure,
    const char *ca_cert_file,
    SynnaxClient **out_client,
    SynnaxError *err
) {
    clear_err(err);
    if (out_client == nullptr) {
        set_err(err, CODE_INTERNAL, "sy.validation", "null out_client");
        return CODE_INTERNAL;
    }
    try {
        synnax::Config c;
        c.host = str_or(host, "localhost");
        if (port != 0) c.port = port;
        c.username = str_or(username, "synnax");
        c.password = str_or(password, "seldon");
        if (secure != 0) {
            const std::string ca = str_or(ca_cert_file, "");
            if (ca.empty()) {
                set_err(
                    err,
                    CODE_INTERNAL,
                    "sy.validation",
                    "secure connection requires ca_cert_file; system-trust TLS is "
                    "not supported"
                );
                return CODE_INTERNAL;
            }
            c.ca_cert_file = ca;
        }

        auto *wrapper = new SynnaxClient(c);
        const auto state = wrapper->client.connectivity->check();
        if (state.status != synnax::connection::Status::CONNECTED) {
            const bool has_err = !state.error.ok();
            set_err(
                err,
                CODE_ERROR,
                has_err ? state.error.type : std::string("sy.connection"),
                has_err ? state.error.data : state.message
            );
            delete wrapper;
            return CODE_ERROR;
        }
        *out_client = wrapper;
        return CODE_OK;
    } catch (const std::exception &e) {
        set_err(err, CODE_INTERNAL, "sy.internal", e.what());
        return CODE_INTERNAL;
    } catch (...) {
        set_err(err, CODE_INTERNAL, "sy.internal", "unknown exception");
        return CODE_INTERNAL;
    }
}

void synnax_client_close(SynnaxClient *client) {
    delete client;
}

const char *synnax_client_version(void) {
    return SYNNAX_VERSION;
}
