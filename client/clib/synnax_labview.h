// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Sanitized prototypes for LabVIEW's Import Shared Library wizard ONLY. The wizard's C
// parser cannot resolve `size_t` (it skips <stddef.h>) or the opaque `SynnaxClient`
// typedef, so it rejects the real headers. This header is ABI-identical to the exports
// in synnax.h / framer/framer.h but spells every type the wizard understands: handles
// are void* (mapped to UPtr), `size_t` is uint64_t (8 bytes on the 64-bit-only target),
// SYNNAX_EXPORT is empty, and no comments sit inside parameter lists. Do not consume
// this from C/C++ — use synnax.h. Point the wizard at this file, then load the DLL.

#pragma once

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

#define SYNNAX_EXPORT

// Opaque handles as void* so the wizard maps them to UPtr instead of empty clusters.
// SynnaxError is a 644-byte struct (i32 code, char type[128], char message[512]); error-
// returning functions take a uint8_t* (Array Data Pointer) to it: allocate U8[644].
typedef void *SynnaxClient;
typedef void *SynnaxWriter;
typedef void *SynnaxError;

SYNNAX_EXPORT int32_t synnax_client_open(
    const char *host,
    uint16_t port,
    const char *username,
    const char *password,
    int32_t secure,
    const char *ca_cert_file,
    const char *client_cert_file,
    const char *client_key_file,
    uint32_t max_retries,
    int64_t clock_skew_threshold,
    SynnaxClient *out_client,
    uint8_t *err
);

SYNNAX_EXPORT void synnax_client_close(SynnaxClient client);

SYNNAX_EXPORT const char *synnax_client_version(void);

SYNNAX_EXPORT int32_t synnax_channel_retrieve_keys(
    SynnaxClient client,
    const char *names,
    uint64_t name_count,
    uint32_t *out_keys,
    uint32_t *out_index_keys,
    char *out_dtypes,
    uint64_t out_dtypes_size,
    uint8_t *err
);

SYNNAX_EXPORT int32_t synnax_channel_create(
    SynnaxClient client,
    const char *names,
    const char *data_types,
    const uint8_t *is_index,
    const uint32_t *index,
    const uint8_t *is_virtual,
    uint64_t count,
    uint32_t *out_keys,
    uint8_t *err
);

SYNNAX_EXPORT int32_t synnax_writer_open(
    SynnaxClient client,
    int64_t start,
    const uint32_t *channels,
    uint64_t channel_count,
    const uint8_t *authorities,
    uint64_t authority_count,
    const char *subject_name,
    uint32_t subject_group,
    int32_t mode,
    int32_t err_on_unauthorized,
    int32_t enable_auto_commit,
    int64_t auto_index_persist_interval,
    int32_t auto_index,
    SynnaxWriter *out_writer,
    uint8_t *err
);

SYNNAX_EXPORT int32_t synnax_writer_write(
    SynnaxWriter writer,
    uint32_t index_channel,
    const int64_t *timestamps,
    const uint32_t *channels,
    uint64_t channel_count,
    const void *data,
    uint64_t data_size,
    uint64_t sample_count,
    const char *data_type,
    uint8_t *err
);

SYNNAX_EXPORT int32_t synnax_writer_write_strings(
    SynnaxWriter writer,
    uint32_t index_channel,
    const int64_t *timestamps,
    const uint32_t *channels,
    uint64_t channel_count,
    const void *data,
    uint64_t data_size,
    uint64_t sample_count,
    uint8_t *err
);

SYNNAX_EXPORT int32_t
synnax_writer_commit(SynnaxWriter writer, int64_t *out_end_ts, uint8_t *err);

SYNNAX_EXPORT int32_t synnax_writer_close(SynnaxWriter writer, uint8_t *err);

#ifdef __cplusplus
}
#endif
