// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <string>

#include <openssl/evp.h>

namespace x::base64 {

/// @brief base64-encodes the input string.
inline std::string encode(const std::string &input) {
    if (input.empty()) return "";
    const auto len = 4 * ((input.size() + 2) / 3);
    std::string out(len + 1, '\0');
    const auto written = EVP_EncodeBlock(
        reinterpret_cast<unsigned char *>(out.data()),
        reinterpret_cast<const unsigned char *>(input.data()),
        static_cast<int>(input.size())
    );
    out.resize(written);
    return out;
}

/// @brief base64-decodes the input string.
inline std::string decode(const std::string &input) {
    if (input.empty()) return "";
    const auto len = 3 * input.size() / 4;
    std::string out(len, '\0');
    const auto written = EVP_DecodeBlock(
        reinterpret_cast<unsigned char *>(out.data()),
        reinterpret_cast<const unsigned char *>(input.data()),
        static_cast<int>(input.size())
    );
    if (written < 0) return "";
    // EVP_DecodeBlock always writes 3*(len/4) bytes regardless of padding. Trim by the
    // number of trailing '=' characters.
    const auto padding = std::count(input.rbegin(), input.rend(), '=');
    out.resize(written - padding);
    return out;
}

}
