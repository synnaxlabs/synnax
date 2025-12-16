// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <vector>

namespace binary {
/// @brief Decodes a base64-encoded string into a vector of bytes
/// @param encoded The base64-encoded string
/// @return A vector of decoded bytes
static std::vector<uint8_t> decode_base64(const std::string &encoded) {
    static const std::string base64_chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
                                            "abcdefghijklmnopqrstuvwxyz"
                                            "0123456789+/";

    std::vector<uint8_t> decoded;
    std::vector<int> temp(4);
    int i = 0;
    int in_len = encoded.size();
    int in_idx = 0;

    while (in_len-- && encoded[in_idx] != '=' &&
           (isalnum(encoded[in_idx]) || encoded[in_idx] == '+' ||
            encoded[in_idx] == '/')) {
        temp[i++] = encoded[in_idx++];
        if (i == 4) {
            for (i = 0; i < 4; i++) {
                temp[i] = base64_chars.find(temp[i]);
            }

            decoded.push_back((temp[0] << 2) + ((temp[1] & 0x30) >> 4));
            decoded.push_back(((temp[1] & 0xf) << 4) + ((temp[2] & 0x3c) >> 2));
            decoded.push_back(((temp[2] & 0x3) << 6) + temp[3]);

            i = 0;
        }
    }

    if (i) {
        for (int j = i; j < 4; j++) {
            temp[j] = 0;
        }

        for (int j = 0; j < i; j++) {
            temp[j] = base64_chars.find(temp[j]);
        }

        decoded.push_back((temp[0] << 2) + ((temp[1] & 0x30) >> 4));

        if (i > 2) {
            decoded.push_back(((temp[1] & 0xf) << 4) + ((temp[2] & 0x3c) >> 2));
        }
    }

    return decoded;
}
}
