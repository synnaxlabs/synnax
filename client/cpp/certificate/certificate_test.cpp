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
    auto r = rack::Rack{.name = unique("certificate rack")};
    ASSERT_NIL(client.racks.create(r));
    Certificate c = FileCertificate{};
    auto &file = std::get<FileCertificate>(c);
    file.name = unique("file");
    file.certificate_path = "/etc/pki/server.pem";
    file.private_key_file = "server-key.pem";
    file.rack = r.key;
    ASSERT_NIL(client.certificates.set(c));
    const auto key = std::get<FileCertificate>(c).key;
    const auto got = std::get<FileCertificate>(
        ASSERT_NIL_P(client.certificates.retrieve(key))
    );
    ASSERT_EQ(got.certificate_path, "/etc/pki/server.pem");
    ASSERT_EQ(got.private_key_file, "server-key.pem");
    ASSERT_EQ(got.rack, r.key);
    ASSERT_FALSE(got.details.has_value());
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
