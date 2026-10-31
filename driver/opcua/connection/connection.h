// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <chrono>
#include <condition_variable>
#include <memory>
#include <mutex>
#include <optional>
#include <string>
#include <unordered_map>
#include <utility>
#include <vector>

#include "open62541/client.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/mono_clock.h"
#include "x/cpp/telem/telem.h"

#include "driver/opcua/certificate/certificate.h"
#include "driver/opcua/certificate/store.h"

namespace driver::opcua::connection {
/// @brief the configuration for an OPC UA connection.
struct Config {
    /// @brief the endpoint of the OPC UA server.
    std::string endpoint;
    /// @brief the username to use for authentication. Optional.
    std::string username;
    /// @brief the password to use for authentication. Optional.
    std::string password;
    /// @brief the security mode.
    std::string security_mode = "None";
    /// @brief the security policy.
    std::string security_policy = "None";
    /// @brief the path of a client certificate on the Driver's computer. Empty, with
    /// an empty client_private_key, selects the Driver certificate.
    std::string client_cert;
    /// @brief the path of the private key for client_cert on the Driver's computer.
    std::string client_private_key;
    /// @brief the trusted server certificate. Empty trusts no certificate, unless
    /// the connection trusts the first one it receives.
    std::optional<certificate::Certificate> server_certificate;
    /// @brief whether the connection accepts any server certificate.
    bool certificate_validation_disabled = false;
    /// @brief SecureChannel lifetime in milliseconds. 0 = use default (2 hours).
    /// For testing, can be set to short values like 10000 (10 seconds).
    uint32_t secure_channel_lifetime_ms = 0;
    /// @brief Session timeout in milliseconds. 0 = use default (4 hours).
    /// For testing, can be set to short values like 20000 (20 seconds).
    uint32_t session_timeout_ms = 0;
    /// @brief General client timeout in milliseconds. 0 = use default (2 hours).
    uint32_t client_timeout_ms = 0;

    Config() = default;

    explicit Config(x::json::Parser parser):
        endpoint(parser.field<std::string>("endpoint")),
        username(parser.field<std::string>("username", "")),
        password(parser.field<std::string>("password", "")),
        security_mode(parser.field<std::string>("security_mode", "None")),
        security_policy(parser.field<std::string>("security_policy", "None")),
        client_cert(parser.field<std::string>("client_certificate", "")),
        client_private_key(parser.field<std::string>("client_private_key", "")),
        certificate_validation_disabled(
            parser.field<bool>("certificate_validation_disabled", false)
        ),
        secure_channel_lifetime_ms(
            parser.field<uint32_t>("secure_channel_lifetime_ms", 0)
        ),
        session_timeout_ms(parser.field<uint32_t>("session_timeout_ms", 0)),
        client_timeout_ms(parser.field<uint32_t>("client_timeout_ms", 0)) {
        const auto server = parser.field<std::string>("server_certificate", "");
        // A value that is not PEM is a path from an earlier release, which had no
        // effect.
        if (!certificate::Certificate::is_pem(server)) return;
        auto [cert, err] = certificate::Certificate::parse(server);
        if (err) {
            parser.field_err("server_certificate", err.data);
            return;
        }
        this->server_certificate = std::move(cert);
    }

    /// @returns true if the connection signs or encrypts its messages.
    [[nodiscard]] bool secured() const { return this->security_policy != "None"; }

    /// @returns true if the connection presents certificate files instead of the
    /// Driver certificate.
    [[nodiscard]] bool uses_files() const {
        return !this->client_cert.empty() || !this->client_private_key.empty();
    }
};

/// @brief the security inputs of a connection that do not come from its Config.
struct Security {
    /// @brief the certificate and private key that the client presents. Required
    /// when the connection is secured.
    std::optional<certificate::Identity> identity;
    /// @brief whether a Config with no server certificate trusts the first one that
    /// the server presents. When false, such a connection rejects every server
    /// certificate.
    bool first_certificate_trusted = false;
    /// @brief returns the time to check the server certificate's validity against.
    x::telem::NowFunc now = x::telem::TimeStamp::now;
};

/// @brief resolves the client identity for the configuration: its certificate files,
/// or the current Driver certificate. Leaves the identity empty for an unsecured
/// configuration.
/// @returns errors::CLIENT_CERTIFICATE_INVALID if the files cannot be read, or
/// errors::DRIVER_CERTIFICATE_FAILED if the Driver certificate cannot be loaded.
std::pair<Security, x::errors::Error>
resolve_security(const Config &cfg, certificate::Store &store);

/// @brief connects a new client to the server.
/// @returns the client even on failure, so that server_certificate can read the
/// certificate that the server presented. On failure, the error is
/// errors::SERVER_CERTIFICATE_UNTRUSTED or errors::SERVER_CERTIFICATE_TIME_INVALID if
/// the client rejected the server certificate, or errors::CLIENT_CERTIFICATE_REJECTED
/// if the server rejected the client certificate.
std::pair<std::shared_ptr<UA_Client>, x::errors::Error>
connect(const Config &cfg, const Security &sec, std::string log_prefix);

/// @brief connects the client again with the security it was created with.
x::errors::Error
reconnect(const std::shared_ptr<UA_Client> &client, const std::string &endpoint);

/// @returns the certificate that the server presented on the client's last
/// connection attempt, or nothing if the connection is not secured.
std::optional<certificate::Certificate> server_certificate(UA_Client *client);

/// @returns the server certificate that the client trusts: the configured one, or the
/// first one the server presented. Nothing if the client trusts any certificate.
std::optional<certificate::Certificate> trusted_certificate(UA_Client *client);

class Pool {
public:
    class Connection {
    public:
        Connection(std::shared_ptr<UA_Client> client, Pool *pool, std::string key):
            client_(std::move(client)), pool_(pool), key_(std::move(key)) {}

        ~Connection() {
            if (pool_ && client_) { pool_->release(key_, client_); }
        }

        Connection(const Connection &) = delete;
        Connection &operator=(const Connection &) = delete;

        Connection(Connection &&other) noexcept:
            client_(std::move(other.client_)),
            pool_(other.pool_),
            key_(std::move(other.key_)) {
            other.pool_ = nullptr;
        }

        Connection &operator=(Connection &&other) noexcept {
            if (this != &other) {
                if (pool_ && client_) { pool_->release(key_, client_); }
                client_ = std::move(other.client_);
                pool_ = other.pool_;
                key_ = std::move(other.key_);
                other.pool_ = nullptr;
            }
            return *this;
        }

        [[nodiscard]] UA_Client *get() const { return client_.get(); }
        [[nodiscard]] std::shared_ptr<UA_Client> shared() const { return client_; }
        explicit operator bool() const { return client_ != nullptr; }

    private:
        std::shared_ptr<UA_Client> client_;
        Pool *pool_;
        std::string key_;

        friend class Pool;
    };

    /// @param store the source of the Driver certificate for secured connections.
    explicit Pool(std::shared_ptr<certificate::Store> store);
    ~Pool() = default;

    Pool(const Pool &) = delete;
    Pool &operator=(const Pool &) = delete;

    /// @brief returns a connection to the server, reusing an idle one with the same
    /// configuration and client identity. A new connection with no server
    /// certificate in its Config trusts the first one that the server presents.
    std::pair<Connection, x::errors::Error>
    acquire(const Config &cfg, const std::string &log_prefix);

    /// @brief replaces conn with a new connection when the configuration now resolves
    /// to another client certificate, such as after the Driver certificate changes.
    /// The new connection trusts only the server certificate that conn trusts. Leaves
    /// conn unchanged when the client certificate is the same or the new connection
    /// fails.
    /// @returns true if it replaced conn.
    std::pair<bool, x::errors::Error>
    refresh(Connection &conn, const Config &cfg, const std::string &log_prefix);

    size_t size() const;
    size_t available_count(const std::string &endpoint) const;

private:
    struct Entry {
        std::shared_ptr<UA_Client> client;
        bool in_use;
        /// @brief whether the connection uses a replaced client certificate. A retired
        /// connection closes when it is released.
        bool retired;
    };

    /// @brief Per-endpoint circuit breaker state. Tracks consecutive
    /// connection failures and enforces a cooldown period after
    /// BREAKER_THRESHOLD failures to prevent session exhaustion.
    struct BreakerState {
        int consecutive_failures = 0;
        std::chrono::steady_clock::time_point cooldown_until;
        /// @brief True while a thread is actively connecting. Other
        /// threads wait on connect_cv instead of creating parallel
        /// connections that could exhaust the server's session table.
        bool connecting = false;
        /// @brief the trusted server certificate of the counted failures. A new
        /// one resets the count.
        std::optional<certificate::Certificate> trusted;
        x::errors::Error last_error = x::errors::NIL;

        /// @returns the error for an acquire that the open breaker refuses. A
        /// security error repeats, since waiting does not fix it.
        [[nodiscard]] x::errors::Error open_error() const;
    };

    static constexpr int BREAKER_THRESHOLD = 3;
    static constexpr std::chrono::milliseconds BREAKER_COOLDOWN{5000};

    std::shared_ptr<certificate::Store> store;
    mutable std::mutex mutex_;
    /// @brief notified when a connecting thread finishes, waking threads
    /// that are waiting for a reusable connection.
    std::condition_variable connect_cv;
    std::unordered_map<std::string, std::vector<Entry>> connections_;
    std::unordered_map<std::string, BreakerState> breakers;
    /// @brief the key of the client certificate that each group of settings uses now.
    std::unordered_map<std::string, std::string> current_keys;

    void release(const std::string &key, const std::shared_ptr<UA_Client> &client);

    /// @brief records key as the client certificate that settings uses now. Removes
    /// the idle connections of the certificate it replaces and retires the ones in use.
    /// The caller holds mutex_ and closes the returned connections outside it.
    std::vector<std::shared_ptr<UA_Client>>
    retire(const std::string &settings, const std::string &key);

    /// @brief collects stale (non-activated) connections under the lock,
    /// then sends CloseSession outside the lock to free server-side
    /// session slots that would otherwise be orphaned.
    void cleanup_stale_entries(const std::string &key, const std::string &log_prefix);

    /// @brief Performs connection maintenance via run_iterate and verifies
    /// session state.
    x::errors::Error run_iterate_checked(
        const std::shared_ptr<UA_Client> &client,
        const std::string &log_prefix
    );

    friend class Connection;
};
}
