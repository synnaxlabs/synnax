// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <string>
#include <variant>

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"
#include "x/cpp/x509/testutil/testutil.h"

namespace driver::secret::testutil {
/// @returns a client that signs in as rack r, as the Driver of r does.
[[nodiscard]] inline std::shared_ptr<synnax::Synnax>
rack_client(synnax::Synnax &client, const synnax::rack::Rack &r) {
    const auto key = ASSERT_NIL_P(
        client.api_keys.create("rack", {"rack", std::to_string(r.key)})
    );
    auto cfg = test_client_config();
    cfg.api_key = key.token;
    return std::make_shared<synnax::Synnax>(cfg);
}

/// @returns a stored certificate of x::x509::testutil::CERTIFICATE, whose private key
/// secret holds x::x509::testutil::KEY.
[[nodiscard]] inline synnax::certificate::StoredCertificate
create_stored_certificate(synnax::Synnax &client) {
    synnax::certificate::Certificate c = synnax::certificate::StoredCertificate{};
    auto &stored = std::get<synnax::certificate::StoredCertificate>(c);
    stored.name = "certificate " + x::uuid::create().to_string();
    stored.pem = x::x509::testutil::CERTIFICATE;
    EXPECT_NIL(client.certificates.set(c, x::x509::testutil::KEY));
    return std::get<synnax::certificate::StoredCertificate>(c);
}

/// @returns the key of a stored secret that holds x::x509::testutil::KEY. The C++
/// client writes secrets only as the private keys of stored certificates.
[[nodiscard]] inline synnax::secret::Key create_secret(synnax::Synnax &client) {
    return create_stored_certificate(client).private_key.value();
}
}
