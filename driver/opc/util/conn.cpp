// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// external
#include "glog/logging.h"
#include "mbedtls/error.h"
#include "mbedtls/x509_crt.h"
#include "open62541/client_config_default.h"
#include "open62541/client_highlevel.h"
#include "open62541/common.h"

/// module
#include "x/cpp/xerrors/errors.h"

/// internal
#include "driver/opc/util/util.h"
#include "driver/task/task.h"

namespace util {
using ClientDeleter = void (*)(UA_Client *);

ClientDeleter client_deleter() {
    return [](UA_Client *client) {
        if (client == nullptr) return;
        UA_Client_disconnect(client);
        UA_Client_delete(client);
    };
}

/// @brief intercepts OPC UA log messages and forwards them to glog. Also inserts a
/// prefix for each message that is extracted from the log context. This function
/// will fail silently if the log context is not a string.
void custom_logger(
    void *_,
    const UA_LogLevel level,
    UA_LogCategory __,
    const char *msg,
    va_list args
) {
    const std::string prefix = "[opc] ";
    char buffer[1024];
    vsnprintf(buffer, sizeof(buffer), msg, args);
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

UA_ByteString load_file(const char *const path) {
    UA_ByteString fileContents = UA_STRING_NULL;
    FILE *fp = fopen(path, "rb");
    if (!fp) {
        errno = 0;
        return fileContents;
    }
    fseek(fp, 0, SEEK_END);
    fileContents.length = static_cast<size_t>(ftell(fp));
    fileContents.data = static_cast<UA_Byte *>(
        UA_malloc(fileContents.length * sizeof(UA_Byte))
    );
    if (fileContents.data) {
        fseek(fp, 0, SEEK_SET);
        const size_t
            read = fread(fileContents.data, sizeof(UA_Byte), fileContents.length, fp);
        if (read != fileContents.length) UA_ByteString_clear(&fileContents);
    } else {
        fileContents.length = 0;
    }

    return fileContents;
}

UA_ByteString ua_byte_string(const std::string &certString) {
    UA_ByteString byteString;
    byteString.length = certString.size();
    byteString.data = static_cast<UA_Byte *>(
        UA_malloc(byteString.length * sizeof(UA_Byte))
    );
    if (byteString.data) memcpy(byteString.data, certString.data(), byteString.length);
    return byteString;
}

#ifndef MBEDTLS_X509_SAN_UNIFORM_RESOURCE_IDENTIFIER
// Standard value for URI in X.509 Subject Alternative Name. We need to use this
// instead of the macro MBEDTLS_X509_SAN_UNIFORM_RESOURCE_IDENTIFIER in mbedtls,
// as it's not available in ubuntu 20.04. The actual value is 6, as
// defined in RFC 5280.
#define MBEDTLS_X509_SAN_UNIFORM_RESOURCE_IDENTIFIER 6
#endif


std::string app_uri_from_cert(const std::string &certPath) {
    mbedtls_x509_crt crt;
    mbedtls_x509_crt_init(&crt);

    // Load the certificate
    UA_ByteString certData = load_file(certPath.c_str());
    if (certData.length == 0) {
        LOG(ERROR) << "Failed to load certificate from " << certPath;
        return "";
    }

    const int ret = mbedtls_x509_crt_parse(&crt, certData.data, certData.length);
    if (ret != 0) {
        char errBuf[100];
        mbedtls_strerror(ret, errBuf, sizeof(errBuf));
        LOG(ERROR) << "Failed to parse certificate: " << errBuf;
        UA_ByteString_clear(&certData);
        mbedtls_x509_crt_free(&crt);
        return "";
    }

    // Extract the URI from the SAN field
    std::string applicationUri;
    const mbedtls_asn1_sequence *cur = &crt.subject_alt_names;
    while (cur != nullptr) {
        if (cur->buf.tag == (MBEDTLS_ASN1_CONTEXT_SPECIFIC |
                             MBEDTLS_X509_SAN_UNIFORM_RESOURCE_IDENTIFIER)) {
            applicationUri.assign(reinterpret_cast<char *>(cur->buf.p), cur->buf.len);
            break;
        }
        cur = cur->next;
    }

    if (applicationUri.empty()) {
        LOG(ERROR) << "No URI found in the Subject Alternative Name field of the "
                      "certificate.";
    }

    // Clean up
    UA_ByteString_clear(&certData);
    mbedtls_x509_crt_free(&crt);
    return applicationUri;
}

UA_StatusCode
priv_key_pass_callback(UA_ClientConfig *_, [[maybe_unused]] UA_ByteString *__) {
    return UA_STATUSCODE_BADSECURITYCHECKSFAILED;
}

const std::string SECURITY_URI_BASE = "http://opcfoundation.org/UA/SecurityPolicy#";

xerrors::Error configure_encryption(
    const ConnectionConfig &cfg,
    const std::shared_ptr<UA_Client> &client
) {
    const auto client_config = UA_Client_getConfig(client.get());
    if (cfg.security_mode == "Sign")
        client_config->securityMode = UA_MESSAGESECURITYMODE_SIGN;
    else if (cfg.security_mode == "SignAndEncrypt")
        client_config->securityMode = UA_MESSAGESECURITYMODE_SIGNANDENCRYPT;
    else
        client_config->securityMode = UA_MESSAGESECURITYMODE_NONE;
    if (cfg.security_policy == "None") return xerrors::NIL;

    client_config->privateKeyPasswordCallback = priv_key_pass_callback;

    const std::string uri = SECURITY_URI_BASE + cfg.security_policy;
    client_config->securityPolicyUri = UA_STRING_ALLOC(uri.c_str());
    client_config->authSecurityPolicyUri = UA_STRING_ALLOC(uri.c_str());
    UA_String_clear(&client_config->clientDescription.applicationUri);

    std::string app_uri = app_uri_from_cert(cfg.client_cert);
    if (app_uri.empty()) app_uri = "urn:synnax.opcua.client";
    client_config->clientDescription.applicationUri = UA_STRING_ALLOC(app_uri.c_str());

    const UA_ByteString certificate = load_file(cfg.client_cert.c_str());
    const UA_ByteString priv_key = load_file(cfg.client_private_key.c_str());

    constexpr size_t trust_list_size = 0;
    UA_STACKARRAY(UA_ByteString, trustList, trust_list_size + 1);
    if (!cfg.server_cert.empty()) trustList[0] = load_file(cfg.server_cert.c_str());

    const UA_StatusCode e_err = UA_ClientConfig_setDefaultEncryption(
        client_config,
        certificate,
        priv_key,
        trustList,
        trust_list_size,
        nullptr,
        0
    );

    if (e_err != UA_STATUSCODE_GOOD) {
        LOG(ERROR) << "[opc.scanner] Failed to configure encryption: "
                   << UA_StatusCode_name(e_err);
        const auto status_name = UA_StatusCode_name(e_err);
        return xerrors::Error(
            freighter::TYPE_UNREACHABLE,
            "Failed to configure encryption: " + std::string(status_name)
        );
    }
    return xerrors::NIL;
}

void fetch_endpoint_diagnostic_info(
    const std::shared_ptr<UA_Client> &client,
    const std::string &endpoint
) {
    size_t endpoint_count = 0;
    UA_EndpointDescription *endpoints = nullptr;
    const UA_StatusCode retval = UA_Client_getEndpoints(
        client.get(),
        endpoint.c_str(),
        &endpoint_count,
        &endpoints
    );
    if (retval != UA_STATUSCODE_GOOD) {
        LOG(ERROR) << "[opc.scanner] Failed to get endpoints: "
                   << std::string(UA_StatusCode_name(retval));
        return;
    }
    // get the client config
    for (size_t i = 0; i < endpoint_count; i++) {
        const auto ep = endpoints[i];
        LOG(INFO) << "[opc.scanner] Endpoint " << i << std::endl;
        // if the security policy uri is not null, then the endpoint is secure
        // get the client config
        // get config.userIdentityToken.content.decoded.type
        if (ep.securityPolicyUri.data)
            LOG(INFO) << "[opc.scanner] \t security policy uri: "
                      << ep.securityPolicyUri.data;
        const auto security_mode = ep.securityMode;
        if (security_mode == UA_MESSAGESECURITYMODE_NONE)
            LOG(INFO) << "[opc.scanner] \t security: unencrypted";
        else if (security_mode == UA_MESSAGESECURITYMODE_SIGN)
            LOG(INFO) << "[opc.scanner] \t security: signed";
        else if (security_mode == UA_MESSAGESECURITYMODE_SIGNANDENCRYPT)
            LOG(INFO) << "[opc.scanner] \t security: signed and encrypted";

        for (size_t j = 0; j < ep.userIdentityTokensSize; j++) {
            const UA_UserTokenPolicy policy = ep.userIdentityTokens[j];
            if (policy.tokenType == UA_USERTOKENTYPE_ANONYMOUS)
                LOG(INFO) << "[opc.scanner] \t supports anonymous authentication";
            else if (policy.tokenType == UA_USERTOKENTYPE_USERNAME)
                LOG(
                    INFO
                ) << "[opc.scanner] \t supports username/password authentication";
            else if (policy.tokenType == UA_USERTOKENTYPE_ISSUEDTOKEN)
                LOG(INFO) << "[opc.scanner] \t supports issued token authentication";
            else if (policy.tokenType == UA_USERTOKENTYPE_CERTIFICATE)
                LOG(INFO) << "[opc.scanner] \t supports certificate authentication";
            else
                LOG(INFO) << "[opc.scanner] \t supports unknown authentication type";
        }
    }
}


std::pair<std::shared_ptr<UA_Client>, xerrors::Error>
connect(const ConnectionConfig &cfg, std::string log_prefix) {
    auto client = std::shared_ptr<UA_Client>(UA_Client_new(), client_deleter());
    UA_ClientConfig *config = UA_Client_getConfig(client.get());
    config->logging->log = custom_logger;
    config->logging->context = &log_prefix;

    // Use configured timeouts if provided, otherwise use production defaults
    config->secureChannelLifeTime = cfg.secure_channel_lifetime_ms > 0
                                      ? cfg.secure_channel_lifetime_ms
                                      : 7200000; // Default: 2 hours
    config->requestedSessionTimeout = cfg.session_timeout_ms > 0
                                        ? cfg.session_timeout_ms
                                        : 14400000; // Default: 4 hours
    config->timeout = cfg.client_timeout_ms > 0 ? cfg.client_timeout_ms
                                                : 7200000; // Default: 2 hours

    configure_encryption(cfg, client);
    if (!cfg.username.empty() || !cfg.password.empty()) {
        if (const auto err = parse_error(UA_ClientConfig_setAuthenticationUsername(
                config,
                cfg.username.c_str(),
                cfg.password.c_str()
            )))
            return {nullptr, err};
    }

    const auto err = parse_error(UA_Client_connect(client.get(), cfg.endpoint.c_str()));
    return {std::move(client), err};
}

xerrors::Error
reconnect(const std::shared_ptr<UA_Client> &client, const std::string &endpoint) {
    const auto err = parse_error(UA_Client_connect(client.get(), endpoint.c_str()));
    if (!err) return xerrors::NIL;
    return parse_error(UA_Client_connect(client.get(), endpoint.c_str()));
}
}
