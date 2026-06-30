// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Internal shared helpers for the Synnax C-ABI implementation: status codes, the
// error-filling helpers, and the full definition of the opaque client handle. Not
// part of the public surface and not exported from the DLL.

#pragma once

#include <cstring>
#include <sstream>
#include <string>
#include <vector>

#include "client/clib/types.h"
#include "client/cpp/synnax.h"

namespace synnax::clib {

inline constexpr int32_t CODE_OK = 0;
inline constexpr int32_t CODE_ERROR = 1; // failure reported by the client
inline constexpr int32_t CODE_INTERNAL = 2; // bad argument or caught C++ exception

/// @brief copies src into the fixed dst buffer, always null-terminating.
inline void copy_str(char *dst, const size_t dst_size, const std::string &src) {
    if (dst == nullptr || dst_size == 0) return;
    const size_t n = src.size() < dst_size - 1 ? src.size() : dst_size - 1;
    std::memcpy(dst, src.data(), n);
    dst[n] = '\0';
}

/// @brief fills err (when non-NULL) with the given code, type, and message.
inline void set_err(
    SynnaxError *err,
    const int32_t code,
    const std::string &type,
    const std::string &message
) {
    if (err == nullptr) return;
    err->code = code;
    copy_str(err->type, sizeof(err->type), type);
    copy_str(err->message, sizeof(err->message), message);
}

/// @brief resets err (when non-NULL) to the success state.
inline void clear_err(SynnaxError *err) {
    if (err == nullptr) return;
    err->code = CODE_OK;
    err->type[0] = '\0';
    err->message[0] = '\0';
}

/// @brief returns s as a std::string, or fallback when s is NULL.
inline std::string str_or(const char *s, const char *fallback) {
    return s != nullptr ? std::string(s) : std::string(fallback);
}

/// @brief splits s into tokens on '\n'; returns empty when s is NULL.
inline std::vector<std::string> split_newlines(const char *s) {
    std::vector<std::string> out;
    std::stringstream ss(str_or(s, ""));
    std::string token;
    while (std::getline(ss, token, '\n'))
        out.push_back(token);
    return out;
}

}

/// @brief full definition of the opaque client handle, shared by the client and
/// framer translation units.
struct SynnaxClient {
    synnax::Synnax client;

    explicit SynnaxClient(const synnax::Config &cfg): client(cfg) {}
};
