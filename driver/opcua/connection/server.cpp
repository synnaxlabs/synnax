// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>
#include <string>
#include <type_traits>
#include <variant>

#include "driver/opcua/connection/server.h"

namespace driver::opcua::connection {
namespace {
namespace srv = ::synnax::opcua::server;

uint32_t to_ms(const x::telem::TimeSpan span) {
    return static_cast<uint32_t>(span.milliseconds());
}

void apply_channel(Config &cfg, const srv::SecuredChannel &channel) {
    cfg.security_policy = channel.policy;
    if (channel.client_certificate.has_value())
        cfg.client_certificate_item = *channel.client_certificate;
    std::visit(
        [&cfg]<typename T>(const T &trust) {
            if constexpr (std::is_same_v<T, srv::PinnedServerTrust>)
                cfg.server_certificate_item = trust.certificate;
            else if constexpr (std::is_same_v<T, srv::ValidationDisabledServerTrust>)
                cfg.certificate_validation_disabled = true;
        },
        channel.server_trust
    );
}
}

std::pair<Config, x::errors::Error> from_server(const srv::Connection &conn) {
    Config cfg;
    cfg.endpoint = conn.endpoint;
    cfg.client_timeout_ms = to_ms(conn.client_timeout);
    cfg.secure_channel_lifetime_ms = to_ms(conn.secure_channel_lifetime);
    cfg.session_timeout_ms = to_ms(conn.session_timeout);
    if (std::holds_alternative<srv::CertificateAuth>(conn.auth))
        return {
            Config(),
            x::errors::Error(
                x::errors::VALIDATION,
                "the Driver does not support certificate login"
            )
        };
    if (const auto *auth = std::get_if<srv::UsernameAuth>(&conn.auth)) {
        cfg.username = auth->username;
        if (auth->password.has_value()) cfg.password_secret = *auth->password;
    }
    if (const auto *sign = std::get_if<srv::SignSecurity>(&conn.security)) {
        cfg.security_mode = "Sign";
        apply_channel(cfg, *sign);
    } else if (
        const auto *enc = std::get_if<srv::SignAndEncryptSecurity>(&conn.security)
    ) {
        cfg.security_mode = "SignAndEncrypt";
        apply_channel(cfg, *enc);
    }
    return {std::move(cfg), x::errors::NIL};
}
}
