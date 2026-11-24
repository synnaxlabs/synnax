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

#include "client/cpp/auth/apikey/apikey.h"
#include "freighter/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "core/pkg/transport/grpc/auth/auth.pb.h"

namespace synnax::auth::apikey {
namespace {
Client client_with(
    std::unique_ptr<CreateClient> create,
    std::unique_ptr<RetrieveClient> retrieve = nullptr
) {
    return Client(std::move(create), std::move(retrieve), nullptr);
}
}

/// @brief it should fail to create a key when the Core returns a key that is not a
/// UUID.
TEST(TestAPIKeyClient, CreateRejectsNonUUIDKey) {
    grpc::auth::APIKeyCreateResponse res;
    res.set_token("syk_token");
    res.mutable_key()->set_key("not-a-uuid");
    const auto client = client_with(
        std::make_unique<freighter::testutil::UnaryClient<
            grpc::auth::APIKeyCreateRequest,
            grpc::auth::APIKeyCreateResponse>>(res, x::errors::NIL)
    );
    ASSERT_OCCURRED_AS_P(client.create("bad"), x::uuid::INVALID);
}

/// @brief it should fail to list keys when one has a key that is not a UUID.
TEST(TestAPIKeyClient, RetrieveRejectsNonUUIDKey) {
    grpc::auth::APIKeyRetrieveResponse res;
    res.add_keys()->set_key(x::uuid::create().to_string());
    res.add_keys()->set_key("not-a-uuid");
    const auto client = client_with(
        nullptr,
        std::make_unique<freighter::testutil::UnaryClient<
            grpc::auth::APIKeyRetrieveRequest,
            grpc::auth::APIKeyRetrieveResponse>>(res, x::errors::NIL)
    );
    ASSERT_OCCURRED_AS_P(client.retrieve(), x::uuid::INVALID);
}
}
