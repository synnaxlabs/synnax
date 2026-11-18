// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <array>
#include <string>
#include <vector>

#include "absl/log/log.h"
#include "open62541/plugin/create_certificate.h"
#include "open62541/types.h"
#include "open62541/util.h"

#include "x/cpp/fs/fs.h"

#include "driver/opcua/certificate/certificate.h"
#include "driver/opcua/errors/errors.h"

namespace driver::opcua::certificate {
x::json::json Info::to_json() const {
    return {
        {"pem", this->pem},
        {"thumbprint", this->thumbprint},
        {"subject", this->subject},
        {"not_after", this->not_after.iso8601()},
    };
}

Info info(const x::x509::Certificate &cert) {
    return Info{
        .pem = cert.pem(),
        .thumbprint = cert.thumbprint(),
        .subject = cert.subject(),
        .not_after = cert.not_after(),
    };
}

std::pair<Identity, x::errors::Error> Identity::load(
    const std::filesystem::path &certificate,
    const std::filesystem::path &key
) {
    auto [cert_data, cert_err] = x::fs::read_file(certificate.string());
    if (cert_err)
        return {
            Identity{},
            x::errors::Error(
                errors::CLIENT_CERTIFICATE_INVALID,
                "Failed to read " + certificate.string() + ". The path must exist " +
                    "on the computer that runs the Driver."
            )
        };
    auto [cert, parse_err] = x::x509::Certificate::parse(cert_data);
    if (parse_err)
        return {
            Identity{},
            x::errors::Error(
                errors::CLIENT_CERTIFICATE_INVALID,
                certificate.string() + ": " + parse_err.data
            )
        };
    auto [key_data, key_err] = x::fs::read_file(key.string());
    if (key_err)
        return {
            Identity{},
            x::errors::Error(
                errors::CLIENT_CERTIFICATE_INVALID,
                "Failed to read " + key.string() + ". The path must exist on the " +
                    "computer that runs the Driver."
            )
        };
    if (const auto match_err = cert.check_key(key_data))
        return {
            Identity{},
            x::errors::Error(
                errors::CLIENT_CERTIFICATE_INVALID,
                key.string() + ": " + match_err.data
            )
        };
    return {
        Identity{.certificate = std::move(cert), .private_key = key_data},
        x::errors::NIL
    };
}

std::pair<Identity, x::errors::Error> create(
    const std::string &application_uri,
    const std::string &common_name,
    const std::string &host,
    const x::telem::TimeSpan validity
) {
    const std::string cn = "CN=" + common_name;
    const std::string uri = "URI:" + application_uri;
    const std::string dns = "DNS:" + host;
    const std::array subject{UA_STRING(const_cast<char *>(cn.c_str()))};
    std::vector san{UA_STRING(const_cast<char *>(uri.c_str()))};
    if (!host.empty()) san.push_back(UA_STRING(const_cast<char *>(dns.c_str())));

    UA_UInt16 key_size = 2048;
    auto days = static_cast<UA_UInt16>(validity.days());
    UA_KeyValuePair params_buf[2];
    params_buf[0].key = UA_QUALIFIEDNAME(0, const_cast<char *>("key-size-bits"));
    UA_Variant_setScalar(&params_buf[0].value, &key_size, &UA_TYPES[UA_TYPES_UINT16]);
    params_buf[1].key = UA_QUALIFIEDNAME(0, const_cast<char *>("expires-in-days"));
    UA_Variant_setScalar(&params_buf[1].value, &days, &UA_TYPES[UA_TYPES_UINT16]);
    UA_KeyValueMap params{2, params_buf};

    UA_Logger logger{};
    logger.log =
        [](void *, UA_LogLevel, UA_LogCategory, const char *msg, va_list args) {
            std::array<char, 1024> buf{};
            UA_String out{
                .length = buf.size(),
                .data = reinterpret_cast<UA_Byte *>(buf.data()),
            };
            UA_String_vformat(&out, msg, args);
            VLOG(1) << "[opc.certificate] " << buf.data();
        };
    UA_ByteString key = UA_BYTESTRING_NULL;
    UA_ByteString cert_data = UA_BYTESTRING_NULL;
    const auto status = UA_CreateCertificate(
        &logger,
        subject.data(),
        subject.size(),
        san.data(),
        san.size(),
        UA_CERTIFICATEFORMAT_PEM,
        &params,
        &key,
        &cert_data
    );
    if (status != UA_STATUSCODE_GOOD)
        return {
            Identity{},
            x::errors::Error(
                errors::DRIVER_CERTIFICATE_FAILED,
                std::string("Failed to create certificate: ") +
                    UA_StatusCode_name(status)
            )
        };
    const std::string key_pem(reinterpret_cast<char *>(key.data), key.length);
    const std::string cert_pem(
        reinterpret_cast<char *>(cert_data.data),
        cert_data.length
    );
    UA_ByteString_clear(&key);
    UA_ByteString_clear(&cert_data);
    auto [cert, err] = x::x509::Certificate::parse(cert_pem);
    if (err)
        return {
            Identity{},
            x::errors::Error(errors::DRIVER_CERTIFICATE_FAILED, err.data)
        };
    return {
        Identity{.certificate = std::move(cert), .private_key = key_pem},
        x::errors::NIL
    };
}
}
