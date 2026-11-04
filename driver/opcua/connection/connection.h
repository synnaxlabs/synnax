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
#include <thread>
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
#include "driver/opcua/types/types.h"

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
    /// @brief the SecureChannel lifetime in milliseconds. 0 selects 10 minutes.
    uint32_t secure_channel_lifetime_ms = 0;
    /// @brief the Session timeout in milliseconds. 0 selects 20 minutes.
    uint32_t session_timeout_ms = 0;
    /// @brief the timeout of each request and connection attempt in milliseconds. 0
    /// selects 5 seconds.
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

/// @brief closes the client's channel and connects it again with the security it was
/// created with. The server activates the client's Session again, or the client creates
/// a new one if the server dropped it.
x::errors::Error
reconnect(const std::shared_ptr<UA_Client> &client, const std::string &endpoint);

/// @returns errors::NO_CONNECTION if the client has no open channel or no active
/// Session. Check it before each service call: a call on a lost connection connects
/// again by itself, outside the Pool's breaker.
x::errors::Error lost(UA_Client *client);

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

        /// @brief reads the attributes that req names.
        /// @returns errors::NO_CONNECTION without sending req when the connection is
        /// lost, or the error of the service result.
        [[nodiscard]] std::pair<types::ReadResponse, x::errors::Error>
        read(const UA_ReadRequest &req) const;

        /// @brief writes the values in req.
        /// @returns errors::NO_CONNECTION without sending req when the connection is
        /// lost, or the error of the service result.
        [[nodiscard]] std::pair<types::WriteResponse, x::errors::Error>
        write(const UA_WriteRequest &req) const;

    private:
        /// @returns the error of the service result of a call. A call that fails for a
        /// lost connection closes the channel, so the next acquire connects again at
        /// once instead of probing first. Any other result records a reply.
        [[nodiscard]] x::errors::Error check(UA_StatusCode result) const;

        std::shared_ptr<UA_Client> client_;
        Pool *pool_;
        std::string key_;

        friend class Pool;
    };

    /// @brief how long a Pool keeps idle connections and pauses failed attempts.
    struct Timing {
        /// @brief how long a connection stays open while no one uses it.
        std::chrono::milliseconds idle_timeout = IDLE_TIMEOUT;
        /// @brief the first pause of the attempts to a server after three failures in a
        /// row.
        std::chrono::milliseconds min_cooldown = MIN_COOLDOWN;
        /// @brief the longest pause. Each pause in a row doubles up to it.
        std::chrono::milliseconds max_cooldown = MAX_COOLDOWN;
    };

    /// @param store the source of the Driver certificate for secured connections.
    explicit Pool(std::shared_ptr<certificate::Store> store);
    Pool(std::shared_ptr<certificate::Store> store, Timing timing);
    /// @brief blocks while it sends CloseSession for each connection.
    ~Pool();

    Pool(const Pool &) = delete;
    Pool &operator=(const Pool &) = delete;

    /// @brief returns a connection to the server. It reuses an idle connection with the
    /// same configuration and client identity, and connects a lost one again in place,
    /// so the server keeps one Session for it. A new connection with no server
    /// certificate in its Config trusts the first one that the server presents.
    ///
    /// The Pool makes one connection attempt to a server at a time. After three
    /// failures in a row, it refuses attempts for a cooldown that starts at
    /// Timing::min_cooldown and doubles up to Timing::max_cooldown. A connection that
    /// no one uses for Timing::idle_timeout closes.
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

    /// @returns the time of the last reply from the server on a connection with the
    /// configuration, or a zero timestamp if none replied.
    x::telem::TimeStamp last_reply(const Config &cfg) const;

    size_t size() const;
    size_t available_count(const std::string &endpoint) const;

private:
    static constexpr int BREAKER_THRESHOLD = 3;
    static constexpr std::chrono::seconds MIN_COOLDOWN{5};
    static constexpr std::chrono::seconds MAX_COOLDOWN{30};
    static constexpr std::chrono::seconds IDLE_TIMEOUT{60};

    struct Entry {
        std::shared_ptr<UA_Client> client;
        bool in_use;
        /// @brief whether the connection uses a replaced client certificate. A retired
        /// connection closes when it is released.
        bool retired;
        /// @brief when the connection was last released.
        std::chrono::steady_clock::time_point idle_since;
    };

    /// @brief the connection attempts to one server. Failures in a row open the
    /// breaker, which refuses attempts until cooldown_until, so that a server that
    /// refuses Sessions is not flooded with them.
    struct BreakerState {
        int consecutive_failures = 0;
        std::chrono::steady_clock::time_point cooldown_until;
        /// @brief the next pause, or zero for the first one.
        std::chrono::milliseconds cooldown{0};
        /// @brief true while a thread connects. Other threads wait on connect_cv.
        bool connecting = false;
        /// @brief the trusted server certificate of the counted failures. A new
        /// one resets the count.
        std::optional<certificate::Certificate> trusted;
        x::errors::Error last_error = x::errors::NIL;

        /// @returns the error for an acquire that the open breaker refuses. A
        /// security error repeats, since waiting does not fix it.
        [[nodiscard]] x::errors::Error open_error() const;
        [[nodiscard]] bool open() const;
        void failed(
            const x::errors::Error &err,
            const Timing &timing,
            const std::string &log_prefix
        );
        void succeeded();
    };

    std::shared_ptr<certificate::Store> store;
    Timing timing;
    mutable std::mutex mutex_;
    /// @brief notified when a connecting thread finishes.
    std::condition_variable connect_cv;
    /// @brief notified when a connection goes idle or the Pool closes.
    std::condition_variable idle_cv;
    bool closing = false;
    std::unordered_map<std::string, std::vector<Entry>> connections_;
    std::unordered_map<std::string, BreakerState> breakers;
    /// @brief the key of the client certificate that each group of settings uses now.
    std::unordered_map<std::string, std::string> current_keys;
    /// @brief the time of the last reply from each server, by pool key.
    std::unordered_map<std::string, x::telem::TimeStamp> replies;

    void release(const std::string &key, const std::shared_ptr<UA_Client> &client);

    /// @brief records a reply from the server of key. Requires mutex_ not held.
    void replied(const std::string &key);

    /// @brief marks an idle entry of key that trusts cfg as in use, preferring one with
    /// an active Session. Requires mutex_.
    std::shared_ptr<UA_Client> take(const std::string &key, const Config &cfg);

    /// @brief marks the entry of client as idle again. Requires mutex_.
    void put_back(const std::string &key, const std::shared_ptr<UA_Client> &client);

    /// @brief records key as the client certificate that settings uses now. Removes
    /// the idle connections of the certificate it replaces and retires the ones in use.
    /// The caller holds mutex_ and closes the returned connections outside it.
    std::vector<std::shared_ptr<UA_Client>>
    retire(const std::string &settings, const std::string &key);

    /// @brief closes each connection that no one used for the idle timeout, outside
    /// mutex_, until the Pool closes.
    void evict();

    /// @brief runs evict. It starts last, after the members it uses.
    std::thread evictor;

    friend class Connection;
};
}
