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

namespace x::x509::testutil {
/// @returns a self-signed DER certificate of key, with uri in its alternative name,
/// that holds pem as the value of a non-critical extension, as a forged certificate
/// would. The usages fit an OPC UA application.
[[nodiscard]] std::string create_der_holding(
    const std::string &key,
    const std::string &uri,
    const std::string &pem
);

/// @brief a self-signed P-256 certificate for "CN=Test Client,O=Synnax", valid from
/// 2026-01-01 to 2125-12-31, with a URI and a DNS name in its alternative name.
const std::string CERTIFICATE = R"(-----BEGIN CERTIFICATE-----
MIIBhzCCAS2gAwIBAgIUCo7esOasXFQfGXnv+UE0prCzDx0wCgYIKoZIzj0EAwIw
JzEPMA0GA1UECgwGU3lubmF4MRQwEgYDVQQDDAtUZXN0IENsaWVudDAgFw0yNjAx
MDEwMDAwMDBaGA8yMTI1MTIzMTIzNTk1OVowJzEPMA0GA1UECgwGU3lubmF4MRQw
EgYDVQQDDAtUZXN0IENsaWVudDBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABEw8
BF4DCqCo3fOG3vyEr5iCb2/ZdJt4jiuBYWn1bwIl9ic7slEK8o5JVKlfWv5yJwyX
UPVXDGBFuV2Z+vlgvL+jNTAzMDEGA1UdEQQqMCiGG3VybjpzeW5uYXg6dGVzdDpj
ZXJ0aWZpY2F0ZYIJdGVzdC1ob3N0MAoGCCqGSM49BAMCA0gAMEUCIQDZFy41nKIT
t0DFDW3L/CTyBqqQcAnxq1zl2nIphE/9YwIgFYy5vAB7Je/V7TbYLYtdePvcELKN
nNLdp+AYW7gMXR8=
-----END CERTIFICATE-----
)";
}
