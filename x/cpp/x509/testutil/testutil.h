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

/// @brief the private key of CERTIFICATE.
const std::string KEY = R"(-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgXsLANwJSgnKYh56a
45Wp6vcgMMl4Pkvp3saAT1dd8sGhRANCAARMPAReAwqgqN3zht78hK+Ygm9v2XSb
eI4rgWFp9W8CJfYnO7JRCvKOSVSpX1r+cicMl1D1VwxgRbldmfr5YLy/
-----END PRIVATE KEY-----
)";

/// @brief a certificate with the same names and validity as CERTIFICATE, and another
/// key.
const std::string OTHER_CERTIFICATE = R"(-----BEGIN CERTIFICATE-----
MIIBhzCCAS2gAwIBAgIUTGjJZ0xaaYJBQFgv6TBHTiS+DFcwCgYIKoZIzj0EAwIw
JzEPMA0GA1UECgwGU3lubmF4MRQwEgYDVQQDDAtUZXN0IENsaWVudDAgFw0yNjAx
MDEwMDAwMDBaGA8yMTI1MTIzMTIzNTk1OVowJzEPMA0GA1UECgwGU3lubmF4MRQw
EgYDVQQDDAtUZXN0IENsaWVudDBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABBWm
0NyFDkPyCL9rC6DNDclSKXX6DXSOTF1YU1sj4S5yi9uU0qR2fH7DM0Z7WNo7U4id
bS4P55szWw7ZLosJziCjNTAzMDEGA1UdEQQqMCiGG3VybjpzeW5uYXg6dGVzdDpj
ZXJ0aWZpY2F0ZYIJdGVzdC1ob3N0MAoGCCqGSM49BAMCA0gAMEUCIHDMPuPZQ6ZC
tmu7DORs3l8SqhMJup0hji1cFem/bYoqAiEApzKRjaxzHxrgPQgIWT2NnHvajpoE
BWGYvh2eKE/z/9s=
-----END CERTIFICATE-----
)";
}
