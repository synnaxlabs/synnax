// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <array>
#include <string>

#include "gtest/gtest.h"
#include "mbedtls/ctr_drbg.h"
#include "mbedtls/entropy.h"
#include "mbedtls/oid.h"
#include "mbedtls/pk.h"
#include "mbedtls/x509_crt.h"

#include "x/cpp/x509/testutil/testutil.h"

namespace x::x509::testutil {
namespace {
// 1.2.3.4, an OID that no parser knows.
constexpr std::array<char, 3> UNKNOWN_OID = {0x2A, 0x03, 0x04};
}

std::string create_der_holding(
    const std::string &key,
    const std::string &uri,
    const std::string &pem
) {
    mbedtls_entropy_context entropy;
    mbedtls_ctr_drbg_context drbg;
    mbedtls_pk_context pk;
    mbedtls_x509write_cert crt;
    mbedtls_entropy_init(&entropy);
    mbedtls_ctr_drbg_init(&drbg);
    mbedtls_pk_init(&pk);
    mbedtls_x509write_crt_init(&crt);
    std::string der;
    [&] {
        ASSERT_EQ(
            mbedtls_ctr_drbg_seed(&drbg, mbedtls_entropy_func, &entropy, nullptr, 0),
            0
        );
        ASSERT_EQ(
            mbedtls_pk_parse_key(
                &pk,
                reinterpret_cast<const unsigned char *>(key.c_str()),
                key.size() + 1,
                nullptr,
                0,
                mbedtls_ctr_drbg_random,
                &drbg
            ),
            0
        );
        mbedtls_x509write_crt_set_version(&crt, MBEDTLS_X509_CRT_VERSION_3);
        mbedtls_x509write_crt_set_md_alg(&crt, MBEDTLS_MD_SHA256);
        mbedtls_x509write_crt_set_subject_key(&crt, &pk);
        mbedtls_x509write_crt_set_issuer_key(&crt, &pk);
        ASSERT_EQ(mbedtls_x509write_crt_set_subject_name(&crt, "CN=Holder"), 0);
        ASSERT_EQ(mbedtls_x509write_crt_set_issuer_name(&crt, "CN=Holder"), 0);
        std::array<unsigned char, 1> serial = {1};
        ASSERT_EQ(
            mbedtls_x509write_crt_set_serial_raw(&crt, serial.data(), serial.size()),
            0
        );
        ASSERT_EQ(
            mbedtls_x509write_crt_set_validity(
                &crt,
                "20260101000000",
                "21251231235959"
            ),
            0
        );
        ASSERT_EQ(mbedtls_x509write_crt_set_basic_constraints(&crt, 0, -1), 0);
        ASSERT_EQ(
            mbedtls_x509write_crt_set_key_usage(
                &crt,
                MBEDTLS_X509_KU_DIGITAL_SIGNATURE | MBEDTLS_X509_KU_NON_REPUDIATION |
                    MBEDTLS_X509_KU_KEY_ENCIPHERMENT | MBEDTLS_X509_KU_DATA_ENCIPHERMENT
            ),
            0
        );
        mbedtls_asn1_sequence client_auth{};
        client_auth.buf.tag = MBEDTLS_ASN1_OID;
        client_auth.buf.p = reinterpret_cast<unsigned char *>(
            const_cast<char *>(MBEDTLS_OID_CLIENT_AUTH)
        );
        client_auth.buf.len = MBEDTLS_OID_SIZE(MBEDTLS_OID_CLIENT_AUTH);
        mbedtls_asn1_sequence usages{};
        usages.buf.tag = MBEDTLS_ASN1_OID;
        usages.buf.p = reinterpret_cast<unsigned char *>(
            const_cast<char *>(MBEDTLS_OID_SERVER_AUTH)
        );
        usages.buf.len = MBEDTLS_OID_SIZE(MBEDTLS_OID_SERVER_AUTH);
        usages.next = &client_auth;
        ASSERT_EQ(mbedtls_x509write_crt_set_ext_key_usage(&crt, &usages), 0);
        mbedtls_x509_san_list san{};
        san.node.type = MBEDTLS_X509_SAN_UNIFORM_RESOURCE_IDENTIFIER;
        san.node.san.unstructured_name.p = reinterpret_cast<unsigned char *>(
            const_cast<char *>(uri.data())
        );
        san.node.san.unstructured_name.len = uri.size();
        ASSERT_EQ(mbedtls_x509write_crt_set_subject_alternative_name(&crt, &san), 0);
        ASSERT_EQ(
            mbedtls_x509write_crt_set_extension(
                &crt,
                UNKNOWN_OID.data(),
                UNKNOWN_OID.size(),
                0,
                reinterpret_cast<const unsigned char *>(pem.data()),
                pem.size()
            ),
            0
        );
        std::array<unsigned char, 8192> buf{};
        const int len = mbedtls_x509write_crt_der(
            &crt,
            buf.data(),
            buf.size(),
            mbedtls_ctr_drbg_random,
            &drbg
        );
        ASSERT_GT(len, 0);
        der.assign(reinterpret_cast<const char *>(buf.data() + buf.size() - len), len);
    }();
    mbedtls_x509write_crt_free(&crt);
    mbedtls_pk_free(&pk);
    mbedtls_ctr_drbg_free(&drbg);
    mbedtls_entropy_free(&entropy);
    return der;
}
}
