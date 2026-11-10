// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <array>
#include <cstdint>
#include <string>
#include <vector>

#include "absl/log/log.h"
#include "mbedtls/ctr_drbg.h"
#include "mbedtls/entropy.h"
#include "mbedtls/error.h"
#include "mbedtls/pem.h"
#include "mbedtls/pk.h"
#include "mbedtls/sha1.h"
#include "mbedtls/x509_crt.h"
#include "open62541/plugin/create_certificate.h"
#include "open62541/types.h"
#include "open62541/util.h"

#include "x/cpp/date/date.h"
#include "x/cpp/defer/defer.h"
#include "x/cpp/fs/fs.h"

#include "driver/opcua/certificate/certificate.h"
#include "driver/opcua/errors/errors.h"

namespace driver::opcua::certificate {
namespace {
constexpr std::string_view PEM_HEADER = "-----BEGIN CERTIFICATE-----";
constexpr std::string_view PEM_FOOTER = "-----END CERTIFICATE-----";
// A URI in the subject alternative name uses tag [6].
constexpr int SAN_URI_TAG = MBEDTLS_ASN1_CONTEXT_SPECIFIC | 6;

std::string mbedtls_message(const int code) {
    std::array<char, 128> buf{};
    mbedtls_strerror(code, buf.data(), buf.size());
    return buf.data();
}

x::telem::TimeStamp to_timestamp(const mbedtls_x509_time &t) {
    const auto days = date::days_from_civil({
        static_cast<uint16_t>(t.year),
        static_cast<uint8_t>(t.mon),
        static_cast<uint8_t>(t.day),
    });
    return x::telem::TimeStamp(
        days * x::telem::DAY + t.hour * x::telem::HOUR + t.min * x::telem::MINUTE +
        t.sec * x::telem::SECOND
    );
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

/// @brief checks that the key parses without a password and matches the certificate.
/// @returns the reason as a plain message, or an empty string.
std::string check_key(const Certificate &cert, const std::string &key) {
    Random random;
    if (const int ret = random.seed(); ret != 0) return mbedtls_message(ret);
    mbedtls_x509_crt crt;
    mbedtls_x509_crt_init(&crt);
    mbedtls_pk_context pk;
    mbedtls_pk_init(&pk);
    const x::defer::defer free_all([&] {
        mbedtls_pk_free(&pk);
        mbedtls_x509_crt_free(&crt);
    });
    const int crt_ret = mbedtls_x509_crt_parse_der(
        &crt,
        reinterpret_cast<const unsigned char *>(cert.der().data()),
        cert.der().size()
    );
    if (crt_ret != 0) return mbedtls_message(crt_ret);
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
        return "The private key needs a password, which the Driver does not support.";
    if (key_ret != 0)
        return "Failed to parse the private key: " + mbedtls_message(key_ret);
    if (mbedtls_pk_check_pair(&crt.pk, &pk, mbedtls_ctr_drbg_random, &random.drbg) != 0)
        return "The private key does not match the certificate.";
    return "";
}
}

x::json::json Info::to_json() const {
    return {
        {"pem", this->pem},
        {"thumbprint", this->thumbprint},
        {"subject", this->subject},
        {"not_after", this->not_after.iso8601()},
    };
}

bool Certificate::is_pem(const std::string_view data) {
    const auto start = data.find_first_not_of(" \t\r\n");
    return start != std::string_view::npos &&
           data.substr(start).starts_with(PEM_HEADER);
}

std::pair<Certificate, x::errors::Error>
Certificate::parse(const std::string_view data) {
    mbedtls_x509_crt crt;
    mbedtls_x509_crt_init(&crt);
    // mbedTLS reads PEM only with its terminating null byte in the length.
    const std::string buf(data);
    const auto len = is_pem(data) ? buf.size() + 1 : buf.size();
    if (const int ret = mbedtls_x509_crt_parse(
            &crt,
            reinterpret_cast<const unsigned char *>(buf.c_str()),
            len
        );
        ret != 0) {
        mbedtls_x509_crt_free(&crt);
        return {
            Certificate(),
            x::errors::Error(
                errors::INVALID_CERTIFICATE,
                "Failed to parse certificate: " + mbedtls_message(ret)
            )
        };
    }
    Certificate cert;
    cert.der_bytes.assign(reinterpret_cast<const char *>(crt.raw.p), crt.raw.len);
    std::array<char, 1024> subject{};
    if (mbedtls_x509_dn_gets(subject.data(), subject.size(), &crt.subject) > 0)
        cert.subject_name = subject.data();
    for (const auto *san = &crt.subject_alt_names; san != nullptr; san = san->next)
        if (san->buf.tag == SAN_URI_TAG) {
            cert.uri.assign(reinterpret_cast<const char *>(san->buf.p), san->buf.len);
            break;
        }
    cert.valid_from = to_timestamp(crt.valid_from);
    cert.valid_to = to_timestamp(crt.valid_to);
    mbedtls_x509_crt_free(&crt);
    return {std::move(cert), x::errors::NIL};
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
    std::array<unsigned char, 20> hash{};
    mbedtls_sha1(
        reinterpret_cast<const unsigned char *>(this->der_bytes.data()),
        this->der_bytes.size(),
        hash.data()
    );
    static constexpr std::string_view HEX = "0123456789ABCDEF";
    std::string out;
    out.reserve(hash.size() * 2);
    for (const auto b: hash) {
        out.push_back(HEX[b >> 4]);
        out.push_back(HEX[b & 0x0F]);
    }
    return out;
}

bool Certificate::valid_at(const x::telem::TimeStamp time) const {
    return time >= this->valid_from && time <= this->valid_to;
}

Info Certificate::info() const {
    return Info{
        .pem = this->pem(),
        .thumbprint = this->thumbprint(),
        .subject = this->subject_name,
        .not_after = this->valid_to,
    };
}

std::pair<Identity, x::errors::Error> Identity::load(
    const std::filesystem::path &certificate,
    const std::filesystem::path &key
) {
    auto [cert_data, cert_err] = x::fs::read_file(certificate.string());
    if (cert_err)
        return {
            Identity{},
            x::errors::Error(
                errors::CLIENT_CERTIFICATE_INVALID,
                "Failed to read " + certificate.string() + ". The path must exist " +
                    "on the computer that runs the Driver."
            )
        };
    auto [cert, parse_err] = Certificate::parse(cert_data);
    if (parse_err)
        return {
            Identity{},
            x::errors::Error(
                errors::CLIENT_CERTIFICATE_INVALID,
                certificate.string() + ": " + parse_err.data
            )
        };
    auto [key_data, key_err] = x::fs::read_file(key.string());
    if (key_err)
        return {
            Identity{},
            x::errors::Error(
                errors::CLIENT_CERTIFICATE_INVALID,
                "Failed to read " + key.string() + ". The path must exist on the " +
                    "computer that runs the Driver."
            )
        };
    if (const auto reason = check_key(cert, key_data); !reason.empty())
        return {
            Identity{},
            x::errors::Error(
                errors::CLIENT_CERTIFICATE_INVALID,
                key.string() + ": " + reason
            )
        };
    return {
        Identity{.certificate = std::move(cert), .private_key = key_data},
        x::errors::NIL
    };
}

std::pair<Identity, x::errors::Error> create(
    const std::string &application_uri,
    const std::string &common_name,
    const std::string &host,
    const x::telem::TimeSpan validity
) {
    const std::string cn = "CN=" + common_name;
    const std::string uri = "URI:" + application_uri;
    const std::string dns = "DNS:" + host;
    const std::array subject{UA_STRING(const_cast<char *>(cn.c_str()))};
    std::vector san{UA_STRING(const_cast<char *>(uri.c_str()))};
    if (!host.empty()) san.push_back(UA_STRING(const_cast<char *>(dns.c_str())));

    UA_UInt16 key_size = 2048;
    auto days = static_cast<UA_UInt16>(validity.days());
    UA_KeyValuePair params_buf[2];
    params_buf[0].key = UA_QUALIFIEDNAME(0, const_cast<char *>("key-size-bits"));
    UA_Variant_setScalar(&params_buf[0].value, &key_size, &UA_TYPES[UA_TYPES_UINT16]);
    params_buf[1].key = UA_QUALIFIEDNAME(0, const_cast<char *>("expires-in-days"));
    UA_Variant_setScalar(&params_buf[1].value, &days, &UA_TYPES[UA_TYPES_UINT16]);
    UA_KeyValueMap params{2, params_buf};

    UA_Logger logger{};
    logger.log =
        [](void *, UA_LogLevel, UA_LogCategory, const char *msg, va_list args) {
            std::array<char, 1024> buf{};
            UA_String out{
                .length = buf.size(),
                .data = reinterpret_cast<UA_Byte *>(buf.data()),
            };
            UA_String_vformat(&out, msg, args);
            VLOG(1) << "[opc.certificate] " << buf.data();
        };
    UA_ByteString key = UA_BYTESTRING_NULL;
    UA_ByteString cert_data = UA_BYTESTRING_NULL;
    const auto status = UA_CreateCertificate(
        &logger,
        subject.data(),
        subject.size(),
        san.data(),
        san.size(),
        UA_CERTIFICATEFORMAT_PEM,
        &params,
        &key,
        &cert_data
    );
    if (status != UA_STATUSCODE_GOOD)
        return {
            Identity{},
            x::errors::Error(
                errors::DRIVER_CERTIFICATE_FAILED,
                std::string("Failed to create certificate: ") +
                    UA_StatusCode_name(status)
            )
        };
    const std::string key_pem(reinterpret_cast<char *>(key.data), key.length);
    const std::string cert_pem(
        reinterpret_cast<char *>(cert_data.data),
        cert_data.length
    );
    UA_ByteString_clear(&key);
    UA_ByteString_clear(&cert_data);
    auto [cert, err] = Certificate::parse(cert_pem);
    if (err) return {Identity{}, err};
    return {
        Identity{.certificate = std::move(cert), .private_key = key_pem},
        x::errors::NIL
    };
}
}
