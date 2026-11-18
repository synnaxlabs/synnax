// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <variant>
#include <vector>

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"
#include "x/cpp/x509/testutil/testutil.h"

#include "driver/http/processor/processor.h"
#include "driver/http/types/types.h"

namespace driver::secret::testutil {
/// @returns a new stored certificate of pem. Its private key secret holds private_key,
/// or it has none when private_key is empty.
[[nodiscard]] inline synnax::certificate::StoredCertificate create_stored_certificate(
    synnax::Synnax &client,
    const std::string &pem = x::x509::testutil::CERTIFICATE,
    const std::string &private_key = x::x509::testutil::KEY
) {
    synnax::certificate::Certificate c = synnax::certificate::StoredCertificate{};
    auto &stored = std::get<synnax::certificate::StoredCertificate>(c);
    stored.name = "certificate " + x::uuid::create().to_string();
    stored.pem = pem;
    EXPECT_NIL(client.certificates.set(c, private_key));
    return std::get<synnax::certificate::StoredCertificate>(c);
}

/// @returns the key of a new stored certificate of pem, with no private key.
[[nodiscard]] inline synnax::certificate::Key
create_certificate(synnax::Synnax &client, const std::string &pem) {
    return create_stored_certificate(client, pem, "").key;
}

/// @returns the key of a stored secret that holds x::x509::testutil::KEY. The C++
/// client writes secrets only as the private keys of stored certificates.
[[nodiscard]] inline synnax::secret::Key create_secret(synnax::Synnax &client) {
    return create_stored_certificate(client).private_key.value();
}

/// @returns the key of a grant that lets rack run cmd once with secrets. It logs in to
/// the Core that test_client_config names and posts to its HTTP route, as the Console
/// does, since the C++ client has no grant create.
[[nodiscard]] inline x::uuid::UUID create_grant(
    const synnax::rack::Key rack,
    const synnax::task::Command &cmd,
    const std::vector<synnax::secret::Key> &secrets
) {
    const auto cfg = test_client_config();
    const auto base = "http://" + cfg.host + ":" + std::to_string(cfg.port);
    driver::http::Processor processor;
    const auto post = [&](const std::string &path,
                          const x::json::json &body,
                          const std::string &token) {
        std::vector<std::string> headers{"Content-Type: application/json"};
        if (!token.empty()) headers.push_back("Authorization: Bearer " + token);
        const auto res = ASSERT_NIL_P(processor.execute(
            driver::http::Request{
                .url = base + "/api/v1" + path,
                .base_url = base,
                .method = driver::http::Method::POST,
                .timeout = 5 * x::telem::SECOND,
                .verify_ssl = false,
                .headers = headers,
                .body = body.dump(),
            }
        ));
        EXPECT_EQ(res.status_code, 200) << res.body;
        return x::json::json::parse(res.body);
    };
    const auto session = post(
        "/auth/session/create",
        {{"proof",
          {{"method", "password"},
           {"username", cfg.username},
           {"password", cfg.password}}}},
        ""
    );
    x::json::json keys = x::json::json::array();
    for (const auto &k: secrets)
        keys.push_back(k.to_string());
    const auto res = post(
        "/secret/grant/create",
        {{"subject", {{"type", "rack"}, {"key", std::to_string(rack)}}},
         {"secrets", keys},
         {"command", cmd.to_json()}},
        session.at("token").get<std::string>()
    );
    return ASSERT_NIL_P(x::uuid::UUID::parse(res.at("grant").get<std::string>()));
}
}
