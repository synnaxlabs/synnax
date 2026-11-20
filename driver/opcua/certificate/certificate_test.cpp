// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/opcua/certificate/certificate.h"
#include "driver/opcua/errors/errors.h"

namespace driver::opcua::certificate {
namespace {
const std::string URI = "urn:synnax:test:certificate";

class CertificateTest : public ::testing::Test {
protected:
    /// @brief an identity that the tests share, since making a key takes seconds.
    static inline Identity identity;
    /// @brief an identity with the same name as identity and a different key.
    static inline Identity other;

    static void SetUpTestSuite() {
        identity = ASSERT_NIL_P(create(URI, "Test Client", "", x::telem::DAY));
        other = ASSERT_NIL_P(create(URI, "Test Client", "", x::telem::DAY));
    }
};
}

/// @brief it should make a certificate with the URI, name, host, and validity.
TEST_F(CertificateTest, CreateSetsIdentityFields) {
    const auto before = x::telem::TimeStamp::now() - x::telem::MINUTE;
    const auto id = ASSERT_NIL_P(
        create(URI, "Test Client", "test-host", x::telem::DAY)
    );
    const auto &cert = id.certificate;
    ASSERT_EQ(cert.uri(), URI);
    ASSERT_EQ(cert.subject(), "CN=Test Client");
    ASSERT_GE(cert.not_before(), before);
    ASSERT_LE(cert.not_after() - cert.not_before(), x::telem::DAY + x::telem::MINUTE);
    ASSERT_GE(
        cert.not_after() - cert.not_before(),
        x::telem::DAY - x::telem::HOUR * 24
    );
    ASSERT_FALSE(id.private_key.empty());
}

/// @brief it should give two certificates with the same name different thumbprints.
TEST_F(CertificateTest, ThumbprintsDiffer) {
    ASSERT_NE(identity.certificate.thumbprint(), other.certificate.thumbprint());
    ASSERT_FALSE(identity.certificate == other.certificate);
}

/// @brief it should encode the public description as JSON.
TEST_F(CertificateTest, InfoToJson) {
    const auto j = info(identity.certificate).to_json();
    ASSERT_EQ(j["pem"], identity.certificate.pem());
    ASSERT_EQ(j["thumbprint"], identity.certificate.thumbprint());
    ASSERT_EQ(j["subject"], "CN=Test Client");
    ASSERT_EQ(j["not_after"], identity.certificate.not_after().iso8601());
}

/// @brief it should read an identity from a PEM certificate and its key.
TEST_F(CertificateTest, ParsePem) {
    const auto parsed = ASSERT_NIL_P(
        Identity::parse(identity.certificate.pem(), identity.private_key)
    );
    ASSERT_EQ(parsed.certificate, identity.certificate);
    ASSERT_EQ(parsed.private_key, identity.private_key);
}

/// @brief it should read an identity from a DER certificate.
TEST_F(CertificateTest, ParseDer) {
    const auto parsed = ASSERT_NIL_P(
        Identity::parse(identity.certificate.der(), identity.private_key)
    );
    ASSERT_EQ(parsed.certificate, identity.certificate);
}

/// @brief it should reject data that is not a certificate.
TEST_F(CertificateTest, ParseInvalidCertificate) {
    ASSERT_OCCURRED_AS_P(
        Identity::parse("not a certificate", identity.private_key),
        errors::CLIENT_CERTIFICATE_INVALID
    );
}

/// @brief it should reject a certificate with no private key.
TEST_F(CertificateTest, ParseNoKey) {
    const auto err = Identity::parse(identity.certificate.pem(), "").second;
    ASSERT_MATCHES(err, errors::CLIENT_CERTIFICATE_INVALID);
    ASSERT_EQ(err.data, "The client certificate has no private key.");
}

/// @brief it should reject a key that does not match the certificate.
TEST_F(CertificateTest, ParseMismatchedKey) {
    const auto err = Identity::parse(identity.certificate.pem(), other.private_key)
                         .second;
    ASSERT_MATCHES(err, errors::CLIENT_CERTIFICATE_INVALID);
    ASSERT_EQ(err.data, "The private key does not match the certificate.");
}
}
