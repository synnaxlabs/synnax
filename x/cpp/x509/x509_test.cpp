// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>

#include "gtest/gtest.h"

#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/x509/x509.h"

namespace x::x509 {
namespace {
/// @brief a self-signed P-256 certificate for "O=Synnax, CN=Test Client", valid from
/// 2026-01-01 to 2125-12-31, with a URI and a DNS name in its alternative name.
const std::string CERT_A = R"(-----BEGIN CERTIFICATE-----
MIIBhzCCAS2gAwIBAgIUCo7esOasXFQfGXnv+UE0prCzDx0wCgYIKoZIzj0EAwIw
JzEPMA0GA1UECgwGU3lubmF4MRQwEgYDVQQDDAtUZXN0IENsaWVudDAgFw0yNjAx
MDEwMDAwMDBaGA8yMTI1MTIzMTIzNTk1OVowJzEPMA0GA1UECgwGU3lubmF4MRQw
EgYDVQQDDAtUZXN0IENsaWVudDBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABEw8
BF4DCqCo3fOG3vyEr5iCb2/ZdJt4jiuBYWn1bwIl9ic7slEK8o5JVKlfWv5yJwyX
UPVXDGBFuV2Z+vlgvL+jNTAzMDEGA1UdEQQqMCiGG3VybjpzeW5uYXg6dGVzdDpj
ZXJ0aWZpY2F0ZYIJdGVzdC1ob3N0MAoGCCqGSM49BAMCA0gAMEUCIQDZFy41nKIT
t0DFDW3L/CTyBqqQcAnxq1zl2nIphE/9YwIgFYy5vAB7Je/V7TbYLYtdePvcELKN
nNLdp+AYW7gMXR8=
-----END CERTIFICATE-----
)";

/// @brief the private key of CERT_A.
const std::string KEY_A = R"(-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgXsLANwJSgnKYh56a
45Wp6vcgMMl4Pkvp3saAT1dd8sGhRANCAARMPAReAwqgqN3zht78hK+Ygm9v2XSb
eI4rgWFp9W8CJfYnO7JRCvKOSVSpX1r+cicMl1D1VwxgRbldmfr5YLy/
-----END PRIVATE KEY-----
)";

/// @brief KEY_A, encrypted with the password "secret".
const std::string KEY_A_ENCRYPTED = R"(-----BEGIN ENCRYPTED PRIVATE KEY-----
MIHsMFcGCSqGSIb3DQEFDTBKMCkGCSqGSIb3DQEFDDAcBAh6l9s/jp0Z5gICCAAw
DAYIKoZIhvcNAgkFADAdBglghkgBZQMEASoEEP2M3vyz3Hi4XlI+UqITcUsEgZB5
lyPw9zMGjJnpBDMo1aXLYNGlwrACsW44yY+5IMROKxsPmaDaULzSOGhMmAofxLzU
FtePTAVFQPeF2toKSLj7zSFXPdbbXWQfMa3nga+GT6pnxlpBPMrEULsLS0koxnSM
82GxB6ZfywcF6CI1VMiwwclNfaBB2Blw08vyLlWq9o4U4o4guR8AuTPpK5tkp44=
-----END ENCRYPTED PRIVATE KEY-----
)";

/// @brief a certificate with the same names and validity as CERT_A, and another key.
const std::string CERT_B = R"(-----BEGIN CERTIFICATE-----
MIIBhzCCAS2gAwIBAgIUTGjJZ0xaaYJBQFgv6TBHTiS+DFcwCgYIKoZIzj0EAwIw
JzEPMA0GA1UECgwGU3lubmF4MRQwEgYDVQQDDAtUZXN0IENsaWVudDAgFw0yNjAx
MDEwMDAwMDBaGA8yMTI1MTIzMTIzNTk1OVowJzEPMA0GA1UECgwGU3lubmF4MRQw
EgYDVQQDDAtUZXN0IENsaWVudDBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABBWm
0NyFDkPyCL9rC6DNDclSKXX6DXSOTF1YU1sj4S5yi9uU0qR2fH7DM0Z7WNo7U4id
bS4P55szWw7ZLosJziCjNTAzMDEGA1UdEQQqMCiGG3VybjpzeW5uYXg6dGVzdDpj
ZXJ0aWZpY2F0ZYIJdGVzdC1ob3N0MAoGCCqGSM49BAMCA0gAMEUCIHDMPuPZQ6ZC
tmu7DORs3l8SqhMJup0hji1cFem/bYoqAiEApzKRjaxzHxrgPQgIWT2NnHvajpoE
BWGYvh2eKE/z/9s=
-----END CERTIFICATE-----
)";

/// @brief the private key of CERT_B.
const std::string KEY_B = R"(-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQg160R6M902UlqFdVs
cetBa4/zrJdySu6u85NfP93UBWihRANCAAQVptDchQ5D8gi/awugzQ3JUil1+g10
jkxdWFNbI+EucovblNKkdnx+wzNGe1jaO1OInW0uD+ebM1sO2S6LCc4g
-----END PRIVATE KEY-----
)";
}

/// @brief it should read the names, validity, and thumbprint of a PEM certificate.
TEST(CertificateTest, ParsePem) {
    const auto cert = ASSERT_NIL_P(Certificate::parse(CERT_A));
    ASSERT_EQ(cert.subject(), "O=Synnax, CN=Test Client");
    ASSERT_EQ(cert.uri(), "urn:synnax:test:certificate");
    ASSERT_EQ(cert.not_before(), telem::TimeStamp(1767225600000000000));
    ASSERT_EQ(cert.not_after(), telem::TimeStamp(4922899199000000000));
    ASSERT_EQ(cert.thumbprint(), "915D90FFF8FB63289202F5155CE6CBC027CC1C57");
}

/// @brief it should give the same certificate after a PEM or DER round trip.
TEST(CertificateTest, PemRoundTrip) {
    const auto cert = ASSERT_NIL_P(Certificate::parse(CERT_A));
    const auto pem = cert.pem();
    ASSERT_EQ(pem, CERT_A);
    ASSERT_TRUE(Certificate::is_pem(pem));
    ASSERT_EQ(ASSERT_NIL_P(Certificate::parse(pem)), cert);
    ASSERT_EQ(ASSERT_NIL_P(Certificate::parse(cert.der())), cert);
    ASSERT_FALSE(Certificate::is_pem(cert.der()));
}

/// @brief it should find a PEM header after leading white space.
TEST(CertificateTest, IsPemAfterWhitespace) {
    ASSERT_TRUE(Certificate::is_pem(" \t\r\n" + CERT_A));
    ASSERT_FALSE(Certificate::is_pem(""));
    ASSERT_FALSE(Certificate::is_pem("not a certificate"));
}

/// @brief it should keep only the first certificate of a chain.
TEST(CertificateTest, ParseChainKeepsFirst) {
    const auto cert = ASSERT_NIL_P(Certificate::parse(CERT_A + CERT_B));
    ASSERT_EQ(cert, ASSERT_NIL_P(Certificate::parse(CERT_A)));
}

/// @brief it should reject data that holds no certificate.
TEST(CertificateTest, ParseRejectsGarbage) {
    ASSERT_OCCURRED_AS_P(Certificate::parse("not a certificate"), INVALID);
    ASSERT_OCCURRED_AS_P(
        Certificate::parse(
            "-----BEGIN CERTIFICATE-----\nAAAA\n-----END CERTIFICATE-----\n"
        ),
        INVALID
    );
}

/// @brief it should report the validity period.
TEST(CertificateTest, ValidAt) {
    const auto cert = ASSERT_NIL_P(Certificate::parse(CERT_A));
    ASSERT_TRUE(cert.valid_at(telem::TimeStamp::now()));
    ASSERT_TRUE(cert.valid_at(cert.not_before()));
    ASSERT_TRUE(cert.valid_at(cert.not_after()));
    ASSERT_FALSE(cert.valid_at(cert.not_after() + telem::SECOND));
    ASSERT_FALSE(cert.valid_at(cert.not_before() - telem::SECOND));
}

/// @brief it should accept the private key of the certificate.
TEST(CertificateTest, CheckKeyMatches) {
    const auto cert = ASSERT_NIL_P(Certificate::parse(CERT_A));
    ASSERT_NIL(cert.check_key(KEY_A));
}

/// @brief it should reject the private key of another certificate.
TEST(CertificateTest, CheckKeyMismatched) {
    const auto cert = ASSERT_NIL_P(Certificate::parse(CERT_A));
    const auto err = cert.check_key(KEY_B);
    ASSERT_MATCHES(err, KEY_INVALID);
    ASSERT_EQ(err.data, "The private key does not match the certificate.");
}

/// @brief it should reject a private key that needs a password.
TEST(CertificateTest, CheckKeyEncrypted) {
    const auto cert = ASSERT_NIL_P(Certificate::parse(CERT_A));
    const auto err = cert.check_key(KEY_A_ENCRYPTED);
    ASSERT_MATCHES(err, KEY_INVALID);
    ASSERT_EQ(err.data, "The private key needs a password, which is not supported.");
}

/// @brief it should reject data that holds no private key.
TEST(CertificateTest, CheckKeyGarbage) {
    const auto cert = ASSERT_NIL_P(Certificate::parse(CERT_A));
    const auto err = cert.check_key("not a key");
    ASSERT_MATCHES(err, KEY_INVALID);
    ASSERT_TRUE(err.data.starts_with("Failed to parse the private key: ")) << err.data;
}
}
