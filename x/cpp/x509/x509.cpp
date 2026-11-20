// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <array>
#include <cctype>
#include <cstdint>
#include <iterator>
#include <limits>
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include "mbedtls/asn1.h"
#include "mbedtls/ctr_drbg.h"
#include "mbedtls/entropy.h"
#include "mbedtls/error.h"
#include "mbedtls/oid.h"
#include "mbedtls/pem.h"
#include "mbedtls/pk.h"
#include "mbedtls/sha1.h"
#include "mbedtls/sha256.h"
#include "mbedtls/x509_crt.h"

#include "x/cpp/date/date.h"
#include "x/cpp/defer/defer.h"
#include "x/cpp/strings/strings.h"
#include "x/cpp/x509/x509.h"

namespace x::x509 {
namespace {
constexpr std::string_view PEM_HEADER = "-----BEGIN CERTIFICATE-----";
constexpr std::string_view PEM_FOOTER = "-----END CERTIFICATE-----";
// A URI in the subject alternative name uses tag [6].
constexpr int SAN_URI_TAG = MBEDTLS_ASN1_CONTEXT_SPECIFIC | 6;

/// @returns true if data starts as a DER certificate: a SEQUENCE tag, then a long-form
/// length. Text cannot start this way, since 0x81 to 0x84 never follow an ASCII byte in
/// UTF-8.
bool is_der(const std::string_view data) {
    if (data.size() < 2 || static_cast<unsigned char>(data[0]) != 0x30) return false;
    const auto length = static_cast<unsigned char>(data[1]);
    return length >= 0x81 && length <= 0x84;
}

std::string mbedtls_message(const int code) {
    std::array<char, 128> buf{};
    mbedtls_strerror(code, buf.data(), buf.size());
    return buf.data();
}

/// @returns the time, clamped as the Core clamps it to the range of a TimeStamp: from
/// the Unix epoch to the largest TimeStamp. A certificate may expire in 9999.
telem::TimeStamp to_timestamp(const mbedtls_x509_time &t) {
    const int64_t days = date::days_from_civil({
        static_cast<uint16_t>(t.year),
        static_cast<uint8_t>(t.mon),
        static_cast<uint8_t>(t.day),
    });
    const int64_t seconds = days * 86400 + int64_t{t.hour} * 3600 +
                            int64_t{t.min} * 60 + t.sec;
    const int64_t max_seconds = std::numeric_limits<int64_t>::max() /
                                telem::SECOND.nanoseconds();
    if (seconds < 0) return telem::TimeStamp(0);
    if (seconds > max_seconds) return telem::TimeStamp::max();
    return telem::TimeStamp(seconds * telem::SECOND);
}

/// @brief a random number generator for mbedTLS key operations.
class Random {
public:
    Random() {
        mbedtls_entropy_init(&this->entropy);
        mbedtls_ctr_drbg_init(&this->drbg);
    }

    ~Random() {
        mbedtls_ctr_drbg_free(&this->drbg);
        mbedtls_entropy_free(&this->entropy);
    }

    Random(const Random &) = delete;
    Random &operator=(const Random &) = delete;

    /// @returns 0, or the mbedTLS error code.
    int seed() {
        return mbedtls_ctr_drbg_seed(
            &this->drbg,
            mbedtls_entropy_func,
            &this->entropy,
            nullptr,
            0
        );
    }

    mbedtls_entropy_context entropy{};
    mbedtls_ctr_drbg_context drbg{};
};

constexpr std::string_view LOWER_HEX = "0123456789abcdef";
constexpr std::string_view UPPER_HEX = "0123456789ABCDEF";

/// @returns the bytes as hex in the given digits.
template<typename Bytes>
std::string hex(const Bytes &bytes, const std::string_view digits = LOWER_HEX) {
    std::string out;
    out.reserve(std::size(bytes) * 2);
    for (const unsigned char b: bytes) {
        out.push_back(digits[b >> 4]);
        out.push_back(digits[b & 0x0F]);
    }
    return out;
}

/// @brief one attribute of a distinguished name.
struct Attribute {
    /// @brief the attribute type as a dotted OID, such as "2.5.4.3".
    std::string oid;
    /// @brief the value as UTF-8 when text is true, or else "#" and the hex of its DER.
    std::string value;
    /// @brief true when the value is a string type that decodes to UTF-8.
    bool text = true;
};

void append_utf8(std::string &out, const uint32_t code_point) {
    if (code_point < 0x80) {
        out.push_back(static_cast<char>(code_point));
    } else if (code_point < 0x800) {
        out.push_back(static_cast<char>(0xC0 | (code_point >> 6)));
        out.push_back(static_cast<char>(0x80 | (code_point & 0x3F)));
    } else {
        out.push_back(static_cast<char>(0xE0 | (code_point >> 12)));
        out.push_back(static_cast<char>(0x80 | ((code_point >> 6) & 0x3F)));
        out.push_back(static_cast<char>(0x80 | (code_point & 0x3F)));
    }
}

bool printable(const unsigned char c) {
    return std::isalnum(c) != 0 || (c >= '\'' && c <= ')') || (c >= '+' && c <= '/') ||
           c == ' ' || c == ':' || c == '=' || c == '?' || c == '*' || c == '&';
}

/// @returns the value of a string type as UTF-8, read as Go's crypto/x509 reads it, or
/// nothing when Go refuses it.
std::optional<std::string> decode_text(const int tag, std::string_view value) {
    switch (tag) {
        case MBEDTLS_ASN1_T61_STRING: {
            std::string out;
            for (const unsigned char c: value)
                append_utf8(out, c);
            return out;
        }
        case MBEDTLS_ASN1_PRINTABLE_STRING:
            if (!std::ranges::all_of(value, printable)) return std::nullopt;
            return std::string(value);
        case MBEDTLS_ASN1_UTF8_STRING:
            if (!strings::valid_utf8(value)) return std::nullopt;
            return std::string(value);
        case MBEDTLS_ASN1_IA5_STRING:
            if (!std::ranges::all_of(value, [](const unsigned char c) {
                    return c < 0x80;
                }))
                return std::nullopt;
            return std::string(value);
        case MBEDTLS_ASN1_BMP_STRING: {
            if (value.size() % 2 != 0) return std::nullopt;
            if (value.ends_with(std::string_view("\0\0", 2))) value.remove_suffix(2);
            std::string out;
            for (size_t i = 0; i < value.size(); i += 2) {
                const uint32_t unit = static_cast<unsigned char>(value[i]) << 8 |
                                      static_cast<unsigned char>(value[i + 1]);
                if (unit == 0xFFFE || unit == 0xFFFF ||
                    (unit >= 0xFDD0 && unit <= 0xFDEF) ||
                    (unit >= 0xD800 && unit <= 0xDFFF))
                    return std::nullopt;
                append_utf8(out, unit);
            }
            return out;
        }
        default:
            return std::nullopt;
    }
}

bool text_tag(const int tag) {
    return tag == MBEDTLS_ASN1_T61_STRING || tag == MBEDTLS_ASN1_PRINTABLE_STRING ||
           tag == MBEDTLS_ASN1_UTF8_STRING || tag == MBEDTLS_ASN1_IA5_STRING ||
           tag == MBEDTLS_ASN1_BMP_STRING;
}

/// @returns the attributes of a DER encoded name in the order of the encoding, or
/// nothing when the encoding is not valid.
std::optional<std::vector<Attribute>> read_attributes(const mbedtls_x509_buf &raw) {
    std::vector<Attribute> attrs;
    unsigned char *p = raw.p;
    const unsigned char *end = raw.p + raw.len;
    size_t len = 0;
    constexpr int SEQUENCE = MBEDTLS_ASN1_CONSTRUCTED | MBEDTLS_ASN1_SEQUENCE;
    constexpr int SET = MBEDTLS_ASN1_CONSTRUCTED | MBEDTLS_ASN1_SET;
    if (mbedtls_asn1_get_tag(&p, end, &len, SEQUENCE) != 0) return std::nullopt;
    while (p < end) {
        if (mbedtls_asn1_get_tag(&p, end, &len, SET) != 0) return std::nullopt;
        const unsigned char *set_end = p + len;
        while (p < set_end) {
            if (mbedtls_asn1_get_tag(&p, set_end, &len, SEQUENCE) != 0)
                return std::nullopt;
            unsigned char *const attr_end = p + len;
            mbedtls_asn1_buf oid{};
            if (mbedtls_asn1_get_tag(&p, attr_end, &oid.len, MBEDTLS_ASN1_OID) != 0)
                return std::nullopt;
            oid.tag = MBEDTLS_ASN1_OID;
            oid.p = p;
            p += oid.len;
            const unsigned char *const value_start = p;
            if (p >= attr_end) return std::nullopt;
            const int tag = *p++;
            if (mbedtls_asn1_get_len(&p, attr_end, &len) != 0) return std::nullopt;
            std::array<char, 128> dotted{};
            if (mbedtls_oid_get_numeric_string(dotted.data(), dotted.size(), &oid) < 0)
                return std::nullopt;
            Attribute attr{.oid = dotted.data()};
            const std::string_view value(reinterpret_cast<const char *>(p), len);
            auto text = decode_text(tag, value);
            // Go refuses the whole certificate here. The raw bytes keep a name such as
            // CN=PLC_01 readable.
            if (!text && text_tag(tag) && strings::valid_utf8(value))
                text = std::string(value);
            if (text) {
                attr.value = std::move(*text);
            } else {
                attr.text = false;
                attr.value = "#" + hex(std::string_view(
                                       reinterpret_cast<const char *>(value_start),
                                       static_cast<size_t>(attr_end - value_start)
                                   ));
            }
            attrs.push_back(std::move(attr));
            p = attr_end;
        }
    }
    return attrs;
}

/// @brief the attribute types that Go names. Go gathers the values of each type in
/// this order, and keeps only the last value of CN and SERIALNUMBER.
constexpr std::array<std::pair<std::string_view, std::string_view>, 9> NAMED_TYPES = {{
    {"2.5.4.6", "C"},
    {"2.5.4.8", "ST"},
    {"2.5.4.7", "L"},
    {"2.5.4.9", "STREET"},
    {"2.5.4.17", "POSTALCODE"},
    {"2.5.4.10", "O"},
    {"2.5.4.11", "OU"},
    {"2.5.4.3", "CN"},
    {"2.5.4.5", "SERIALNUMBER"},
}};

bool single_valued(const std::string_view name) {
    return name == "CN" || name == "SERIALNUMBER";
}

/// @returns the value with the RFC 2253 escapes that Go adds.
std::string escape(const std::string &value) {
    std::string out;
    out.reserve(value.size());
    for (size_t i = 0; i < value.size(); i++) {
        const char c = value[i];
        const bool escaped = c == ',' || c == '+' || c == '"' || c == '\\' ||
                             c == '<' || c == '>' || c == ';' ||
                             (c == ' ' && (i == 0 || i == value.size() - 1)) ||
                             (c == '#' && i == 0);
        if (escaped) out.push_back('\\');
        out.push_back(c);
    }
    return out;
}

/// @returns the name in the form of Go's pkix.Name.String(), which the Core stores,
/// or nothing when the encoding is not valid.
std::optional<std::string> format_name(const mbedtls_x509_buf &raw) {
    const auto attrs = read_attributes(raw);
    if (!attrs) return std::nullopt;
    using RDN = std::vector<std::pair<std::string, std::string>>;
    std::vector<RDN> rdns;
    for (const auto &attr: *attrs)
        if (std::ranges::none_of(NAMED_TYPES, [&](const auto &t) {
                return t.first == attr.oid;
            }))
            rdns.push_back({{attr.oid, attr.text ? escape(attr.value) : attr.value}});
    for (const auto &[oid, name]: NAMED_TYPES) {
        RDN rdn;
        for (const auto &attr: *attrs)
            if (attr.oid == oid && attr.text)
                rdn.emplace_back(name, escape(attr.value));
        if (rdn.empty()) continue;
        if (single_valued(name)) {
            if (rdn.back().second.empty()) continue;
            rdn = {rdn.back()};
        }
        rdns.push_back(std::move(rdn));
    }
    std::string out;
    for (auto rdn = rdns.rbegin(); rdn != rdns.rend(); ++rdn) {
        if (rdn != rdns.rbegin()) out.push_back(',');
        for (size_t i = 0; i < rdn->size(); i++) {
            if (i > 0) out.push_back('+');
            out += (*rdn)[i].first + "=" + (*rdn)[i].second;
        }
    }
    return out;
}
}

std::vector<std::string_view> Certificate::pem_blocks(const std::string_view data) {
    std::vector<std::string_view> blocks;
    if (is_der(data)) return blocks;
    for (auto start = data.find(PEM_HEADER); start != std::string_view::npos;
         start = data.find(PEM_HEADER, start)) {
        const auto footer = data.find(PEM_FOOTER, start);
        if (footer == std::string_view::npos) break;
        const auto end = footer + PEM_FOOTER.size();
        blocks.push_back(data.substr(start, end - start));
        start = end;
    }
    return blocks;
}

bool Certificate::is_pem(const std::string_view data) {
    return !is_der(data) && data.contains(PEM_HEADER);
}

std::pair<Certificate, errors::Error> Certificate::parse(const std::string_view data) {
    mbedtls_x509_crt crt;
    mbedtls_x509_crt_init(&crt);
    const defer::defer free_crt([&] { mbedtls_x509_crt_free(&crt); });
    const auto blocks = pem_blocks(data);
    const std::string buf(blocks.empty() ? data : blocks.front());
    // mbedTLS reads PEM only with its terminating null byte in the length.
    const auto len = blocks.empty() ? buf.size() : buf.size() + 1;
    if (const int ret = mbedtls_x509_crt_parse(
            &crt,
            reinterpret_cast<const unsigned char *>(buf.c_str()),
            len
        );
        ret != 0)
        return {
            Certificate(),
            errors::Error(
                INVALID,
                "Failed to parse certificate: " + mbedtls_message(ret)
            )
        };
    auto subject = format_name(crt.subject_raw);
    auto issuer = format_name(crt.issuer_raw);
    if (!subject || !issuer)
        return {
            Certificate(),
            errors::Error(INVALID, "Failed to parse certificate: a name is not valid.")
        };
    Certificate cert;
    cert.der_bytes.assign(reinterpret_cast<const char *>(crt.raw.p), crt.raw.len);
    cert.subject_name = std::move(*subject);
    cert.issuer_name = std::move(*issuer);
    for (const auto *san = &crt.subject_alt_names; san != nullptr; san = san->next)
        if (san->buf.tag == SAN_URI_TAG) {
            cert.san_uri.assign(
                reinterpret_cast<const char *>(san->buf.p),
                san->buf.len
            );
            break;
        }
    cert.valid_from = to_timestamp(crt.valid_from);
    cert.valid_to = to_timestamp(crt.valid_to);
    return {std::move(cert), errors::NIL};
}

std::string Certificate::pem() const {
    const std::string header = std::string(PEM_HEADER) + "\n";
    const std::string footer = std::string(PEM_FOOTER) + "\n";
    const auto *der = reinterpret_cast<const unsigned char *>(this->der_bytes.data());
    size_t len = 0;
    mbedtls_pem_write_buffer(
        header.c_str(),
        footer.c_str(),
        der,
        this->der_bytes.size(),
        nullptr,
        0,
        &len
    );
    std::vector<unsigned char> buf(len);
    if (mbedtls_pem_write_buffer(
            header.c_str(),
            footer.c_str(),
            der,
            this->der_bytes.size(),
            buf.data(),
            buf.size(),
            &len
        ) != 0)
        return "";
    // The written length counts the terminating null byte.
    return {reinterpret_cast<const char *>(buf.data()), len - 1};
}

std::string Certificate::thumbprint() const {
    return hex(this->sha1_hash(), UPPER_HEX);
}

std::string Certificate::sha1() const {
    return hex(this->sha1_hash(), LOWER_HEX);
}

std::string Certificate::sha256() const {
    std::array<unsigned char, 32> hash{};
    mbedtls_sha256(
        reinterpret_cast<const unsigned char *>(this->der_bytes.data()),
        this->der_bytes.size(),
        hash.data(),
        0
    );
    return hex(hash, LOWER_HEX);
}

std::array<unsigned char, 20> Certificate::sha1_hash() const {
    std::array<unsigned char, 20> hash{};
    mbedtls_sha1(
        reinterpret_cast<const unsigned char *>(this->der_bytes.data()),
        this->der_bytes.size(),
        hash.data()
    );
    return hash;
}

bool Certificate::valid_at(const telem::TimeStamp time) const {
    return time >= this->valid_from && time <= this->valid_to;
}

errors::Error Certificate::check_key(const std::string &key) const {
    Random random;
    if (const int ret = random.seed(); ret != 0)
        return errors::Error(KEY_INVALID, mbedtls_message(ret));
    mbedtls_x509_crt crt;
    mbedtls_x509_crt_init(&crt);
    mbedtls_pk_context pk;
    mbedtls_pk_init(&pk);
    const defer::defer free_all([&] {
        mbedtls_pk_free(&pk);
        mbedtls_x509_crt_free(&crt);
    });
    const int crt_ret = mbedtls_x509_crt_parse_der(
        &crt,
        reinterpret_cast<const unsigned char *>(this->der_bytes.data()),
        this->der_bytes.size()
    );
    if (crt_ret != 0) return errors::Error(KEY_INVALID, mbedtls_message(crt_ret));
    // mbedTLS reads a PEM key only with its terminating null byte in the length.
    const auto key_len = key.contains("-----BEGIN") ? key.size() + 1 : key.size();
    const int key_ret = mbedtls_pk_parse_key(
        &pk,
        reinterpret_cast<const unsigned char *>(key.c_str()),
        key_len,
        nullptr,
        0,
        mbedtls_ctr_drbg_random,
        &random.drbg
    );
    if (key_ret == MBEDTLS_ERR_PK_PASSWORD_REQUIRED)
        return errors::Error(
            KEY_INVALID,
            "The private key needs a password, which is not supported."
        );
    if (key_ret != 0)
        return errors::Error(
            KEY_INVALID,
            "Failed to parse the private key: " + mbedtls_message(key_ret)
        );
    if (mbedtls_pk_check_pair(&crt.pk, &pk, mbedtls_ctr_drbg_random, &random.drbg) != 0)
        return errors::Error(
            KEY_INVALID,
            "The private key does not match the certificate."
        );
    return errors::NIL;
}
}
