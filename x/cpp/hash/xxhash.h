// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <cstring>
#include <string>
#include <string_view>

/// @brief non-cryptographic hash functions.
namespace x::hash {
namespace priv {
constexpr uint64_t XXH_P1 = 0x9e3779b185ebca87ULL;
constexpr uint64_t XXH_P2 = 0xc2b2ae3d27d4eb4fULL;
constexpr uint64_t XXH_P3 = 0x165667b19e3779f9ULL;
constexpr uint64_t XXH_P4 = 0x85ebca77c2b2ae63ULL;
constexpr uint64_t XXH_P5 = 0x27d4eb2f165667c5ULL;

inline uint64_t rotl(const uint64_t v, const int r) {
    return (v << r) | (v >> (64 - r));
}

inline uint64_t round(const uint64_t acc, const uint64_t lane) {
    return rotl(acc + lane * XXH_P2, 31) * XXH_P1;
}

inline uint64_t merge_round(const uint64_t h, const uint64_t acc) {
    return (h ^ round(0, acc)) * XXH_P1 + XXH_P4;
}

inline uint64_t read_u64(const char *p) {
    uint64_t v;
    std::memcpy(&v, p, sizeof(v));
    return v;
}

inline uint32_t read_u32(const char *p) {
    uint32_t v;
    std::memcpy(&v, p, sizeof(v));
    return v;
}
}

/// @brief computes the 64-bit xxHash (XXH64) of the input bytes.
/// @param data the bytes to hash.
/// @param seed the hash seed; defaults to 0.
/// @returns the 64-bit hash value.
inline uint64_t xxhash64(const std::string_view data, const uint64_t seed = 0) {
    // Little-endian lane reads per the xxHash spec; all supported targets are
    // little-endian.
    const char *p = data.data();
    const char *end = p + data.size();
    uint64_t h;
    if (data.size() >= 32) {
        uint64_t a1 = seed + priv::XXH_P1 + priv::XXH_P2;
        uint64_t a2 = seed + priv::XXH_P2;
        uint64_t a3 = seed;
        uint64_t a4 = seed - priv::XXH_P1;
        for (; p + 32 <= end; p += 32) {
            a1 = priv::round(a1, priv::read_u64(p));
            a2 = priv::round(a2, priv::read_u64(p + 8));
            a3 = priv::round(a3, priv::read_u64(p + 16));
            a4 = priv::round(a4, priv::read_u64(p + 24));
        }
        h = priv::rotl(a1, 1) + priv::rotl(a2, 7) + priv::rotl(a3, 12) +
            priv::rotl(a4, 18);
        h = priv::merge_round(h, a1);
        h = priv::merge_round(h, a2);
        h = priv::merge_round(h, a3);
        h = priv::merge_round(h, a4);
    } else
        h = seed + priv::XXH_P5;
    h += data.size();
    for (; p + 8 <= end; p += 8)
        h = priv::rotl(h ^ priv::round(0, priv::read_u64(p)), 27) * priv::XXH_P1 +
            priv::XXH_P4;
    if (p + 4 <= end) {
        h = priv::rotl(h ^ priv::read_u32(p) * priv::XXH_P1, 23) * priv::XXH_P2 +
            priv::XXH_P3;
        p += 4;
    }
    for (; p < end; ++p)
        h = priv::rotl(h ^ static_cast<uint8_t>(*p) * priv::XXH_P5, 11) * priv::XXH_P1;
    h ^= h >> 33;
    h *= priv::XXH_P2;
    h ^= h >> 29;
    h *= priv::XXH_P3;
    h ^= h >> 32;
    return h;
}

/// @brief computes the 64-bit xxHash (XXH64, seed 0) of the input bytes.
/// @param data the bytes to hash.
/// @returns the hash as a 16-character zero-padded lowercase hex string.
inline std::string xxhash64_hex(const std::string_view data) {
    uint64_t h = xxhash64(data);
    std::string out(16, '0');
    constexpr char digits[] = "0123456789abcdef";
    for (int i = 15; i >= 0; --i) {
        out[i] = digits[h & 0xf];
        h >>= 4;
    }
    return out;
}
}
