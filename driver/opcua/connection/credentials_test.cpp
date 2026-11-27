// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/x509/testutil/testutil.h"

#include "driver/opcua/connection/connection.h"
#include "driver/opcua/errors/errors.h"
#include "driver/secret/secret.h"
#include "driver/secret/testutil/testutil.h"

namespace driver::opcua::connection {
class ReadCertificatesTest : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client;
    std::unique_ptr<secret::Reader> reader;

    void SetUp() override {
        this->client = std::make_shared<synnax::Synnax>(new_test_client());
        auto r = synnax::rack::Rack{.name = "opcua credentials test"};
        ASSERT_NIL(this->client->racks.create(r));
        this->reader = std::make_unique<secret::Reader>(
            std::make_shared<synnax::Synnax>(rack_client_config(*this->client, r.key)),
            secret::Config{.rack = r.key}
        );
    }
};

/// @brief it should read the login certificate and its private key from a grant.
TEST_F(ReadCertificatesTest, LoginCertificateFromGrant) {
    const auto stored = secret::testutil::create_stored_certificate(*this->client);
    const auto key = stored.private_key.value();
    Config cfg;
    cfg.login_certificate_item = stored.key;
    ASSERT_NIL(read_certificates(
        cfg,
        *this->reader,
        synnax::secret::Grant{
            .secrets = {key},
            .values = {{key, x::x509::testutil::KEY}},
        }
    ));
    ASSERT_TRUE(cfg.login_identity.has_value());
    ASSERT_EQ(cfg.login_identity.value().private_key, x::x509::testutil::KEY);
    ASSERT_FALSE(cfg.client_identity.has_value());
}

/// @brief it should name the login certificate when it has no private key.
TEST_F(ReadCertificatesTest, LoginCertificateWithoutKey) {
    Config cfg;
    cfg.login_certificate_item = secret::testutil::create_certificate(
        *this->client,
        x::x509::testutil::CERTIFICATE
    );
    const auto err = read_certificates(cfg, *this->reader);
    ASSERT_MATCHES(err, errors::LOGIN_CERTIFICATE_INVALID);
    ASSERT_EQ(
        err.data(),
        "The login certificate is invalid: The certificate has no private key."
    );
    ASSERT_FALSE(cfg.login_identity.has_value());
}
}
