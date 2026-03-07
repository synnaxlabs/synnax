// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/base64/base64.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/test/test.h"

#include "driver/http/device/device.h"

namespace driver::http::device {

TEST(AuthConfigTest, V0APIKeyDefaultsToHeader) {
    x::json::json j = {
        {"type", "api_key"},
        {"header", "X-API-Key"},
        {"key", "secret123"}
    };
    x::json::Parser parser(j);
    AuthConfig auth(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(auth.type, "api_key");
    EXPECT_EQ(auth.send_as, "header");
    EXPECT_EQ(auth.header, "X-API-Key");
    EXPECT_EQ(auth.key, "secret123");
    EXPECT_TRUE(auth.parameter.empty());
}

TEST(AuthConfigTest, V1APIKeyHeader) {
    x::json::json j = {
        {"type", "api_key"},
        {"send_as", "header"},
        {"header", "X-API-Key"},
        {"key", "secret123"}
    };
    x::json::Parser parser(j);
    AuthConfig auth(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(auth.send_as, "header");
    EXPECT_EQ(auth.header, "X-API-Key");
    EXPECT_EQ(auth.key, "secret123");
    EXPECT_TRUE(auth.parameter.empty());
}

TEST(AuthConfigTest, V1APIKeyQueryParam) {
    x::json::json j = {
        {"type", "api_key"},
        {"send_as", "query_param"},
        {"parameter", "api_key"},
        {"key", "secret123"}
    };
    x::json::Parser parser(j);
    AuthConfig auth(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(auth.send_as, "query_param");
    EXPECT_EQ(auth.parameter, "api_key");
    EXPECT_EQ(auth.key, "secret123");
    EXPECT_TRUE(auth.header.empty());
}

TEST(AuthConfigTest, V1APIKeyQueryParamMissingParameterErrors) {
    x::json::json j = {
        {"type", "api_key"},
        {"send_as", "query_param"},
        {"key", "secret123"}
    };
    x::json::Parser parser(j);
    AuthConfig auth(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(AuthConfigTest, V1APIKeyHeaderMissingHeaderErrors) {
    x::json::json j = {
        {"type", "api_key"},
        {"send_as", "header"},
        {"key", "secret123"}
    };
    x::json::Parser parser(j);
    AuthConfig auth(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(AuthConfigTest, APIKeyInvalidSendAsErrors) {
    x::json::json j = {
        {"type", "api_key"},
        {"header", "X-Key"},
        {"key", "secret"},
        {"send_as", "body"}
    };
    x::json::Parser parser(j);
    AuthConfig auth(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(AuthConfigTest, ParsesBearerToken) {
    x::json::json j = {{"type", "bearer"}, {"token", "my-jwt"}};
    x::json::Parser parser(j);
    AuthConfig auth(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(auth.type, "bearer");
    EXPECT_EQ(auth.token, "my-jwt");
}

TEST(AuthConfigTest, ParsesBasic) {
    x::json::json j = {{"type", "basic"}, {"username", "user"}, {"password", "pass"}};
    x::json::Parser parser(j);
    AuthConfig auth(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(auth.type, "basic");
    EXPECT_EQ(auth.username, "user");
    EXPECT_EQ(auth.password, "pass");
}

TEST(AuthConfigTest, BearerMissingTokenErrors) {
    x::json::json j = {{"type", "bearer"}};
    x::json::Parser parser(j);
    AuthConfig auth(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(AuthConfigTest, BasicMissingFieldsErrors) {
    x::json::json j = {{"type", "basic"}, {"username", "user"}};
    x::json::Parser parser(j);
    AuthConfig auth(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(AuthConfigTest, V0APIKeyMissingKeyErrors) {
    x::json::json j = {{"type", "api_key"}, {"header", "X-Key"}};
    x::json::Parser parser(j);
    AuthConfig auth(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(AuthConfigTest, V0APIKeyMissingHeaderErrors) {
    x::json::json j = {{"type", "api_key"}, {"key", "secret"}};
    x::json::Parser parser(j);
    AuthConfig auth(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(AuthConfigTest, UnknownTypeError) {
    x::json::json j = {{"type", "oauth2"}};
    x::json::Parser parser(j);
    AuthConfig auth(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(AuthConfigTest, NoneTypeNoErrors) {
    x::json::json j = {{"type", "none"}};
    x::json::Parser parser(j);
    AuthConfig auth(parser);
    EXPECT_TRUE(parser.ok());
}

TEST(AuthConfigTest, DefaultsToNone) {
    x::json::json j = x::json::json::object();
    x::json::Parser parser(j);
    AuthConfig auth(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(auth.type, "none");
}

TEST(ConnectionConfigTest, FromJSON) {
    x::json::json j = {
        {"base_url", "http://192.168.1.100:8080"},
        {"timeout_ms", 5000},
        {"verify_ssl", true},
        {"auth", {{"type", "bearer"}, {"token", "abc123"}}},
        {"headers", {{"X-Custom", "value"}}}
    };
    x::json::Parser parser(j);
    ConnectionConfig config(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(config.base_url, "http://192.168.1.100:8080");
    EXPECT_EQ(config.timeout, 5 * x::telem::SECOND);
    EXPECT_TRUE(config.verify_ssl);
    EXPECT_EQ(config.auth.type, "bearer");
    EXPECT_EQ(config.auth.token, "abc123");
    EXPECT_EQ(config.headers["X-Custom"], "value");
}

TEST(ConnectionConfigTest, DefaultsApplied) {
    x::json::json j = {{"base_url", "http://localhost"}};
    x::json::Parser parser(j);
    ConnectionConfig config(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(config.base_url, "http://localhost");
    EXPECT_EQ(config.timeout, 100 * x::telem::MILLISECOND);
    EXPECT_TRUE(config.verify_ssl);
    EXPECT_EQ(config.auth.type, "none");
    EXPECT_TRUE(config.headers.empty());
}

TEST(ConnectionConfigTest, VerifySSLFalse) {
    x::json::json j = {
        {"base_url", "http://localhost"},
        {"verify_ssl", false},
    };
    x::json::Parser parser(j);
    ConnectionConfig config(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_FALSE(config.verify_ssl);
}

TEST(ConnectionConfigTest, BaseURLMustStartWithHTTP) {
    x::json::json j = {{"base_url", "ftp://example.com"}};
    x::json::Parser parser(j);
    ConnectionConfig config(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(ConnectionConfigTest, BaseURLAcceptsHTTPS) {
    x::json::json j = {{"base_url", "https://example.com"}};
    x::json::Parser parser(j);
    ConnectionConfig config(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(config.base_url, "https://example.com");
}

TEST(ConnectionConfigTest, BaseURLAcceptsHTTP) {
    x::json::json j = {{"base_url", "http://example.com"}};
    x::json::Parser parser(j);
    ConnectionConfig config(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(config.base_url, "http://example.com");
}

TEST(ConnectionConfigTest, BaseURLRejectsBareHostname) {
    x::json::json j = {{"base_url", "example.com"}};
    x::json::Parser parser(j);
    ConnectionConfig config(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(ConnectionConfigTest, BaseURLRejectsBareHTTPScheme) {
    x::json::json j = {{"base_url", "http://"}};
    x::json::Parser parser(j);
    ConnectionConfig config(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(ConnectionConfigTest, BaseURLRejectsBareHTTPSScheme) {
    x::json::json j = {{"base_url", "https://"}};
    x::json::Parser parser(j);
    ConnectionConfig config(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(ConnectionConfigTest, MissingBaseURLErrors) {
    x::json::json j = {{"timeout_ms", 5000}};
    x::json::Parser parser(j);
    ConnectionConfig config(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(ConnectionConfigTest, InvalidAuthErrors) {
    x::json::json j = {
        {"base_url", "http://localhost"},
        {"auth", {{"type", "bearer"}}}
    };
    x::json::Parser parser(j);
    ConnectionConfig config(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(ConnectionConfigTest, ZeroTimeoutErrors) {
    x::json::json j = {{"base_url", "http://localhost"}, {"timeout_ms", 0}};
    x::json::Parser parser(j);
    ConnectionConfig config(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(ConnectionConfigTest, EmptyJSONErrors) {
    x::json::json j = x::json::json::object();
    x::json::Parser parser(j);
    ConnectionConfig config(parser);
    EXPECT_FALSE(parser.ok());
}

TEST(RetrieveConnectionTest, SecureDefaultBaseURL) {
    auto client = new_test_client();
    auto r = synnax::rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    synnax::device::Device dev{
        .key = "retrieve-conn-test-secure",
        .name = "retrieve-conn-test-secure",
        .rack = r.key,
        .location = "192.168.1.100:8080",
        .make = "http",
        .properties = {{"timeout_ms", 5000}},
    };
    ASSERT_NIL(client.devices.create(dev));

    const auto conn = ASSERT_NIL_P(retrieve_connection(client.devices, dev.key));
    EXPECT_EQ(conn.base_url, "https://192.168.1.100:8080");
    EXPECT_EQ(conn.timeout, 5 * x::telem::SECOND);
}

TEST(RetrieveConnectionTest, InsecureBaseURL) {
    auto client = new_test_client();
    auto r = synnax::rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    synnax::device::Device dev{
        .key = "retrieve-conn-test-insecure",
        .name = "retrieve-conn-test-insecure",
        .rack = r.key,
        .location = "10.0.0.1:9090",
        .make = "http",
        .properties = {{"secure", false}, {"timeout_ms", 2000}},
    };
    ASSERT_NIL(client.devices.create(dev));

    const auto conn = ASSERT_NIL_P(retrieve_connection(client.devices, dev.key));
    EXPECT_EQ(conn.base_url, "http://10.0.0.1:9090");
}

TEST(RetrieveConnectionTest, DeviceNotFound) {
    auto client = new_test_client();
    ASSERT_OCCURRED_AS_P(
        retrieve_connection(client.devices, "non-existent-device-key"),
        x::errors::NOT_FOUND
    );
}

TEST(BuildRequestTest, MergesURLAndPath) {
    auto conn = ConnectionConfig(x::json::Parser({{"base_url", "http://example.com"}}));
    RequestConfig req{.method = Method::GET, .path = "/api/data"};
    auto r = build_request(conn, req);
    EXPECT_EQ(r.url, "http://example.com/api/data");
    EXPECT_EQ(r.method, Method::GET);
    EXPECT_EQ(r.timeout, conn.timeout);
    EXPECT_EQ(r.verify_ssl, true);
}

TEST(BuildRequestTest, PreservesDoubleSlashInPath) {
    auto conn = ConnectionConfig(x::json::Parser({{"base_url", "http://example.com"}}));
    RequestConfig req{.method = Method::GET, .path = "//twoslashes"};
    auto r = build_request(conn, req);
    EXPECT_EQ(r.url, "http://example.com//twoslashes");
}

TEST(BuildRequestTest, MergesConnectionAndRequestHeaders) {
    auto conn = ConnectionConfig(
        x::json::Parser({
            {"base_url", "http://example.com"},
            {"headers", {{"X-Global", "g"}}},
        })
    );
    RequestConfig req{
        .method = Method::GET,
        .path = "/",
        .headers = {{"X-Request", "r"}},
    };
    auto r = build_request(conn, req);
    EXPECT_EQ(r.headers["X-Global"], "g");
    EXPECT_EQ(r.headers["X-Request"], "r");
}

TEST(BuildRequestTest, MergesConnectionAndRequestQueryParams) {
    auto conn = ConnectionConfig(
        x::json::Parser({
            {"base_url", "http://example.com"},
            {"query_params", {{"api_key", "secret"}}},
        })
    );
    RequestConfig req{
        .method = Method::GET,
        .path = "/",
        .query_params = {{"limit", "10"}},
    };
    auto r = build_request(conn, req);
    EXPECT_EQ(r.url, "http://example.com/?api_key=secret&limit=10");
}

TEST(BuildRequestTest, TestQueryParamsEncoding) {
    auto conn = ConnectionConfig(
        x::json::Parser({
            {"base_url", "http://example.com"},
        })
    );
    RequestConfig req{
        .method = Method::GET,
        .path = "/",
        .query_params = {
            {"key with space", "value with ="},
            {"key with &", "value with ?%+#/"}
        },
    };
    auto r = build_request(conn, req);
    EXPECT_TRUE(r.url.find("key+with+space=value+with+%3d") != std::string::npos);
    EXPECT_TRUE(
        r.url.find("key+with+%26=value+with+%3f%25%2b%23%2f") != std::string::npos
    );
}

TEST(BuildRequestTest, ResolvesBearerAuth) {
    auto conn = ConnectionConfig(
        x::json::Parser({
            {"base_url", "http://example.com"},
            {"auth", {{"type", "bearer"}, {"token", "my-jwt"}}},
        })
    );
    RequestConfig req{.method = Method::GET, .path = "/"};
    auto r = build_request(conn, req);
    EXPECT_EQ(r.headers["Authorization"], "Bearer my-jwt");
}

TEST(BuildRequestTest, ResolvesBasicAuth) {
    auto conn = ConnectionConfig(
        x::json::Parser({
            {"base_url", "http://example.com"},
            {"auth", {{"type", "basic"}, {"username", "user"}, {"password", "pass"}}},
        })
    );
    RequestConfig req{.method = Method::GET, .path = "/"};
    auto r = build_request(conn, req);
    EXPECT_EQ(r.headers["Authorization"], "Basic " + x::base64::encode("user:pass"));
}

TEST(BuildRequestTest, ResolvesAPIKeyAsHeader) {
    auto conn = ConnectionConfig(
        x::json::Parser({
            {"base_url", "http://example.com"},
            {"auth", {{"type", "api_key"}, {"header", "X-API-Key"}, {"key", "s123"}}},
        })
    );
    RequestConfig req{.method = Method::GET, .path = "/"};
    auto r = build_request(conn, req);
    EXPECT_EQ(r.headers["X-API-Key"], "s123");
    EXPECT_EQ(r.url.find("X-API-Key"), std::string::npos);
}

TEST(BuildRequestTest, ResolvesV1APIKeyAsQueryParam) {
    auto conn = ConnectionConfig(
        x::json::Parser({
            {"base_url", "http://example.com"},
            {"auth",
             {
                 {"type", "api_key"},
                 {"parameter", "api_key"},
                 {"key", "s123"},
                 {"send_as", "query_param"},
             }},
        })
    );
    RequestConfig req{.method = Method::GET, .path = "/"};
    auto r = build_request(conn, req);
    EXPECT_EQ(r.url, "http://example.com/?api_key=s123");
    EXPECT_TRUE(r.headers.find("api_key") == r.headers.end());
}

TEST(BuildRequestTest, SetsContentTypeForRequestBody) {
    auto conn = ConnectionConfig(x::json::Parser({{"base_url", "http://example.com"}}));
    RequestConfig req{
        .method = Method::POST,
        .path = "/",
        .request_content_type = "application/xml",
    };
    auto r = build_request(conn, req);
    EXPECT_EQ(r.headers["Content-Type"], "application/xml");
}

TEST(BuildRequestTest, NoContentTypeIfRequestContentTypeIsEmpty) {
    auto conn = ConnectionConfig(x::json::Parser({{"base_url", "http://example.com"}}));
    RequestConfig req{.method = Method::GET, .path = "/"};
    auto r = build_request(conn, req);
    EXPECT_TRUE(r.headers.find("Content-Type") == r.headers.end());
}

TEST(BuildRequestTest, NoContentTypeForPOSTIfRequestContentTypeIsEmpty) {
    auto conn = ConnectionConfig(x::json::Parser({{"base_url", "http://example.com"}}));
    RequestConfig req{.method = Method::POST, .path = "/", .request_content_type = ""};
    auto r = build_request(conn, req);
    EXPECT_TRUE(r.headers.find("Content-Type") == r.headers.end());
}

TEST(BuildRequestTest, ContentTypeForPOSTIfRequestContentTypeIsSet) {
    auto conn = ConnectionConfig(x::json::Parser({{"base_url", "http://example.com"}}));
    RequestConfig req{
        .method = Method::GET,
        .path = "/",
        .request_content_type = "application/xml"
    };
    auto r = build_request(conn, req);
    EXPECT_EQ(r.headers["Content-Type"], "application/xml");
}

TEST(BuildRequestTest, ContentTypeForGETIfRequestContentTypeIsSet) {
    auto conn = ConnectionConfig(x::json::Parser({{"base_url", "http://example.com"}}));
    RequestConfig req{
        .method = Method::GET,
        .path = "/",
        .request_content_type = "application/xml"
    };
    auto r = build_request(conn, req);
    EXPECT_EQ(r.headers["Content-Type"], "application/xml");
}
}
