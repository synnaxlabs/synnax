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

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"
#include "x/cpp/x509/testutil/testutil.h"

namespace synnax::secret {
namespace {
std::string unique(const std::string &name) {
    return name + " " + x::uuid::create().to_string();
}

/// @returns the private key of a new stored certificate named name. The C++ client
/// writes secrets only as the private keys of stored certificates.
Key create_secret(const Synnax &client, const std::string &name) {
    certificate::Certificate c = certificate::StoredCertificate{};
    auto &stored = std::get<certificate::StoredCertificate>(c);
    stored.name = name;
    stored.pem = x::x509::testutil::CERTIFICATE;
    EXPECT_NIL(client.certificates.set(c, x::x509::testutil::KEY));
    return std::get<certificate::StoredCertificate>(c).private_key.value();
}

/// @returns a new rack with a task that uses the secret with key.
rack::Rack create_rack_using(const Synnax &client, const Key &key) {
    auto r = rack::Rack{.name = unique("secret rack")};
    EXPECT_NIL(client.racks.create(r));
    auto t = task::Task{
        .name = unique("alerts"),
        .type = "pagerduty_alert",
        .config = {{"auto_start", false}, {"routing_key", key.to_string()}},
    };
    EXPECT_NIL(r.tasks.create(t));
    return r;
}
}

/// @brief it should retrieve secrets by key, and skip a key that no secret has.
TEST(SecretTest, Retrieve) {
    const auto client = new_test_client();
    const auto name = unique("stored");
    const auto key = create_secret(client, name);
    const auto got = std::get<StoredSecret>(ASSERT_NIL_P(client.secrets.retrieve(key)));
    ASSERT_EQ(got.key, key);
    ASSERT_EQ(got.name, name + " private key");
    const auto many = ASSERT_NIL_P(client.secrets.retrieve({key, x::uuid::create()}));
    ASSERT_EQ(many.size(), 1);
    ASSERT_EQ(std::get<StoredSecret>(many[0]).key, key);
}

/// @brief it should return NOT_FOUND for a key that no secret has.
TEST(SecretTest, RetrieveMissing) {
    const auto client = new_test_client();
    ASSERT_OCCURRED_AS_P(
        client.secrets.retrieve(x::uuid::create()),
        x::errors::NOT_FOUND
    );
}

/// @brief it should resolve a secret that a task on the calling rack uses, and refuse
/// it to another rack.
TEST(SecretTest, Resolve) {
    const auto client = new_test_client();
    const auto key = create_secret(client, unique("resolved"));
    const auto r = create_rack_using(client, key);
    const Synnax as_rack(rack_client_config(client, r.key));
    const auto values = ASSERT_NIL_P(as_rack.secrets.resolve({key}));
    ASSERT_EQ(values.size(), 1);
    ASSERT_EQ(values.at(key), x::x509::testutil::KEY);
    auto other = rack::Rack{.name = unique("other rack")};
    ASSERT_NIL(client.racks.create(other));
    const Synnax as_other(rack_client_config(client, other.key));
    const auto [denied, err] = as_other.secrets.resolve({key});
    ASSERT_OCCURRED_AS(err, auth::ERR_ACCESS_DENIED);
    ASSERT_EQ(
        err.data,
        "no device or task on rack:" + std::to_string(other.key) + " uses secret " +
            key.to_string() + ": access denied: auth error"
    );
}
}
