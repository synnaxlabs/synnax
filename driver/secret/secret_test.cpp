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
#include <memory>
#include <string>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/fs/fs.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/x509/testutil/testutil.h"
#include "x/cpp/x509/x509.h"

#include "driver/secret/secret.h"

namespace driver::secret { namespace {
void write(const std::filesystem::path &path, const std::string &data) {
    std::ofstream out(path, std::ios::binary);
    out << data;
}

std::string unique(const std::string &name) {
    return name + " " + x::uuid::create().to_string();
}

class ReaderTest : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client;
    synnax::rack::Rack rack;
    std::filesystem::path dir;
    std::filesystem::path secrets;

    void SetUp() override {
        this->client = std::make_shared<synnax::Synnax>(new_test_client());
        this->rack = this->create_rack();
        this->dir = std::filesystem::temp_directory_path() /
                    ("secret-reader-" + x::uuid::create().to_string());
        this->secrets = this->dir / "secrets";
        std::filesystem::create_directories(this->secrets);
    }

    void TearDown() override { std::filesystem::remove_all(this->dir); }

    [[nodiscard]] synnax::rack::Rack create_rack() const {
        auto r = synnax::rack::Rack{.name = unique("reader rack")};
        EXPECT_NIL(this->client->racks.create(r));
        return r;
    }

    /// @returns a reader that signs in as r and reads files from the secrets folder,
    /// or from no folder when secrets_dir is false.
    [[nodiscard]] Reader
    reader(const synnax::rack::Rack &r, const bool secrets_dir = true) const {
        const auto key = ASSERT_NIL_P(
            this->client->api_keys.create("reader", {"rack", std::to_string(r.key)})
        );
        auto cfg = test_client_config();
        cfg.api_key = key.token;
        return Reader(
            std::make_shared<synnax::Synnax>(cfg),
            Config{
                .secrets_dir = secrets_dir ? this->secrets : std::filesystem::path{},
                .rack = r.key,
            }
        );
    }

    /// @returns a stored certificate of x::x509::testutil::CERTIFICATE, whose private
    /// key secret holds x::x509::testutil::KEY.
    [[nodiscard]] synnax::certificate::StoredCertificate
    create_stored_certificate() const {
        synnax::certificate::Certificate c = synnax::certificate::StoredCertificate{};
        auto &stored = std::get<synnax::certificate::StoredCertificate>(c);
        stored.name = unique("reader");
        stored.pem = x::x509::testutil::CERTIFICATE;
        EXPECT_NIL(this->client->certificates.set(c, x::x509::testutil::KEY));
        return std::get<synnax::certificate::StoredCertificate>(c);
    }

    /// @returns a stored secret that holds x::x509::testutil::KEY. The C++ client
    /// writes secrets only as the private keys of stored certificates.
    [[nodiscard]] synnax::secret::Key create_secret() const {
        return this->create_stored_certificate().private_key.value();
    }

    /// @brief creates a task on r that uses the secret with key.
    void use(synnax::rack::Rack &r, const synnax::secret::Key &key) const {
        auto t = synnax::task::Task{
            .name = unique("alerts"),
            .type = "pagerduty_alert",
            .config = {{"auto_start", false}, {"routing_key", key.to_string()}},
        };
        ASSERT_NIL(r.tasks.create(t));
    }

    /// @returns a file certificate of r whose certificate file holds pem, and whose
    /// key file in the secrets folder holds x::x509::testutil::KEY.
    [[nodiscard]] synnax::certificate::Key
    create_file_certificate(const synnax::rack::Rack &r, const std::string &pem) const {
        write(this->dir / "server.pem", pem);
        write(this->secrets / "server-key.pem", x::x509::testutil::KEY);
        synnax::certificate::Certificate c = synnax::certificate::FileCertificate{};
        auto &file = std::get<synnax::certificate::FileCertificate>(c);
        file.name = "server";
        file.certificate_path = (this->dir / "server.pem").string();
        file.private_key_file = "server-key.pem";
        file.rack = r.key;
        EXPECT_NIL(this->client->certificates.set(c));
        return std::get<synnax::certificate::FileCertificate>(c).key;
    }

    [[nodiscard]] std::string sha256(const synnax::certificate::Key &key) const {
        const auto c = ASSERT_NIL_P(this->client->certificates.retrieve(key));
        const auto &details = std::get<synnax::certificate::FileCertificate>(c).details;
        return details.has_value() ? details->sha256 : "";
    }
};

/// @brief it should read a stored secret that a task on the rack uses.
TEST_F(ReaderTest, ReadsSecretThatTaskUses) {
    const auto key = this->create_secret();
    this->use(this->rack, key);
    const auto values = ASSERT_NIL_P(this->reader(this->rack).read({key}));
    ASSERT_EQ(values.at(key), x::x509::testutil::KEY);
}

/// @brief it should refuse a stored secret that only a task on another rack uses.
TEST_F(ReaderTest, RefusesSecretOfAnotherRack) {
    const auto key = this->create_secret();
    this->use(this->rack, key);
    ASSERT_OCCURRED_AS_P(
        this->reader(this->create_rack()).read({key}),
        synnax::auth::ERR_ACCESS_DENIED
    );
}

/// @brief it should read a stored secret from a grant, and refuse one the grant does
/// not hold.
TEST_F(ReaderTest, ReadsSecretFromGrant) {
    const auto key = this->create_secret();
    const auto r = this->reader(this->rack);
    const auto values = ASSERT_NIL_P(r.read({key}, synnax::secret::Values{{key, "v"}}));
    ASSERT_EQ(values.at(key), "v");
    const auto [missing, err] = r.read({key}, synnax::secret::Values{});
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    ASSERT_EQ(err.data, "The grant does not hold secret " + key.to_string() + ".");
}

/// @brief it should return NOT_FOUND for a key that no secret has.
TEST_F(ReaderTest, MissingSecret) {
    ASSERT_OCCURRED_AS_P(
        this->reader(this->rack).read({x::uuid::create()}),
        x::errors::NOT_FOUND
    );
}

/// @brief it should read a file certificate, its key from the secrets folder, and save
/// its details.
TEST_F(ReaderTest, ReadsFileCertificate) {
    const auto key = this->create_file_certificate(
        this->rack,
        x::x509::testutil::CERTIFICATE
    );
    ASSERT_EQ(this->sha256(key), "");
    const auto c = ASSERT_NIL_P(this->reader(this->rack).read_certificate(key));
    ASSERT_EQ(c.pem, x::x509::testutil::CERTIFICATE);
    ASSERT_EQ(c.private_key, x::x509::testutil::KEY);
    ASSERT_EQ(
        this->sha256(key),
        "7b42752d0bb377f9c35ac660d491fa8fa46e5b8885fd41a2650f19f8bd52309e"
    );
}

/// @brief it should read a stored certificate, and its private key from a grant.
TEST_F(ReaderTest, ReadsStoredCertificateFromGrant) {
    const auto stored = this->create_stored_certificate();
    const auto private_key = stored.private_key.value();
    const auto c = ASSERT_NIL_P(
        this->reader(this->rack)
            .read_certificate(
                stored.key,
                synnax::secret::Values{{private_key, x::x509::testutil::KEY}}
            )
    );
    ASSERT_EQ(c.pem, x::x509::testutil::CERTIFICATE);
    ASSERT_EQ(c.private_key, x::x509::testutil::KEY);
}

/// @brief it should fail to read a file certificate whose certificate file is missing.
TEST_F(ReaderTest, FailsWithMissingCertificateFile) {
    const auto key = this->create_file_certificate(
        this->rack,
        x::x509::testutil::CERTIFICATE
    );
    std::filesystem::remove(this->dir / "server.pem");
    const auto [c, err] = this->reader(this->rack).read_certificate(key);
    ASSERT_OCCURRED_AS(err, x::fs::NOT_FOUND);
    ASSERT_EQ(
        err.data,
        "Failed to read certificate \"server\": failed to open " +
            (this->dir / "server.pem").string()
    );
    ASSERT_EQ(this->sha256(key), "");
}

/// @brief it should save new details when the fingerprint of the certificate file
/// changes.
TEST_F(ReaderTest, RefreshesChangedCertificate) {
    const auto key = this->create_file_certificate(
        this->rack,
        x::x509::testutil::CERTIFICATE
    );
    const auto r = this->reader(this->rack);
    ASSERT_NIL_P(r.read_certificate(key));
    write(this->dir / "server.pem", x::x509::testutil::OTHER_CERTIFICATE);
    const auto c = ASSERT_NIL_P(r.read_certificate(key));
    ASSERT_EQ(c.pem, x::x509::testutil::OTHER_CERTIFICATE);
    const auto other = ASSERT_NIL_P(
        x::x509::Certificate::parse(x::x509::testutil::OTHER_CERTIFICATE)
    );
    ASSERT_EQ(this->sha256(key), other.sha256());
}

/// @brief it should fail to read a key file when the Driver has no secrets folder.
TEST_F(ReaderTest, FailsWithNoSecretsFolder) {
    const auto key = this->create_file_certificate(
        this->rack,
        x::x509::testutil::CERTIFICATE
    );
    const auto [c, err] = this->reader(this->rack, false).read_certificate(key);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    ASSERT_EQ(
        err.data,
        "Cannot read the private key of certificate \"server\": it is a file, and the "
        "Driver has no secrets folder. Set secrets_dir in the Driver's config."
    );
    ASSERT_EQ(this->sha256(key), "");
}

/// @brief it should refuse to read a key file of another rack.
TEST_F(ReaderTest, RefusesFileOfAnotherRack) {
    const auto key = this->create_file_certificate(
        this->rack,
        x::x509::testutil::CERTIFICATE
    );
    const auto other = this->create_rack();
    const auto [c, err] = this->reader(other).read_certificate(key);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    ASSERT_EQ(
        err.data,
        "Cannot read the private key of certificate \"server\": it is a file of rack " +
            std::to_string(this->rack.key) + ", and this Driver runs rack " +
            std::to_string(other.key) + "."
    );
}
}}
