// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "x/cpp/xerrors/errors.h"
#include "driver/errors/errors.h"
#include "driver/visa/api/types.h"
#include "driver/visa/api/api.h"

namespace visa {

using namespace visa_types;

const xerrors::Error CRITICAL_ERROR = driver::CRITICAL_HARDWARE_ERROR.sub("visa");
const xerrors::Error TEMPORARY_ERROR = driver::TEMPORARY_HARDWARE_ERROR.sub("visa");

/// @brief Converts a VISA status code to a xerrors::Error using the API wrapper.
/// @param api The VISA API wrapper (can be null if VISA not loaded)
/// @param status The VISA status code
/// @param session Optional session for getting detailed error description
/// @return A xerrors::Error wrapping the VISA error
template<typename API_T>
inline xerrors::Error parse_visa_error(
    const std::shared_ptr<API_T> &api,
    const ViStatus status,
    const ViSession session = VI_NULL
) {
    if (status >= VI_SUCCESS) return xerrors::NIL;

    // Get error description if API is available
    ViChar err_desc[256] = "VISA error (API not available)";
    if (api != nullptr) {
        const ViStatus desc_status = api->status_desc(session, status, err_desc);
        if (desc_status < VI_SUCCESS) {
            // If getting description fails, use default message
            snprintf(err_desc, sizeof(err_desc), "VISA error (code: 0x%08X)", static_cast<unsigned int>(status));
        }
    }

    // Build error code string
    char code_str[32];
    snprintf(code_str, sizeof(code_str), "0x%08X", static_cast<unsigned int>(status));

    // Classify errors as temporary or critical
    switch (status) {
        case VI_ERROR_TMO:
        case VI_ERROR_CONN_LOST:
        case VI_ERROR_IO:
            return xerrors::Error(TEMPORARY_ERROR.sub(code_str), std::string(err_desc));
        default:
            return xerrors::Error(CRITICAL_ERROR.sub(code_str), std::string(err_desc));
    }
}

}