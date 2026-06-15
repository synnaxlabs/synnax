// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <exception>
#include <memory>
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
    const char *client_cert_file,
    const char *client_key_file,
    const uint32_t max_retries,
    const int64_t clock_skew_threshold,
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
        const std::string client_cert = str_or(client_cert_file, "");
        if (!client_cert.empty()) c.client_cert_file = client_cert;
        const std::string client_key = str_or(client_key_file, "");
        if (!client_key.empty()) c.client_key_file = client_key;
        if (max_retries != 0) c.max_retries = max_retries;
        if (clock_skew_threshold > 0)
            c.clock_skew_threshold = x::telem::TimeSpan(clock_skew_threshold);

        auto wrapper = std::make_unique<SynnaxClient>(c);
        const auto state = wrapper->client.connectivity->check();
        if (state.status != synnax::connection::Status::CONNECTED) {
            const bool has_err = !state.error.ok();
            set_err(
                err,
                CODE_ERROR,
                has_err ? state.error.type : std::string("sy.connection"),
                has_err ? state.error.data : state.message
            );
            return CODE_ERROR;
        }
        *out_client = wrapper.release();
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
