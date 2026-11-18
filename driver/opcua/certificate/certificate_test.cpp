// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <filesystem>
#include <fstream>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/opcua/certificate/certificate.h"
#include "driver/opcua/errors/errors.h"

namespace driver::opcua::certificate {
namespace {
const std::string URI = "urn:synnax:test:certificate";

void write(const std::filesystem::path &path, const std::string &data) {
    std::ofstream out(path, std::ios::binary);
    out << data;
}

class CertificateTest : public ::testing::Test {
protected:
    /// @brief an identity that the tests share, since making a key takes seconds.
    static inline Identity identity;
    /// @brief an identity with the same name as identity and a different key.
    static inline Identity other;

    std::filesystem::path dir;

    static void SetUpTestSuite() {
        identity = ASSERT_NIL_P(create(URI, "Test Client", "", x::telem::DAY));
        other = ASSERT_NIL_P(create(URI, "Test Client", "", x::telem::DAY));
    }

    void SetUp() override {
        this->dir = std::filesystem::temp_directory_path() /
                    ("opcua-certificate-" + x::uuid::create().to_string());
        std::filesystem::create_directories(this->dir);
    }

    void TearDown() override { std::filesystem::remove_all(this->dir); }
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

/// @brief it should load an identity from certificate and key files.
TEST_F(CertificateTest, LoadFromFiles) {
    write(this->dir / "cert.pem", identity.certificate.pem());
    write(this->dir / "key.pem", identity.private_key);
    const auto loaded = ASSERT_NIL_P(
        Identity::load(this->dir / "cert.pem", this->dir / "key.pem")
    );
    ASSERT_EQ(loaded.certificate, identity.certificate);
    ASSERT_EQ(loaded.private_key, identity.private_key);
}

/// @brief it should load a DER certificate file.
TEST_F(CertificateTest, LoadDerCertificate) {
    write(this->dir / "cert.der", identity.certificate.der());
    write(this->dir / "key.pem", identity.private_key);
    const auto loaded = ASSERT_NIL_P(
        Identity::load(this->dir / "cert.der", this->dir / "key.pem")
    );
    ASSERT_EQ(loaded.certificate, identity.certificate);
}

/// @brief it should reject a certificate file that does not exist.
TEST_F(CertificateTest, LoadMissingCertificate) {
    ASSERT_OCCURRED_AS_P(
        Identity::load(this->dir / "missing.pem", this->dir / "key.pem"),
        errors::CLIENT_CERTIFICATE_INVALID
    );
}

/// @brief it should reject a key file that does not exist.
TEST_F(CertificateTest, LoadMissingKey) {
    write(this->dir / "cert.pem", identity.certificate.pem());
    ASSERT_OCCURRED_AS_P(
        Identity::load(this->dir / "cert.pem", this->dir / "missing.pem"),
        errors::CLIENT_CERTIFICATE_INVALID
    );
}

/// @brief it should reject a key that does not match the certificate.
TEST_F(CertificateTest, LoadMismatchedKey) {
    write(this->dir / "cert.pem", identity.certificate.pem());
    write(this->dir / "key.pem", other.private_key);
    const auto key = this->dir / "key.pem";
    const auto err = Identity::load(this->dir / "cert.pem", key).second;
    ASSERT_MATCHES(err, errors::CLIENT_CERTIFICATE_INVALID);
    ASSERT_EQ(
        err.data,
        key.string() + ": The private key does not match the certificate."
    );
}
}
