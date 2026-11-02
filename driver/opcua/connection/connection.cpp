// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <string_view>

#include "absl/log/log.h"
#include "open62541/client_config_default.h"
#include "open62541/common.h"
#include "open62541/plugin/certificategroup.h"

#include "x/cpp/errors/errors.h"

#include "driver/opcua/connection/connection.h"
#include "driver/opcua/errors/errors.h"

namespace driver::opcua::connection {
namespace {
using ClientDeleter = void (*)(UA_Client *);

ClientDeleter client_deleter() {
    return [](UA_Client *client) {
        if (client == nullptr) return;
        UA_Client_disconnect(client);
        UA_Client_delete(client);
    };
}

/// @brief intercepts OPC UA log messages and forwards them to the Driver log. Also
/// inserts a prefix for each message that is extracted from the log context. This
/// function will fail silently if the log context is not a string.
void custom_logger(
    void *,
    const UA_LogLevel level,
    UA_LogCategory,
    const char *msg,
    va_list args
) {
    const std::string prefix = "[opc] ";
    char buffer[1024] = {};
    // open62541 messages use specifiers that vsnprintf does not know, such as %S for a
    // UA_String.
    UA_String out{
        .length = sizeof(buffer),
        .data = reinterpret_cast<UA_Byte *>(buffer)
    };
    UA_String_vformat(&out, msg, args);
    switch (level) {
        case UA_LOGLEVEL_TRACE:
        case UA_LOGLEVEL_DEBUG:
        case UA_LOGLEVEL_INFO:
        case UA_LOGLEVEL_WARNING:
            VLOG(1) << prefix << buffer;
            break;
        case UA_LOGLEVEL_ERROR:
            LOG(WARNING) << prefix << buffer;
            break;
        case UA_LOGLEVEL_FATAL:
            LOG(FATAL) << prefix << buffer;
            break;
        default:
            LOG(INFO) << prefix << buffer; // Default case falls back to INFO level
    }
}

const std::string SECURITY_URI_BASE = "http://opcfoundation.org/UA/SecurityPolicy#";
/// @brief the ApplicationUri of a client certificate that has no URI. Servers that
/// check the URI reject it.
const std::string FALLBACK_APPLICATION_URI = "urn:synnax.opcua.client";

UA_StatusCode priv_key_pass_callback(UA_ClientConfig *, UA_ByteString *) {
    return UA_STATUSCODE_BADSECURITYCHECKSFAILED;
}

UA_ByteString byte_string(const std::string &data) {
    return {data.size(), reinterpret_cast<UA_Byte *>(const_cast<char *>(data.data()))};
}

/// @brief checks the certificate that the server presents when it opens a secure
/// channel, and keeps it for server_certificate and the connection error.
struct Verifier {
    std::optional<certificate::Certificate> trusted;
    bool first_certificate_trusted = false;
    bool disabled = false;
    x::telem::NowFunc now;
    std::string client_thumbprint;
    std::optional<certificate::Certificate> received;
    x::errors::Error rejection = x::errors::NIL;
};

x::errors::Error untrusted_error(
    const std::optional<certificate::Certificate> &trusted,
    const certificate::Certificate &received
) {
    if (!trusted.has_value())
        return {
            errors::SERVER_CERTIFICATE_UNTRUSTED,
            "The Driver does not trust the server certificate " +
                received.thumbprint() +
                " yet. Compare the thumbprint with the certificate on the server, " +
                "then trust it."
        };
    return {
        errors::SERVER_CERTIFICATE_UNTRUSTED,
        "The server presented the certificate " + received.thumbprint() +
            " instead of the trusted certificate " + trusted->thumbprint() +
            ". If the server certificate changed on purpose, trust the new one in "
            "the device's connection settings."
    };
}

x::errors::Error time_invalid_error(
    const certificate::Certificate &received,
    const x::telem::TimeStamp now
) {
    return {
        errors::SERVER_CERTIFICATE_TIME_INVALID,
        "The server certificate is valid from " + received.not_before().iso8601() +
            " to " + received.not_after().iso8601() +
            ", and the Driver's clock reads " + now.iso8601() +
            ". Renew the server certificate, or correct the clock on the Driver's "
            "computer."
    };
}

UA_StatusCode verify_certificate(UA_CertificateGroup *cg, const UA_ByteString *data) {
    auto *v = static_cast<Verifier *>(cg->context);
    auto [cert, err] = certificate::Certificate::parse(
        std::string_view(reinterpret_cast<const char *>(data->data), data->length)
    );
    if (err) {
        v->rejection = {errors::SERVER_CERTIFICATE_UNTRUSTED, err.data};
        return UA_STATUSCODE_BADCERTIFICATEINVALID;
    }
    v->received = cert;
    if (v->disabled) return UA_STATUSCODE_GOOD;
    if ((v->trusted.has_value() && *v->trusted != cert) ||
        (!v->trusted.has_value() && !v->first_certificate_trusted)) {
        v->rejection = untrusted_error(v->trusted, cert);
        return UA_STATUSCODE_BADCERTIFICATEUNTRUSTED;
    }
    if (const auto now = v->now(); !cert.valid_at(now)) {
        v->rejection = time_invalid_error(cert, now);
        return UA_STATUSCODE_BADCERTIFICATETIMEINVALID;
    }
    // The first certificate stays trusted for the life of the client, so a
    // reconnect cannot reach a different server.
    if (!v->trusted.has_value()) v->trusted = std::move(cert);
    return UA_STATUSCODE_GOOD;
}

void clear_verifier(UA_CertificateGroup *cg) {
    delete static_cast<Verifier *>(cg->context);
    cg->context = nullptr;
}

Verifier *verifier(UA_Client *client) {
    const auto &cg = UA_Client_getConfig(client)->certificateVerification;
    if (cg.verifyCertificate != verify_certificate) return nullptr;
    return static_cast<Verifier *>(cg.context);
}

x::errors::Error
configure_encryption(const Config &cfg, const Security &sec, UA_ClientConfig *config) {
    if (cfg.security_mode == "Sign")
        config->securityMode = UA_MESSAGESECURITYMODE_SIGN;
    else if (cfg.security_mode == "SignAndEncrypt")
        config->securityMode = UA_MESSAGESECURITYMODE_SIGNANDENCRYPT;
    else
        config->securityMode = UA_MESSAGESECURITYMODE_NONE;
    if (!cfg.secured()) return x::errors::NIL;
    if (!sec.identity.has_value())
        return {errors::ENCRYPTION_CONFIG_FAILED, "No client certificate to present."};
    const auto &id = *sec.identity;

    config->privateKeyPasswordCallback = priv_key_pass_callback;
    const std::string uri = SECURITY_URI_BASE + cfg.security_policy;
    config->securityPolicyUri = UA_STRING_ALLOC(uri.c_str());
    const auto &app_uri = id.certificate.application_uri().empty()
                            ? FALLBACK_APPLICATION_URI
                            : id.certificate.application_uri();
    UA_String_clear(&config->clientDescription.applicationUri);
    config->clientDescription.applicationUri = UA_STRING_ALLOC(app_uri.c_str());

    if (const auto status = UA_ClientConfig_setDefaultEncryption(
            config,
            byte_string(id.certificate.der()),
            byte_string(id.private_key),
            nullptr,
            0,
            nullptr,
            0
        );
        status != UA_STATUSCODE_GOOD)
        return {
            errors::ENCRYPTION_CONFIG_FAILED,
            "Failed to configure encryption: " + std::string(UA_StatusCode_name(status))
        };

    auto &cg = config->certificateVerification;
    const auto *logging = cg.logging;
    if (cg.clear != nullptr) cg.clear(&cg);
    // A clear leaves the trust list callbacks of the default group in place.
    cg = UA_CertificateGroup{};
    cg.logging = logging;
    cg.context = new Verifier{
        .trusted = cfg.server_certificate,
        .first_certificate_trusted = sec.first_certificate_trusted,
        .disabled = cfg.certificate_validation_disabled,
        .now = sec.now,
        .client_thumbprint = id.certificate.thumbprint(),
    };
    cg.verifyCertificate = verify_certificate;
    cg.clear = clear_verifier;
    return x::errors::NIL;
}

bool rejected_by_server(const UA_StatusCode status) {
    const std::string_view name = UA_StatusCode_name(status);
    return status == UA_STATUSCODE_BADSECURITYCHECKSFAILED ||
           name.starts_with("BadCertificate");
}

/// @brief connects the client and names the side that refused a secure channel.
x::errors::Error connect_client(UA_Client *client, const std::string &endpoint) {
    auto *v = verifier(client);
    if (v != nullptr) {
        v->received.reset();
        v->rejection = x::errors::NIL;
    }
    const auto status = UA_Client_connect(client, endpoint.c_str());
    if (v == nullptr || status == UA_STATUSCODE_GOOD) return errors::parse(status);
    if (v->rejection) return v->rejection;
    if (rejected_by_server(status))
        return {
            errors::CLIENT_CERTIFICATE_REJECTED,
            "The server rejected the connection's security checks (" +
                std::string(UA_StatusCode_name(status)) +
                "). Most often, the server does not trust the client certificate " +
                v->client_thumbprint +
                ". Approve it in the server's trusted certificates, then connect again."
        };
    return errors::parse(status);
}
}

std::pair<Security, x::errors::Error>
resolve_security(const Config &cfg, certificate::Store &store) {
    Security sec;
    if (!cfg.secured()) return {sec, x::errors::NIL};
    if (cfg.uses_files()) {
        auto [id, err] = certificate::Identity::load(
            cfg.client_cert,
            cfg.client_private_key
        );
        if (err) return {sec, err};
        sec.identity = std::move(id);
        return {sec, x::errors::NIL};
    }
    auto [id, err] = store.current();
    if (err) return {sec, err};
    sec.identity = std::move(id);
    return {sec, x::errors::NIL};
}

std::pair<std::shared_ptr<UA_Client>, x::errors::Error>
connect(const Config &cfg, const Security &sec, std::string log_prefix) {
    auto client = std::shared_ptr<UA_Client>(UA_Client_new(), client_deleter());
    UA_ClientConfig *config = UA_Client_getConfig(client.get());
    config->logging->log = custom_logger;
    config->logging->context = &log_prefix;

    // Use configured timeouts if provided, otherwise use production defaults.
    // Short defaults prevent one unresponsive PLC from tying up worker threads.
    config->secureChannelLifeTime = cfg.secure_channel_lifetime_ms > 0
                                      ? cfg.secure_channel_lifetime_ms
                                      : 600000; // Default: 10 minutes
    config->requestedSessionTimeout = cfg.session_timeout_ms > 0
                                        ? cfg.session_timeout_ms
                                        : 1200000; // Default: 20 minutes
    config->timeout = cfg.client_timeout_ms > 0 ? cfg.client_timeout_ms
                                                : 5000; // Default: 5 seconds

    if (const auto enc_err = configure_encryption(cfg, sec, config))
        return {nullptr, enc_err};
    if (!cfg.username.empty() || !cfg.password.empty()) {
        if (const auto err = errors::parse(UA_ClientConfig_setAuthenticationUsername(
                config,
                cfg.username.c_str(),
                cfg.password.c_str()
            )))
            return {nullptr, err};
    }

    const auto err = connect_client(client.get(), cfg.endpoint);
    return {std::move(client), err};
}

x::errors::Error
reconnect(const std::shared_ptr<UA_Client> &client, const std::string &endpoint) {
    const auto err = connect_client(client.get(), endpoint);
    if (!err) return x::errors::NIL;
    return connect_client(client.get(), endpoint);
}

std::optional<certificate::Certificate> server_certificate(UA_Client *client) {
    const auto *v = verifier(client);
    if (v == nullptr) return std::nullopt;
    return v->received;
}

std::optional<certificate::Certificate> trusted_certificate(UA_Client *client) {
    const auto *v = verifier(client);
    if (v == nullptr) return std::nullopt;
    return v->trusted;
}
}
