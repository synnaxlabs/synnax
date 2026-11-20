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
#include "x/cpp/x509/testutil/testutil.h"
#include "x/cpp/x509/x509.h"

namespace x::x509 {
namespace {
/// @brief the private key of testutil::CERTIFICATE.
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

/// @brief a certificate with the same names and validity as testutil::CERTIFICATE,
/// and another key.
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

/// @brief a certificate whose names hold escaped characters, a T61String, a BMPString,
/// two OU values, two CN values, a multi-valued RDN, and attribute types that Go does
/// not name.
const std::string CERT_NAMES = R"(-----BEGIN CERTIFICATE-----
MIICGzCCAcGgAwIBAgIUB9VfRewbStY6eEYpBTBUmKVyCIwwCgYIKoZIzj0EAwIw
JzERMA8GA1UECgwIUGxhbnQgQ0ExEjAQBgNVBAMMCVJvb3Qgw6kgMTAeFw0yNjAx
MDEwMDAwMDBaFw0yNzAxMDEwMDAwMDBaMIHzMQswCQYDVQQGEwJVUzETMBEGA1UE
CAwKQ2FsaWZvcm5pYTETMBEGA1UEBxQKU8OjbyBQYXVsbzETMBEGA1UECgwKQWNt
ZSwgSW5jLjEMMAoGA1UECwwDRW5nMRcwFQYDVQQLHg4ATwBwAHMAIAA8ADEAPjEV
MBMGCSqGSIb3DQEJARYGYUBiLmNvMSIwDAYDVQQDDAVGaXJzdDASBgoJkiaJk/Is
ZAEBDAR1aWQxMR0wGwYDVQQDDBQgI0xlYWQrdHJhaWw7ICJxIiBcIDELMAkGA1UE
BRMCNDIxFzAVBgoJkiaJk/IsZAEZFgdleGFtcGxlMFkwEwYHKoZIzj0CAQYIKoZI
zj0DAQcDQgAEJJM9P9FoypCiKpA+sf0joTUjsn/8fXtQsMVmfCll58341BpIapc4
BD5Aj1vGkWPhV+AKJu5N5zvKwYyFtlcToDAKBggqhkjOPQQDAgNIADBFAiBA6VB9
rfX6G1JJjjTHxMPj/nkSh0AuTAWmd1ODKJ05yQIhAO05zs9wiDDkc+ob2L3wBKYW
zqJ/caxt+tVnaBYVCv4s
-----END CERTIFICATE-----
)";

/// @brief a certificate valid from 1960-01-01 to 9999-12-31, outside the range of a
/// TimeStamp.
const std::string CERT_WIDE = R"(-----BEGIN CERTIFICATE-----
MIIBITCByKADAgECAhQrFoZ0JsEpQhxVZ0sKXm2nvdQRRzAKBggqhkjOPQQDAjAQ
MQ4wDAYDVQQDDAVSYW5nZTAgFw02MDAxMDEwMDAwMDBaGA85OTk5MTIzMTIzNTk1
OVowEDEOMAwGA1UEAwwFUmFuZ2UwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAARb
wGFsINPokD/i+qd4xDFk8yw63w+aZaWie9VImzRjgnGx1iA8gXWcJW6imfSRIY6V
a44FT9kAWJH42k97G0sIMAoGCCqGSM49BAMCA0gAMEUCIQChfz9Ge4JjVsIixuXe
m5WVCIRhrUXR2YFLeitIWiKZFgIgJkv8d2zRLbW6Vf+Vd5u8fUSVUazIkA2tBdLK
FbGJbLw=
-----END CERTIFICATE-----
)";

/// @brief a certificate whose subject holds values that are not text: a BIT STRING and
/// a UniversalString of unknown types, and a BIT STRING for O.
const std::string CERT_VALUES = R"(-----BEGIN CERTIFICATE-----
MIIBfDCCASOgAwIBAgIBBzAKBggqhkjOPQQDAjBIMQ8wDQYDVQQDEwZ2YWx1ZXMx
CzAJBgMqAwQDAgeAMQ0wCwYDKgMFHAQAAABBMQswCQYDVQQKAwIHgDEMMAoGAyoD
BxYDaWE1MB4XDTI2MDEwMTAwMDAwMFoXDTI3MDEwMTAwMDAwMFowSDEPMA0GA1UE
AxMGdmFsdWVzMQswCQYDKgMEAwIHgDENMAsGAyoDBRwEAAAAQTELMAkGA1UECgMC
B4AxDDAKBgMqAwcWA2lhNTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABJxOCBU2
j2WhizBYenYrDLiPyDpk9CJn0pCu76QfzwOHqDof0EEDoEJ7HsjaFwWsvN43TA4V
HAdzwWtaDe9txt8wCgYIKoZIzj0EAwIDRwAwRAIgW6bitmeh7G2crRAhhNXCI0CJ
w+edEQUKCRIq0WNuqNQCIHCuCQTfnoU/SOPymY4QWEbkGNnAtbVukxUcHKEQI011
-----END CERTIFICATE-----
)";

/// @brief a certificate whose subject holds a UTF8String that is not valid UTF-8.
const std::string CERT_BAD_UTF8 = R"(-----BEGIN CERTIFICATE-----
MIIBKjCB0aADAgECAgEHMAoGCCqGSM49BAMCMB8xEDAOBgNVBAMTB2JhZHV0Zjgx
CzAJBgNVBAoMAmH/MB4XDTI2MDEwMTAwMDAwMFoXDTI3MDEwMTAwMDAwMFowHzEQ
MA4GA1UEAxMHYmFkdXRmODELMAkGA1UECgwCYf8wWTATBgcqhkjOPQIBBggqhkjO
PQMBBwNCAAQEJRpbpUnksYbGZeU8BVh1vrIRV4MfxBHU14HPNkLEjzUouSTtddfN
vKDj4STNfDjCXnplcH9mwaJkVA1+IdpoMAoGCCqGSM49BAMCA0gAMEUCIGPutBaD
lMq7daL4Yd2ZzjEg6sbaFUhwneVfE8lxRd+3AiEAt6TfnSIAPVDAtVGxHrAxSBja
gh3RZClLMTaDcCidbus=
-----END CERTIFICATE-----
)";

/// @brief a certificate whose subject holds a BMPString with a surrogate pair.
const std::string CERT_BAD_BMP = R"(-----BEGIN CERTIFICATE-----
MIIBKzCB06ADAgECAgEHMAoGCCqGSM49BAMCMCAxDzANBgNVBAMTBmJhZGJtcDEN
MAsGA1UECh4E2D3eADAeFw0yNjAxMDEwMDAwMDBaFw0yNzAxMDEwMDAwMDBaMCAx
DzANBgNVBAMTBmJhZGJtcDENMAsGA1UECh4E2D3eADBZMBMGByqGSM49AgEGCCqG
SM49AwEHA0IABO6bl2Q9qM6/tzxEbUSCkFaQL50cSbZEWKnzP+smOOfXHXPGL+yg
NSaRzEx68mr2P89mQpkCtzQqMO4eSX2H65swCgYIKoZIzj0EAwIDRwAwRAIgWejw
YodVMp8wh5nLPbwVeHHA5S1WL0mylnd/QF6IzvACIH1/vLFnpK7LRbV8ImCyT7iP
W9WtvFisAUytADsOcOuK
-----END CERTIFICATE-----
)";

/// @brief a certificate whose subject and issuer hold CN=PLC_01 as a PrintableString,
/// which Go refuses for its underscore.
const std::string CERT_PLC = R"(-----BEGIN CERTIFICATE-----
MIIBDzCBtaADAgECAgEHMAoGCCqGSM49BAMCMBExDzANBgNVBAMTBlBMQ18wMTAe
Fw0yNjAxMDEwMDAwMDBaFw0yNzAxMDEwMDAwMDBaMBExDzANBgNVBAMTBlBMQ18w
MTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABP6ShaQQAsVB4MXyrQhjY+i5TB2Q
mw0Kp2bTPGU1kbxeQ+YRsgu81HVE/oFLwg2RKKFplP5nX2hEqVdJ6eUzT/AwCgYI
KoZIzj0EAwIDSQAwRgIhAM6hUXFU3y33cK28Rggw0qUI/ESTeMYk3ruOD4Vi6f1m
AiEA61+Cv4ctjunZDcLqhyRWv1PVyPldZ5/gAOXI3QAozQs=
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

/// @brief it should read the names, validity, and fingerprints of a PEM certificate.
TEST(CertificateTest, ParsePem) {
    const auto cert = ASSERT_NIL_P(Certificate::parse(testutil::CERTIFICATE));
    ASSERT_EQ(cert.subject(), "CN=Test Client,O=Synnax");
    ASSERT_EQ(cert.issuer(), "CN=Test Client,O=Synnax");
    ASSERT_EQ(cert.uri(), "urn:synnax:test:certificate");
    ASSERT_EQ(cert.not_before(), telem::TimeStamp(1767225600000000000));
    ASSERT_EQ(cert.not_after(), telem::TimeStamp(4922899199000000000));
    ASSERT_EQ(cert.thumbprint(), "915D90FFF8FB63289202F5155CE6CBC027CC1C57");
    ASSERT_EQ(cert.sha1(), "915d90fff8fb63289202f5155ce6cbc027cc1c57");
    ASSERT_EQ(
        cert.sha256(),
        "7b42752d0bb377f9c35ac660d491fa8fa46e5b8885fd41a2650f19f8bd52309e"
    );
}

/// @brief it should give the subject and issuer as Go's pkix.Name.String() does.
TEST(CertificateTest, NamesMatchCore) {
    const auto cert = ASSERT_NIL_P(Certificate::parse(CERT_NAMES));
    ASSERT_EQ(
        cert.subject(),
        "SERIALNUMBER=42,CN=\\ #Lead\\+trail\\; \\\"q\\\" \\\\\\ ,"
        "OU=Eng+OU=Ops \\<1\\>,O=Acme\\, Inc.,L=S\xC3\x83\xC2\xA3o Paulo,"
        "ST=California,C=US,0.9.2342.19200300.100.1.25=example,"
        "0.9.2342.19200300.100.1.1=uid1,1.2.840.113549.1.9.1=a@b.co"
    );
    ASSERT_EQ(cert.issuer(), "CN=Root \xC3\xA9 1,O=Plant CA");
}

/// @brief it should give a value that is not text as Go does: the DER as hex for an
/// unknown type, and nothing for a named type.
TEST(CertificateTest, NonTextValuesMatchCore) {
    const auto cert = ASSERT_NIL_P(Certificate::parse(CERT_VALUES));
    ASSERT_EQ(
        cert.subject(),
        "CN=values,1.2.3.7=ia5,1.2.3.5=#1c0400000041,1.2.3.4=#03020780"
    );
}

/// @brief it should print a name value that Go refuses as its bytes when they are
/// UTF-8, and leave it out when they are not.
TEST(CertificateTest, NamesGoRefuses) {
    const auto plc = ASSERT_NIL_P(Certificate::parse(CERT_PLC));
    ASSERT_EQ(plc.subject(), "CN=PLC_01");
    ASSERT_EQ(plc.issuer(), "CN=PLC_01");
    ASSERT_EQ(ASSERT_NIL_P(Certificate::parse(CERT_BAD_UTF8)).subject(), "CN=badutf8");
    ASSERT_EQ(ASSERT_NIL_P(Certificate::parse(CERT_BAD_BMP)).subject(), "CN=badbmp");
}

/// @brief it should clamp the validity period to the range of a TimeStamp, as the
/// Core does.
TEST(CertificateTest, ValidityClamps) {
    const auto cert = ASSERT_NIL_P(Certificate::parse(CERT_WIDE));
    ASSERT_EQ(cert.not_before(), telem::TimeStamp(0));
    ASSERT_EQ(cert.not_after(), telem::TimeStamp::max());
}

/// @brief it should give the same certificate after a PEM or DER round trip.
TEST(CertificateTest, PemRoundTrip) {
    const auto cert = ASSERT_NIL_P(Certificate::parse(testutil::CERTIFICATE));
    const auto pem = cert.pem();
    ASSERT_EQ(pem, testutil::CERTIFICATE);
    ASSERT_TRUE(Certificate::is_pem(pem));
    ASSERT_EQ(ASSERT_NIL_P(Certificate::parse(pem)), cert);
    ASSERT_EQ(ASSERT_NIL_P(Certificate::parse(cert.der())), cert);
    ASSERT_FALSE(Certificate::is_pem(cert.der()));
}

/// @brief it should find a PEM header after leading white space.
TEST(CertificateTest, IsPemAfterWhitespace) {
    ASSERT_TRUE(Certificate::is_pem(" \t\r\n" + testutil::CERTIFICATE));
    ASSERT_FALSE(Certificate::is_pem(""));
    ASSERT_FALSE(Certificate::is_pem("not a certificate"));
}

/// @brief it should parse a PEM certificate block after other text or blocks, such as
/// the Bag Attributes that openssl pkcs12 writes, a UTF-8 byte order mark, or a key.
TEST(CertificateTest, ParsePemAfterOtherContent) {
    const auto expected = ASSERT_NIL_P(Certificate::parse(testutil::CERTIFICATE));
    for (const auto &data: {
             "Bag Attributes\n    localKeyID: 01 00 00 00 \nsubject=/CN=Test Client\n" +
                 testutil::CERTIFICATE,
             "\xEF\xBB\xBF" + testutil::CERTIFICATE,
             KEY_A + testutil::CERTIFICATE,
         }) {
        ASSERT_TRUE(Certificate::is_pem(data));
        ASSERT_EQ(ASSERT_NIL_P(Certificate::parse(data)), expected);
    }
}

/// @brief it should find each certificate block in order, and skip other blocks, text,
/// and a block with no footer.
TEST(CertificateTest, PemBlocks) {
    const auto data = "text\n" + KEY_A + testutil::CERTIFICATE + CERT_B +
                      "-----BEGIN CERTIFICATE-----\nAAAA\n";
    const auto blocks = Certificate::pem_blocks(data);
    ASSERT_EQ(blocks.size(), 2);
    ASSERT_EQ(std::string(blocks[0]) + "\n", testutil::CERTIFICATE);
    ASSERT_EQ(std::string(blocks[1]) + "\n", CERT_B);
    ASSERT_TRUE(Certificate::pem_blocks(KEY_A).empty());
}

/// @brief it should keep only the first certificate of a chain, and skip a later block
/// that is not valid.
TEST(CertificateTest, ParseChainKeepsFirst) {
    const auto expected = ASSERT_NIL_P(Certificate::parse(testutil::CERTIFICATE));
    ASSERT_EQ(
        ASSERT_NIL_P(Certificate::parse(testutil::CERTIFICATE + CERT_B)),
        expected
    );
    ASSERT_EQ(
        ASSERT_NIL_P(
            Certificate::parse(
                testutil::CERTIFICATE +
                "-----BEGIN CERTIFICATE-----\nAAAA\n-----END CERTIFICATE-----\n"
            )
        ),
        expected
    );
}

/// @brief it should treat a certificate with no footer as PEM, and fail to parse it.
TEST(CertificateTest, TruncatedPem) {
    const auto truncated = testutil::CERTIFICATE.substr(
        0,
        testutil::CERTIFICATE.find("-----END")
    );
    ASSERT_TRUE(Certificate::is_pem(truncated));
    ASSERT_OCCURRED_AS_P(Certificate::parse(truncated), INVALID);
}

/// @brief it should read DER that holds PEM text as DER, so that a forged certificate
/// cannot pass for the certificate that it holds.
TEST(CertificateTest, DerHoldingPem) {
    const auto der = testutil::create_der_holding(KEY_A, "urn:holder", CERT_B);
    ASSERT_TRUE(der.contains("-----BEGIN CERTIFICATE-----"));
    ASSERT_FALSE(Certificate::is_pem(der));
    ASSERT_TRUE(Certificate::pem_blocks(der).empty());
    const auto cert = ASSERT_NIL_P(Certificate::parse(der));
    ASSERT_EQ(cert.der(), der);
    ASSERT_NE(cert, ASSERT_NIL_P(Certificate::parse(CERT_B)));
}

/// @brief it should read text that starts with "0" as PEM, though DER starts with the
/// same byte.
TEST(CertificateTest, PemAfterLeadingZero) {
    const auto data = "0 comes first\n" + testutil::CERTIFICATE;
    ASSERT_TRUE(Certificate::is_pem(data));
    ASSERT_EQ(
        ASSERT_NIL_P(Certificate::parse(data)),
        ASSERT_NIL_P(Certificate::parse(testutil::CERTIFICATE))
    );
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
    const auto cert = ASSERT_NIL_P(Certificate::parse(testutil::CERTIFICATE));
    ASSERT_TRUE(cert.valid_at(telem::TimeStamp::now()));
    ASSERT_TRUE(cert.valid_at(cert.not_before()));
    ASSERT_TRUE(cert.valid_at(cert.not_after()));
    ASSERT_FALSE(cert.valid_at(cert.not_after() + telem::SECOND));
    ASSERT_FALSE(cert.valid_at(cert.not_before() - telem::SECOND));
}

/// @brief it should accept the private key of the certificate.
TEST(CertificateTest, CheckKeyMatches) {
    const auto cert = ASSERT_NIL_P(Certificate::parse(testutil::CERTIFICATE));
    ASSERT_NIL(cert.check_key(KEY_A));
}

/// @brief it should reject the private key of another certificate.
TEST(CertificateTest, CheckKeyMismatched) {
    const auto cert = ASSERT_NIL_P(Certificate::parse(testutil::CERTIFICATE));
    const auto err = cert.check_key(KEY_B);
    ASSERT_MATCHES(err, KEY_INVALID);
    ASSERT_EQ(err.data, "The private key does not match the certificate.");
}

/// @brief it should reject a private key that needs a password.
TEST(CertificateTest, CheckKeyEncrypted) {
    const auto cert = ASSERT_NIL_P(Certificate::parse(testutil::CERTIFICATE));
    const auto err = cert.check_key(KEY_A_ENCRYPTED);
    ASSERT_MATCHES(err, KEY_INVALID);
    ASSERT_EQ(err.data, "The private key needs a password, which is not supported.");
}

/// @brief it should reject data that holds no private key.
TEST(CertificateTest, CheckKeyGarbage) {
    const auto cert = ASSERT_NIL_P(Certificate::parse(testutil::CERTIFICATE));
    const auto err = cert.check_key("not a key");
    ASSERT_MATCHES(err, KEY_INVALID);
    ASSERT_TRUE(err.data.starts_with("Failed to parse the private key: ")) << err.data;
}
}
