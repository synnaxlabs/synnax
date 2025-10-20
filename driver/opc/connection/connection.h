// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <mutex>
#include <string>
#include <unordered_map>
#include <vector>

/// external
#include "open62541/client.h"

/// module
#include "x/cpp/xerrors/errors.h"
#include "x/cpp/xjson/xjson.h"

namespace opc::connection {
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
    /// @brief the client certificate used to sign and encrypt messages. Only required
    /// if the security policy is not "None".
    std::string client_cert;
    /// @brief the client private key used to sign and encrypt messages. Only required
    /// if the security policy is not "None".
    std::string client_private_key;
    /// @brief a trusted server certificate. Only required if the security policy is
    /// not "None".
    std::string server_cert;
    /// @brief SecureChannel lifetime in milliseconds. 0 = use default (2 hours).
    /// For testing, can be set to short values like 10000 (10 seconds).
    uint32_t secure_channel_lifetime_ms = 0;
    /// @brief Session timeout in milliseconds. 0 = use default (4 hours).
    /// For testing, can be set to short values like 20000 (20 seconds).
    uint32_t session_timeout_ms = 0;
    /// @brief General client timeout in milliseconds. 0 = use default (2 hours).
    uint32_t client_timeout_ms = 0;

    Config() = default;

    explicit Config(xjson::Parser parser):
        endpoint(parser.required<std::string>("endpoint")),
        username(parser.optional<std::string>("username", "")),
        password(parser.optional<std::string>("password", "")),
        security_mode(parser.optional<std::string>("security_mode", "None")),
        security_policy(parser.optional<std::string>("security_policy", "None")),
        client_cert(parser.optional<std::string>("client_certificate", "")),
        client_private_key(parser.optional<std::string>("client_private_key", "")),
        server_cert(parser.optional<std::string>("server_certificate", "")),
        secure_channel_lifetime_ms(
            parser.optional<uint32_t>("secure_channel_lifetime_ms", 0)
        ),
        session_timeout_ms(parser.optional<uint32_t>("session_timeout_ms", 0)),
        client_timeout_ms(parser.optional<uint32_t>("client_timeout_ms", 0)) {}

    json to_json() const {
        return {
            {"endpoint", endpoint},
            {"username", username},
            {"password", password},
            {"security_mode", security_mode},
            {"security_policy", security_policy},
            {"client_certificate", client_cert},
            {"client_private_key", client_private_key}
        };
    }
};

std::pair<std::shared_ptr<UA_Client>, xerrors::Error>
connect(const Config &cfg, std::string log_prefix);

xerrors::Error
reconnect(const std::shared_ptr<UA_Client> &client, const std::string &endpoint);

class Pool {
public:
    class Connection {
    public:
        Connection(
            std::shared_ptr<UA_Client> client,
            Pool *pool,
            const std::string &key
        ):
            client_(std::move(client)), pool_(pool), key_(key) {}

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

        UA_Client *get() const { return client_.get(); }
        std::shared_ptr<UA_Client> shared() const { return client_; }
        explicit operator bool() const { return client_ != nullptr; }

    private:
        std::shared_ptr<UA_Client> client_;
        Pool *pool_;
        std::string key_;
    };

    Pool() = default;
    ~Pool() = default;

    Pool(const Pool &) = delete;
    Pool &operator=(const Pool &) = delete;

    std::pair<Connection, xerrors::Error>
    acquire(const Config &cfg, const std::string &log_prefix);

    size_t size() const;
    size_t available_count(const std::string &endpoint) const;

private:
    struct Entry {
        std::shared_ptr<UA_Client> client;
        bool in_use;
    };

    mutable std::mutex mutex_;
    std::unordered_map<std::string, std::vector<Entry>> connections_;

    void release(const std::string &key, std::shared_ptr<UA_Client> client);
    friend class Connection;
};
}
