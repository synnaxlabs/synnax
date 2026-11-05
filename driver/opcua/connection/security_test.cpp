// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <filesystem>
#include <fstream>
#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "gtest/gtest.h"
#include "open62541/client.h"

#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/opcua/certificate/certificate.h"
#include "driver/opcua/certificate/store.h"
#include "driver/opcua/connection/connection.h"
#include "driver/opcua/errors/errors.h"
#include "driver/opcua/testutil/server.h"
#include "driver/opcua/testutil/testutil.h"

namespace driver::opcua::connection {
namespace {
constexpr std::uint16_t PORT = 4870;
const std::string ENDPOINT = "opc.tcp://localhost:4870";
const std::string LOG_PREFIX = "[test] ";

bool activated(UA_Client *client) {
    UA_SessionState session = UA_SESSIONSTATE_CLOSED;
    UA_SecureChannelState channel = UA_SECURECHANNELSTATE_CLOSED;
    UA_Client_getState(client, &channel, &session, nullptr);
    return session == UA_SESSIONSTATE_ACTIVATED;
}

std::string mismatch_message(
    const certificate::Certificate &received,
    const certificate::Certificate &trusted
) {
    return "The server presented the certificate " + received.thumbprint() +
           " instead of the trusted certificate " + trusted.thumbprint() +
           ". If the server certificate changed on purpose, trust the new one in the "
           "device's connection settings.";
}

void write_file(const std::filesystem::path &path, const std::string &data) {
    std::ofstream out(path, std::ios::binary);
    out << data;
}

class SecurityTest : public ::testing::Test {
protected:
    /// @brief the certificate that the server presents.
    static inline certificate::Identity server_id;
    /// @brief a certificate that no server presents.
    static inline certificate::Identity other_id;
    /// @brief the certificate that the client presents.
    static inline certificate::Identity client_id;

    /// @brief a Store directory with one Driver identity. Each test copies it, since
    /// making a key takes seconds on Windows.
    static inline std::filesystem::path store_dir;

    std::filesystem::path dir;
    std::unique_ptr<testutil::Server> server;

    static void SetUpTestSuite() {
        store_dir = std::filesystem::temp_directory_path() /
                    ("opcua-security-store-" + x::uuid::create().to_string());
        ASSERT_NIL_P(
            certificate::Store(certificate::StoreConfig{.dir = store_dir}).current()
        );
        server_id = ASSERT_NIL_P(testutil::create_server_identity());
        other_id = ASSERT_NIL_P(testutil::create_server_identity());
        client_id = ASSERT_NIL_P(
            certificate::create(
                "urn:synnax:test:client",
                "Synnax Test Client",
                "localhost",
                x::telem::DAY
            )
        );
    }

    static void TearDownTestSuite() { std::filesystem::remove_all(store_dir); }

    void SetUp() override {
        this->dir = std::filesystem::temp_directory_path() /
                    ("opcua-security-" + x::uuid::create().to_string());
        std::filesystem::copy(
            store_dir,
            this->dir,
            std::filesystem::copy_options::recursive
        );
    }

    void TearDown() override {
        if (this->server) this->server->stop();
        std::filesystem::remove_all(this->dir);
    }

    void start_server(
        const certificate::Identity &id,
        std::vector<certificate::Certificate> trusted_clients = {},
        const std::uint16_t max_sessions = 0,
        std::vector<std::pair<std::string, std::string>> users = {}
    ) {
        if (this->server) this->server->stop();
        auto cfg = testutil::ServerConfig::create_default();
        cfg.port = PORT;
        cfg.identity = id;
        cfg.trusted_clients = std::move(trusted_clients);
        cfg.max_sessions = max_sessions;
        cfg.users = std::move(users);
        this->server = std::make_unique<testutil::Server>(cfg);
        this->server->start();
        ASSERT_TRUE(this->server->wait_until_ready());
    }

    [[nodiscard]] std::shared_ptr<certificate::Store> store() const {
        return std::make_shared<certificate::Store>(
            certificate::StoreConfig{.dir = this->dir}
        );
    }

    [[nodiscard]] static Config config(const std::string &policy = "Basic256Sha256") {
        Config cfg;
        cfg.endpoint = ENDPOINT;
        cfg.security_mode = "SignAndEncrypt";
        cfg.security_policy = policy;
        return cfg;
    }

    [[nodiscard]] static Security security() { return Security{.identity = client_id}; }
};
}

/// @brief it should connect to a server that presents the trusted certificate.
TEST_F(SecurityTest, TrustedCertificate) {
    this->start_server(server_id);
    auto cfg = config();
    cfg.server_certificate = server_id.certificate;
    const auto client = ASSERT_NIL_P(connect(cfg, security(), "test"));
    ASSERT_TRUE(activated(client.get()));
    ASSERT_EQ(server_certificate(client.get()), server_id.certificate);
}

/// @brief it should connect with each security policy and mode that the Console offers.
TEST_F(SecurityTest, EachSecurityPolicy) {
    this->start_server(server_id);
    for (const std::string policy:
         {"Basic128Rsa15",
          "Basic256",
          "Basic256Sha256",
          "Aes128_Sha256_RsaOaep",
          "Aes256_Sha256_RsaPss"})
        for (const std::string mode: {"Sign", "SignAndEncrypt"}) {
            SCOPED_TRACE(policy + " " + mode);
            auto cfg = config(policy);
            cfg.security_mode = mode;
            cfg.server_certificate = server_id.certificate;
            const auto client = ASSERT_NIL_P(connect(cfg, security(), "test"));
            ASSERT_TRUE(activated(client.get()));
        }
}

/// @brief it should reject a server that presents another certificate than the
/// trusted one, and keep the certificate that the server presented.
TEST_F(SecurityTest, CertificateMismatch) {
    this->start_server(server_id);
    auto cfg = config();
    cfg.server_certificate = other_id.certificate;
    const auto [client, err] = connect(cfg, security(), "test");
    ASSERT_MATCHES(err, errors::SERVER_CERTIFICATE_UNTRUSTED);
    ASSERT_EQ(err.data, mismatch_message(server_id.certificate, other_id.certificate));
    ASSERT_FALSE(activated(client.get()));
    ASSERT_EQ(server_certificate(client.get()), server_id.certificate);
}

/// @brief it should reject every server certificate when it trusts none and does not
/// trust the first one.
TEST_F(SecurityTest, NoTrustedCertificate) {
    this->start_server(server_id);
    const auto [client, err] = connect(config(), security(), "test");
    ASSERT_MATCHES(err, errors::SERVER_CERTIFICATE_UNTRUSTED);
    ASSERT_EQ(
        err.data,
        "The Driver does not trust the server certificate " +
            server_id.certificate.thumbprint() +
            " yet. Compare the thumbprint with the certificate on the server, then "
            "trust it."
    );
    ASSERT_EQ(server_certificate(client.get()), server_id.certificate);
}

/// @brief it should trust the first server certificate, then only that one when it
/// connects again.
TEST_F(SecurityTest, FirstCertificateLocks) {
    this->start_server(server_id);
    auto sec = security();
    sec.first_certificate_trusted = true;
    const auto client = ASSERT_NIL_P(connect(config(), sec, "test"));
    ASSERT_EQ(server_certificate(client.get()), server_id.certificate);
    UA_Client_disconnect(client.get());
    this->start_server(other_id);
    const auto err = reconnect(client, ENDPOINT);
    ASSERT_MATCHES(err, errors::SERVER_CERTIFICATE_UNTRUSTED);
    ASSERT_EQ(err.data, mismatch_message(other_id.certificate, server_id.certificate));
    ASSERT_FALSE(activated(client.get()));
}

/// @brief it should accept any server certificate when validation is disabled.
TEST_F(SecurityTest, ValidationDisabled) {
    this->start_server(server_id);
    auto cfg = config();
    cfg.server_certificate = other_id.certificate;
    cfg.certificate_validation_disabled = true;
    const auto client = ASSERT_NIL_P(connect(cfg, security(), "test"));
    ASSERT_TRUE(activated(client.get()));
    ASSERT_EQ(server_certificate(client.get()), server_id.certificate);
}

/// @brief it should reject a trusted server certificate that is not valid at the
/// Driver's time.
TEST_F(SecurityTest, ExpiredCertificate) {
    this->start_server(server_id);
    auto cfg = config();
    cfg.server_certificate = server_id.certificate;
    const auto &cert = server_id.certificate;
    const auto now = cert.not_after() + x::telem::DAY;
    auto sec = security();
    sec.now = [now] { return now; };
    const auto err = connect(cfg, sec, "test").second;
    ASSERT_MATCHES(err, errors::SERVER_CERTIFICATE_TIME_INVALID);
    ASSERT_EQ(
        err.data,
        "The server certificate is valid from " + cert.not_before().iso8601() + " to " +
            cert.not_after().iso8601() + ", and the Driver's clock reads " +
            now.iso8601() +
            ". Renew the server certificate, or correct the clock on the Driver's "
            "computer."
    );
}

/// @brief it should name the client certificate when the server rejects it.
TEST_F(SecurityTest, ServerRejectsClient) {
    this->start_server(server_id, {other_id.certificate});
    auto cfg = config();
    cfg.server_certificate = server_id.certificate;
    const auto err = connect(cfg, security(), "test").second;
    ASSERT_MATCHES(err, errors::CLIENT_CERTIFICATE_REJECTED);
    ASSERT_EQ(
        err.data,
        "The server rejected the connection's security checks "
        "(BadSecurityChecksFailed). Most often, the server does not trust the client "
        "certificate " +
            client_id.certificate.thumbprint() +
            ". Approve it in the server's trusted certificates, then connect again."
    );
}

/// @brief it should connect to a server that trusts only the client certificate.
TEST_F(SecurityTest, ServerTrustsClient) {
    this->start_server(server_id, {client_id.certificate});
    auto cfg = config();
    cfg.server_certificate = server_id.certificate;
    const auto client = ASSERT_NIL_P(connect(cfg, security(), "test"));
    ASSERT_TRUE(activated(client.get()));
}

/// @brief it should present the certificate files that the configuration names
/// instead of a Driver certificate.
TEST_F(SecurityTest, CertificateFiles) {
    this->start_server(server_id, {client_id.certificate});
    std::filesystem::create_directories(this->dir);
    write_file(this->dir / "cert.pem", client_id.certificate.pem());
    write_file(this->dir / "key.pem", client_id.private_key);
    auto cfg = config();
    cfg.server_certificate = server_id.certificate;
    cfg.client_cert = (this->dir / "cert.pem").string();
    cfg.client_private_key = (this->dir / "key.pem").string();
    const auto pki = this->dir / "pki";
    certificate::Store store(certificate::StoreConfig{.dir = pki});
    const auto sec = ASSERT_NIL_P(resolve_security(cfg, store));
    ASSERT_EQ(sec.identity.value().certificate, client_id.certificate);
    ASSERT_FALSE(std::filesystem::exists(pki));
    const auto client = ASSERT_NIL_P(connect(cfg, sec, "test"));
    ASSERT_TRUE(activated(client.get()));
}

/// @brief it should present the Driver certificate to a server that trusts only it.
TEST_F(SecurityTest, DriverCertificate) {
    const auto store = this->store();
    const auto current = ASSERT_NIL_P(store->current());
    this->start_server(server_id, {current.certificate});
    auto cfg = config();
    cfg.server_certificate = server_id.certificate;
    const auto sec = ASSERT_NIL_P(resolve_security(cfg, *store));
    ASSERT_EQ(sec.identity.value().certificate, current.certificate);
    const auto client = ASSERT_NIL_P(connect(cfg, sec, "test"));
    ASSERT_TRUE(activated(client.get()));
}

/// @brief it should serve a device with the connection that trusted the first
/// certificate after the device pins that certificate.
TEST_F(SecurityTest, PoolReusesConnectionAfterPin) {
    this->start_server(server_id);
    Pool pool(this->store());
    auto cfg = config();
    UA_Client *first = nullptr;
    {
        const auto conn = ASSERT_NIL_P(pool.acquire(cfg, LOG_PREFIX));
        first = conn.get();
        ASSERT_EQ(server_certificate(first), server_id.certificate);
    }
    cfg.server_certificate = server_id.certificate;
    const auto conn = ASSERT_NIL_P(pool.acquire(cfg, LOG_PREFIX));
    ASSERT_EQ(conn.get(), first);
    ASSERT_EQ(pool.size(), 1);
}

/// @brief it should not give a caller a connection to a server whose certificate the
/// caller does not trust, and keep that connection for callers that trust it.
TEST_F(SecurityTest, PoolSkipsUntrustedConnection) {
    this->start_server(server_id);
    Pool pool(this->store());
    const auto cfg = config();
    UA_Client *first = nullptr;
    {
        const auto conn = ASSERT_NIL_P(pool.acquire(cfg, LOG_PREFIX));
        first = conn.get();
    }
    auto pinned = cfg;
    pinned.server_certificate = other_id.certificate;
    ASSERT_OCCURRED_AS_P(
        pool.acquire(pinned, LOG_PREFIX),
        errors::SERVER_CERTIFICATE_UNTRUSTED
    );
    ASSERT_EQ(pool.size(), 1);
    ASSERT_EQ(pool.available_count(ENDPOINT), 1);
    const auto conn = ASSERT_NIL_P(pool.acquire(cfg, LOG_PREFIX));
    ASSERT_EQ(conn.get(), first);
}

/// @brief it should not give the connection of one login to another login, even when
/// their fields join into the same text.
TEST_F(SecurityTest, PoolSeparatesLogins) {
    this->start_server(server_id, {}, 0, {{"a|b", "c"}});
    Pool pool(this->store());
    auto first = config();
    first.username = "a|b";
    first.password = "c";
    { const auto conn = ASSERT_NIL_P(pool.acquire(first, LOG_PREFIX)); }
    auto second = config();
    second.username = "a";
    second.password = "b|c";
    ASSERT_OCCURRED_AS_P(pool.acquire(second, LOG_PREFIX), errors::SESSION_REJECTED);
    ASSERT_EQ(pool.available_count(ENDPOINT), 1);
}

/// @brief it should stop connecting after repeated certificate rejections, and repeat
/// the rejection while it waits.
TEST_F(SecurityTest, PoolBreakerCountsCertificateRejections) {
    this->start_server(server_id);
    Pool pool(this->store());
    auto cfg = config();
    cfg.server_certificate = other_id.certificate;
    const auto expected = mismatch_message(server_id.certificate, other_id.certificate);
    for (int i = 0; i < 3; i++) {
        const auto err = pool.acquire(cfg, LOG_PREFIX).second;
        ASSERT_MATCHES(err, errors::SERVER_CERTIFICATE_UNTRUSTED);
        ASSERT_EQ(err.data, expected);
    }
    this->server->stop();
    const auto err = pool.acquire(cfg, LOG_PREFIX).second;
    ASSERT_MATCHES(err, errors::SERVER_CERTIFICATE_UNTRUSTED);
    ASSERT_EQ(err.data, expected);
}

/// @brief it should connect at once when the configuration trusts a new certificate
/// after rejections of the old one opened the breaker.
TEST_F(SecurityTest, PoolBreakerResetsOnNewTrustedCertificate) {
    this->start_server(server_id);
    Pool pool(this->store());
    auto cfg = config();
    cfg.server_certificate = other_id.certificate;
    for (int i = 0; i < 3; i++)
        ASSERT_OCCURRED_AS_P(
            pool.acquire(cfg, LOG_PREFIX),
            errors::SERVER_CERTIFICATE_UNTRUSTED
        );
    cfg.server_certificate = server_id.certificate;
    const auto conn = ASSERT_NIL_P(pool.acquire(cfg, LOG_PREFIX));
    ASSERT_TRUE(activated(conn.get()));
    ASSERT_EQ(pool.size(), 1);
}

/// @brief it should close an idle connection with the old Driver certificate before it
/// connects with the new one.
TEST_F(SecurityTest, PoolClosesIdleConnectionAfterCertificateChange) {
    this->start_server(server_id, {}, 1);
    const auto store = this->store();
    Pool pool(store);
    auto cfg = config();
    cfg.server_certificate = server_id.certificate;
    { auto conn = ASSERT_NIL_P(pool.acquire(cfg, LOG_PREFIX)); }
    ASSERT_EQ(pool.available_count(ENDPOINT), 1);
    ASSERT_NIL(store->regenerate().second);
    const auto conn = ASSERT_NIL_P(pool.acquire(cfg, LOG_PREFIX));
    ASSERT_TRUE(activated(conn.get()));
    ASSERT_EQ(pool.size(), 1);
}

/// @brief it should keep a connection whose client certificate has not changed.
TEST_F(SecurityTest, PoolRefreshKeepsCurrentCertificate) {
    this->start_server(server_id);
    Pool pool(this->store());
    auto cfg = config();
    cfg.server_certificate = server_id.certificate;
    auto conn = ASSERT_NIL_P(pool.acquire(cfg, LOG_PREFIX));
    auto *const first = conn.get();
    const auto next = ASSERT_NIL_P(pool.refresh(conn, cfg, LOG_PREFIX));
    ASSERT_FALSE(next);
    ASSERT_EQ(conn.get(), first);
    ASSERT_EQ(pool.size(), 1);
}

/// @brief it should connect again with the new Driver certificate to a server that
/// trusts only the new certificate.
TEST_F(SecurityTest, PoolRefreshUsesNewDriverCertificate) {
    const auto store = this->store();
    const auto old = ASSERT_NIL_P(store->current());
    this->start_server(server_id, {old.certificate});
    Pool pool(store);
    auto cfg = config();
    cfg.server_certificate = server_id.certificate;
    auto conn = ASSERT_NIL_P(pool.acquire(cfg, LOG_PREFIX));
    const auto next = ASSERT_NIL_P(store->regenerate());
    this->start_server(server_id, {next.certificate});
    conn = ASSERT_NIL_P(pool.refresh(conn, cfg, LOG_PREFIX));
    ASSERT_TRUE(conn);
    ASSERT_NIL_P(testutil::simple_read(conn.shared(), "NS=1;S=TestUInt32"));
}

/// @brief it should close the old connection instead of keeping it idle after a
/// refresh.
TEST_F(SecurityTest, PoolRefreshClosesOldConnection) {
    this->start_server(server_id);
    const auto store = this->store();
    Pool pool(store);
    auto cfg = config();
    cfg.server_certificate = server_id.certificate;
    auto conn = ASSERT_NIL_P(pool.acquire(cfg, LOG_PREFIX));
    ASSERT_NIL(store->regenerate().second);
    conn = ASSERT_NIL_P(pool.refresh(conn, cfg, LOG_PREFIX));
    ASSERT_TRUE(conn);
    ASSERT_EQ(pool.size(), 1);
    ASSERT_EQ(pool.available_count(ENDPOINT), 0);
}

/// @brief it should keep the old connection when the server does not trust the new
/// Driver certificate yet.
TEST_F(SecurityTest, PoolRefreshKeepsConnectionWhenRejected) {
    const auto store = this->store();
    const auto old = ASSERT_NIL_P(store->current());
    this->start_server(server_id, {old.certificate});
    Pool pool(store);
    auto cfg = config();
    cfg.server_certificate = server_id.certificate;
    auto conn = ASSERT_NIL_P(pool.acquire(cfg, LOG_PREFIX));
    auto *const first = conn.get();
    const auto next = ASSERT_NIL_P(store->regenerate());
    const auto err = pool.refresh(conn, cfg, LOG_PREFIX).second;
    ASSERT_MATCHES(err, errors::CLIENT_CERTIFICATE_REJECTED);
    ASSERT_EQ(
        err.data,
        "The server rejected the connection's security checks "
        "(BadSecurityChecksFailed). Most often, the server does not trust the client "
        "certificate " +
            next.certificate.thumbprint() +
            ". Approve it in the server's trusted certificates, then connect again."
    );
    ASSERT_EQ(conn.get(), first);
    ASSERT_NIL_P(testutil::simple_read(conn.shared(), "NS=1;S=TestUInt32"));
}

/// @brief it should connect again only to the server that the old connection trusted,
/// even when the configuration pins no certificate.
TEST_F(SecurityTest, PoolRefreshKeepsTrustedServer) {
    this->start_server(server_id);
    const auto store = this->store();
    Pool pool(store);
    const auto cfg = config();
    auto conn = ASSERT_NIL_P(pool.acquire(cfg, LOG_PREFIX));
    auto *const first = conn.get();
    ASSERT_NIL(store->regenerate().second);
    this->start_server(other_id);
    const auto err = pool.refresh(conn, cfg, LOG_PREFIX).second;
    ASSERT_MATCHES(err, errors::SERVER_CERTIFICATE_UNTRUSTED);
    ASSERT_EQ(err.data, mismatch_message(other_id.certificate, server_id.certificate));
    ASSERT_EQ(conn.get(), first);
}
}
