// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "x/cpp/base64/base64.h"

namespace x::base64 {

namespace {
constexpr char
    ENCODE_TABLE[] = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";

constexpr unsigned char DECODE_TABLE[] = {
    64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64,
    64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64,
    64, 64, 64, 64, 64, 62, 64, 64, 64, 63, 52, 53, 54, 55, 56, 57, 58, 59, 60,
    61, 64, 64, 64, 64, 64, 64, 64, 0,  1,  2,  3,  4,  5,  6,  7,  8,  9,  10,
    11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 64, 64, 64, 64,
    64, 64, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42,
    43, 44, 45, 46, 47, 48, 49, 50, 51, 64, 64, 64, 64, 64,
};
}

std::string encode(const std::string &input) {
    if (input.empty()) return "";
    std::string out;
    out.reserve(4 * ((input.size() + 2) / 3));
    const auto *data = reinterpret_cast<const unsigned char *>(input.data());
    const auto len = input.size();
    size_t i = 0;
    for (; i + 2 < len; i += 3) {
        out.push_back(ENCODE_TABLE[data[i] >> 2]);
        out.push_back(ENCODE_TABLE[((data[i] & 0x03) << 4) | (data[i + 1] >> 4)]);
        out.push_back(ENCODE_TABLE[((data[i + 1] & 0x0f) << 2) | (data[i + 2] >> 6)]);
        out.push_back(ENCODE_TABLE[data[i + 2] & 0x3f]);
    }
    if (i < len) {
        out.push_back(ENCODE_TABLE[data[i] >> 2]);
        if (i + 1 < len) {
            out.push_back(ENCODE_TABLE[((data[i] & 0x03) << 4) | (data[i + 1] >> 4)]);
            out.push_back(ENCODE_TABLE[(data[i + 1] & 0x0f) << 2]);
        } else {
            out.push_back(ENCODE_TABLE[(data[i] & 0x03) << 4]);
            out.push_back('=');
        }
        out.push_back('=');
    }
    return out;
}

std::string decode(const std::string &input) {
    if (input.empty()) return "";
    auto len = input.size();
    while (len > 0 && input[len - 1] == '=')
        --len;
    std::string out;
    out.reserve(3 * len / 4);
    unsigned int buf = 0;
    int bits = 0;
    for (size_t i = 0; i < len; ++i) {
        const auto c = static_cast<unsigned char>(input[i]);
        if (c >= 128) continue;
        const auto val = DECODE_TABLE[c];
        if (val == 64) continue;
        buf = (buf << 6) | val;
        bits += 6;
        if (bits >= 8) {
            bits -= 8;
            out.push_back(static_cast<char>((buf >> bits) & 0xFF));
        }
    }
    return out;
}

}
