// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <variant>

#include "gtest/gtest.h"

#include "client/cpp/certificate/certificate.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"
#include "x/cpp/x509/testutil/testutil.h"

namespace synnax::certificate {
namespace {
std::string unique(const std::string &name) {
    return name + " " + x::uuid::create().to_string();
}

/// @returns a new rack.
rack::Rack create_rack(const Synnax &client) {
    auto r = rack::Rack{.name = unique("certificate rack")};
    EXPECT_NIL(client.racks.create(r));
    return r;
}

/// @returns a new file certificate that the rack with key reads.
FileCertificate create_file(const Synnax &client, const rack::Key rack) {
    Certificate c = FileCertificate{};
    auto &file = std::get<FileCertificate>(c);
    file.name = unique("file");
    file.certificate_path = "/etc/pki/server.pem";
    file.private_key_file = "server-key.pem";
    file.rack = rack;
    EXPECT_NIL(client.certificates.set(c));
    return std::get<FileCertificate>(c);
}
}

/// @brief it should save a stored certificate with its private key, and read it back.
TEST(CertificateTest, SetStored) {
    const auto client = new_test_client();
    const auto name = unique("stored");
    Certificate c = StoredCertificate{};
    std::get<StoredCertificate>(c).name = name;
    std::get<StoredCertificate>(c).pem = x::x509::testutil::CERTIFICATE;
    ASSERT_NIL(client.certificates.set(c, x::x509::testutil::KEY));
    const auto &saved = std::get<StoredCertificate>(c);
    ASSERT_FALSE(saved.key.is_nil());
    ASSERT_TRUE(saved.private_key.has_value());
    ASSERT_EQ(saved.details.subject, "CN=Test Client,O=Synnax");
    const auto got = std::get<StoredCertificate>(
        ASSERT_NIL_P(client.certificates.retrieve(saved.key))
    );
    ASSERT_EQ(got.name, name);
    ASSERT_EQ(got.pem, x::x509::testutil::CERTIFICATE);
    ASSERT_EQ(got.private_key, saved.private_key);
    const auto key = ASSERT_NIL_P(client.secrets.retrieve(saved.private_key.value()));
    ASSERT_EQ(std::get<secret::StoredSecret>(key).name, name + " private key");
}

/// @brief it should save a file certificate, and read it back.
TEST(CertificateTest, SetFile) {
    const auto client = new_test_client();
    const auto r = create_rack(client);
    const auto key = create_file(client, r.key).key;
    const auto got = std::get<FileCertificate>(
        ASSERT_NIL_P(client.certificates.retrieve(key))
    );
    ASSERT_EQ(got.certificate_path, "/etc/pki/server.pem");
    ASSERT_EQ(got.private_key_file, "server-key.pem");
    ASSERT_EQ(got.rack, r.key);
    ASSERT_FALSE(got.details.has_value());
}

/// @brief it should save the details that the rack of a file certificate read.
TEST(CertificateTest, Refresh) {
    const auto client = new_test_client();
    const auto r = create_rack(client);
    const auto key = create_file(client, r.key).key;
    const Synnax as_rack(rack_client_config(client, r.key));
    const Details details{
        .subject = "CN=Test Client,O=Synnax",
        .issuer = "CN=Test Client,O=Synnax",
        .sha1 = "aa",
        .sha256 = "bb",
    };
    ASSERT_NIL(as_rack.certificates.refresh(key, details));
    const auto got = std::get<FileCertificate>(
        ASSERT_NIL_P(client.certificates.retrieve(key))
    );
    ASSERT_TRUE(got.details.has_value());
    ASSERT_EQ(got.details.value().subject, "CN=Test Client,O=Synnax");
    ASSERT_EQ(got.details.value().sha256, "bb");
}

/// @brief it should refuse a refresh from a client that is not the certificate's rack.
TEST(CertificateTest, RefreshRefusesOtherCaller) {
    const auto client = new_test_client();
    const auto r = create_rack(client);
    const auto file = create_file(client, r.key);
    const auto other = create_rack(client);
    const Synnax as_other(rack_client_config(client, other.key));
    const auto other_err = as_other.certificates.refresh(file.key, Details{});
    ASSERT_OCCURRED_AS(other_err, auth::ERR_ACCESS_DENIED);
    ASSERT_EQ(
        other_err.data,
        "certificate \"" + file.name + "\" is read by rack:" + std::to_string(r.key) +
            ", not rack:" + std::to_string(other.key) + ": access denied: auth error"
    );
    const auto user_err = client.certificates.refresh(file.key, Details{});
    ASSERT_OCCURRED_AS(user_err, auth::ERR_ACCESS_DENIED);
    ASSERT_EQ(
        user_err.data,
        "only a rack can refresh a certificate: access denied: auth error"
    );
}

/// @brief it should return NOT_FOUND for a key that no certificate has.
TEST(CertificateTest, RetrieveMissing) {
    const auto client = new_test_client();
    ASSERT_OCCURRED_AS_P(
        client.certificates.retrieve(x::uuid::create()),
        x::errors::NOT_FOUND
    );
}
}
