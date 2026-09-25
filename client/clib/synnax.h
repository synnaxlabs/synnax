// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Pure-C public umbrella header for the Synnax C-ABI library (synnax_clib), consumed
// by flat-C FFI callers such as LabVIEW's Call Library Function Node. Importing this
// header pulls in the full surface (client + framer). The API is deliberately flat
// because the CLFN cannot build C structs that contain pointers.

#pragma once

#include <stdint.h>

#include "client/clib/channel/channel.h"
#include "client/clib/export.h"
#include "client/clib/framer/framer.h"
#include "client/clib/types.h"

#ifdef __cplusplus
extern "C" {
#endif

/// @brief connects to a Synnax Core and verifies connectivity. secure=0 connects in
/// plaintext; secure!=0 uses TLS and verifies the Core against the system trust store.
/// max_retries=0 and clock_skew_threshold<=0 take the Config defaults. NULL
/// host/username/password fall back to localhost/synnax/seldon, port 0 to 9090. On
/// success writes the handle to *out_client and returns 0.
SYNNAX_EXPORT int32_t synnax_client_open(
    const char *host,
    uint16_t port,
    const char *username,
    const char *password,
    int32_t secure,
    uint32_t max_retries,
    int64_t clock_skew_threshold,
    SynnaxClient **out_client,
    SynnaxError *err
);

/// @brief closes and frees a client. Safe on NULL.
SYNNAX_EXPORT void synnax_client_close(SynnaxClient *client);

/// @brief returns the static Synnax client library version string (the analog of the
/// other clients' clientVersion / __version__). The caller must not free it.
SYNNAX_EXPORT const char *synnax_client_version(void);

#ifdef __cplusplus
}
#endif
