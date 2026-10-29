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
#include <cstddef>
#include <functional>
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

/// @brief a way that a server accepts connections.
struct Endpoint {
    /// @brief the URL of the endpoint.
    std::string url;
    /// @brief None, Sign, or SignAndEncrypt.
    std::string security_mode;
    /// @brief the name of the security policy, such as Basic256Sha256.
    std::string security_policy;
    /// @brief the ways a user can log in: anonymous, username, certificate, or
    /// issued_token.
    std::vector<std::string> user_token_types;
    /// @brief how secure the server rates the endpoint. Higher is more secure.
    uint8_t security_level = 0;

    /// @returns the endpoint as JSON.
    [[nodiscard]] x::json::json to_json() const;
};

/// @brief asks the server at cfg.endpoint for its endpoints. Opens a channel with no
/// security and no Session, and closes it before it returns.
/// @returns the error that errors::parse_connect gives if the request fails.
std::pair<std::vector<Endpoint>, x::errors::Error> endpoints(const Config &cfg);

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

/// @brief what the Driver reads from a server at the start of each Session.
struct Server {
    /// @brief the namespace URIs of the server, by index.
    std::vector<std::string> namespaces;
    /// @brief the most nodes that one Read request can name. 0 means no limit.
    uint32_t max_nodes_per_read = 0;
    /// @brief the most nodes that one Write request can name. 0 means no limit.
    uint32_t max_nodes_per_write = 0;
    /// @brief the most nodes that one Browse or BrowseNext request can name. 0 means no
    /// limit.
    uint32_t max_nodes_per_browse = 0;
    /// @brief the most monitored items that one request can create. 0 means no limit.
    uint32_t max_monitored_items_per_call = 0;
    /// @brief the most nodes that one HistoryRead request for values can name. 0 means
    /// no limit.
    uint32_t max_nodes_per_history_read_data = 0;
};

/// @brief handles one event of a subscription: the values of the select clauses of its
/// filter, in order.
using EventHandler = std::function<void(const UA_KeyValueMap &fields)>;

/// @brief handles one change of a node of a subscription: the place of the node in the
/// subscribe request, and its value.
using ChangeHandler = std::function<void(std::size_t node, const UA_DataValue &value)>;

/// @brief the intervals of a subscription to the changes of nodes.
struct Intervals {
    /// @brief how often the server samples each node.
    x::telem::TimeSpan sampling;
    /// @brief how often the server sends the changes.
    x::telem::TimeSpan publishing;
};

/// @brief what the server did with a request to subscribe to the changes of nodes.
struct Monitored {
    /// @brief the error of each node that the server did not monitor, in order.
    std::vector<x::errors::Error> errors;
    /// @brief the intervals that the server chose. The sampling interval is the longest
    /// of any node.
    Intervals intervals;
};

/// @brief the subscription that a connection holds. Its address stays fixed for the
/// life of the subscription, since the client passes it to each callback.
struct Subscription {
    /// @brief the ID that the server gave the subscription.
    UA_UInt32 id = 0;
    /// @brief runs for each event of the subscription.
    EventHandler handle_event;
    /// @brief runs for each change of a node of the subscription.
    ChangeHandler handle_change;
    /// @brief the place of each monitored node in its subscribe request. The client
    /// passes a pointer to one of them to each callback, so the list never grows.
    std::vector<std::size_t> nodes;
    /// @brief the bad status with which the server ended the subscription, or
    /// BadTimeout when the server stopped answering.
    UA_StatusCode status = UA_STATUSCODE_GOOD;
};

class Pool {
public:
    class Connection {
    public:
        Connection(
            std::shared_ptr<UA_Client> client,
            Pool *pool,
            std::string key,
            std::shared_ptr<const Server> server = nullptr
        ):
            client_(std::move(client)),
            pool_(pool),
            key_(std::move(key)),
            server_(std::move(server)) {}

        ~Connection() { this->close(); }

        Connection(const Connection &) = delete;
        Connection &operator=(const Connection &) = delete;

        Connection(Connection &&other) noexcept:
            client_(std::move(other.client_)),
            pool_(other.pool_),
            key_(std::move(other.key_)),
            server_(std::move(other.server_)),
            subscription_(std::move(other.subscription_)) {
            other.pool_ = nullptr;
        }

        Connection &operator=(Connection &&other) noexcept {
            if (this != &other) {
                this->close();
                client_ = std::move(other.client_);
                pool_ = other.pool_;
                key_ = std::move(other.key_);
                server_ = std::move(other.server_);
                subscription_ = std::move(other.subscription_);
                other.pool_ = nullptr;
            }
            return *this;
        }

        [[nodiscard]] UA_Client *get() const { return client_.get(); }
        [[nodiscard]] std::shared_ptr<UA_Client> shared() const { return client_; }
        explicit operator bool() const { return client_ != nullptr; }

        /// @returns what the Driver read from the server when the client last
        /// connected. Requires a connection from Pool::acquire, as do read and write.
        [[nodiscard]] const Server &server() const { return *server_; }

        /// @brief reads the attributes that req names, in as many requests as the
        /// server's limit needs.
        /// @returns errors::NO_CONNECTION without sending req when the connection is
        /// lost, the error of the service result, or errors::CRITICAL when the server
        /// returns a result count that does not match req.
        [[nodiscard]] std::pair<types::ReadResponse, x::errors::Error>
        read(const UA_ReadRequest &req) const;

        /// @brief writes the values in req, in as many requests as the server's limit
        /// needs.
        /// @returns errors::NO_CONNECTION without sending req when the connection is
        /// lost, the error of the service result, or errors::CRITICAL when the server
        /// returns a result count that does not match req.
        [[nodiscard]] std::pair<types::WriteResponse, x::errors::Error>
        write(const UA_WriteRequest &req) const;

        /// @brief finds the children of each node: the targets of its forward
        /// hierarchical references. Follows continuation points until the server
        /// returns every child, in as many requests as the server's limit needs. Skips
        /// a child on another server or in a namespace the server does not list.
        /// @returns one result for each node, in order. Returns errors::NO_CONNECTION
        /// when the connection is lost, the error of a service result, or
        /// errors::CRITICAL when the server returns a result count that does not match
        /// the request, or continuation points with no children.
        [[nodiscard]] std::pair<std::vector<types::BrowseResult>, x::errors::Error>
        browse(const std::vector<UA_NodeId> &nodes) const;

        /// @brief calls one method on the server.
        /// @returns errors::NO_CONNECTION without sending the call when the connection
        /// is lost, the error of the service result, or errors::CRITICAL when the
        /// server returns other than one result.
        [[nodiscard]] std::pair<types::CallResponse, x::errors::Error>
        call(const UA_CallMethodRequest &method) const;

        /// @brief subscribes to the events of each notifier that pass filter, and
        /// replaces the earlier subscription of the connection. handle runs for each
        /// event, on the thread that calls iterate or another method of the connection.
        /// The subscription ends when the connection goes back to its pool.
        /// @returns the error of each notifier the server did not monitor, in order,
        /// and an error when the subscription failed. Returns errors::NO_CONNECTION
        /// when the connection is lost.
        [[nodiscard]] std::pair<std::vector<x::errors::Error>, x::errors::Error>
        subscribe(
            const std::vector<UA_NodeId> &notifiers,
            const UA_EventFilter &filter,
            EventHandler handle
        );

        /// @brief subscribes to the changes of the values of nodes, and replaces the
        /// earlier subscription of the connection. Each node keeps the changes of one
        /// publishing interval, and drops the oldest past that. handle runs for each
        /// change, as for events, and first for the current value of each node. Creates
        /// the nodes in as many requests as the server's limit needs.
        /// @returns errors::NO_CONNECTION when the connection is lost, or the error of
        /// a service result. A lost server ends the subscription with BadTimeout, which
        /// iterate returns as errors::UNREACHABLE.
        [[nodiscard]] std::pair<Monitored, x::errors::Error> subscribe(
            const std::vector<UA_NodeId> &nodes,
            const Intervals &intervals,
            ChangeHandler handle
        );

        /// @brief reads the history that req names, in as many requests as the server's
        /// limit needs. Releases each continuation point that the server returns, so
        /// the Session holds none after the call. A result that keeps a continuation
        /// point has more history, which a later request must read by time.
        /// @returns errors::NO_CONNECTION without sending req when the connection is
        /// lost, the error of the service result, or errors::CRITICAL when the server
        /// returns a result count that does not match req.
        [[nodiscard]] std::pair<types::HistoryReadResponse, x::errors::Error>
        read_history(const UA_HistoryReadRequest &req) const;

        /// @returns the ID of the subscription of the connection, or 0 when it has
        /// none.
        [[nodiscard]] UA_UInt32 subscription_id() const;

        /// @brief handles the notifications that the server sent, waiting up to timeout
        /// for the first one.
        /// @returns errors::NO_CONNECTION when the connection is lost, or the status
        /// with which the server ended the subscription.
        [[nodiscard]] x::errors::Error iterate(x::telem::TimeSpan timeout) const;

    private:
        /// @brief ends the subscription, then gives the client back to the pool.
        void close();

        /// @brief creates a subscription for sub, which replaces the earlier one.
        /// @returns the publishing interval that the server chose.
        [[nodiscard]] std::pair<x::telem::TimeSpan, x::errors::Error>
        create(std::unique_ptr<Subscription> sub, x::telem::TimeSpan publishing);

        /// @brief releases on the server the continuation points of res, the response
        /// to req.
        void release(
            const UA_HistoryReadRequest &req,
            const UA_HistoryReadResponse &res
        ) const;

        /// @brief ends the subscription of the connection. When the server does not
        /// delete it, closes the Session, since a client that the pool hands out again
        /// must not deliver events to a handler that no longer exists.
        void unsubscribe();

        /// @brief browses count nodes in one Browse request, then follows their
        /// continuation points with BrowseNext. Fills count results.
        [[nodiscard]] x::errors::Error browse(
            const UA_NodeId *nodes,
            size_t count,
            types::BrowseResult *results
        ) const;

        std::shared_ptr<UA_Client> client_;
        Pool *pool_;
        std::string key_;
        std::shared_ptr<const Server> server_;
        std::unique_ptr<Subscription> subscription_;

        friend class Pool;
    };

    static constexpr int BREAKER_THRESHOLD = 3;
    static constexpr std::chrono::seconds MIN_COOLDOWN{5};
    static constexpr std::chrono::seconds MAX_COOLDOWN{30};
    static constexpr std::chrono::seconds IDLE_TIMEOUT{60};

    /// @param store the source of the Driver certificate for secured connections.
    /// @param idle_timeout how long a connection stays open while no one uses it.
    explicit Pool(
        std::shared_ptr<certificate::Store> store,
        std::chrono::milliseconds idle_timeout = IDLE_TIMEOUT
    );
    ~Pool() = default;

    Pool(const Pool &) = delete;
    Pool &operator=(const Pool &) = delete;

    /// @brief returns a connection to the server. It reuses an idle connection with the
    /// same configuration and client identity, and connects a lost one again in place,
    /// so the server keeps one Session for it. A new connection with no server
    /// certificate in its Config trusts the first one that the server presents.
    ///
    /// The Pool makes one connection attempt to a server at a time. After
    /// BREAKER_THRESHOLD failures in a row, it refuses attempts for a cooldown that
    /// starts at MIN_COOLDOWN and doubles up to MAX_COOLDOWN.
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

    /// @brief closes the connections that no one has used for the idle timeout. Blocks
    /// while it sends CloseSession to their servers.
    void close_idle();

    /// @returns the time of the last reply from the server on a connection with the
    /// configuration, or a zero timestamp if none replied.
    x::telem::TimeStamp last_reply(const Config &cfg) const;

    size_t size() const;
    size_t available_count(const std::string &endpoint) const;

private:
    struct Entry {
        std::shared_ptr<UA_Client> client;
        /// @brief what the Driver read from the server when the Session started.
        std::shared_ptr<const Server> server;
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
        std::chrono::seconds cooldown = MIN_COOLDOWN;
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
        void failed(const x::errors::Error &err, const std::string &log_prefix);
        void succeeded();
    };

    std::shared_ptr<certificate::Store> store;
    std::chrono::milliseconds idle_timeout;
    mutable std::mutex mutex_;
    /// @brief notified when a connecting thread finishes.
    std::condition_variable connect_cv;
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

    /// @returns the entry of client. Requires mutex_ and an entry in use.
    Entry &entry(const std::string &key, const std::shared_ptr<UA_Client> &client);

    /// @brief records key as the client certificate that settings uses now. Removes
    /// the idle connections of the certificate it replaces and retires the ones in use.
    /// The caller holds mutex_ and closes the returned connections outside it.
    std::vector<std::shared_ptr<UA_Client>>
    retire(const std::string &settings, const std::string &key);

    friend class Connection;
};
}
