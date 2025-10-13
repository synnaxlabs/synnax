// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <mutex>
#include <string>
#include <unordered_map>
#include <vector>

#include "open62541/client.h"

#include "x/cpp/xerrors/errors.h"

#include "driver/opc/util/util.h"

namespace util {

class ConnectionPool {
public:
    class Connection {
    public:
        Connection(
            std::shared_ptr<UA_Client> client,
            ConnectionPool *pool,
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
        ConnectionPool *pool_;
        std::string key_;
    };

    ConnectionPool() = default;
    ~ConnectionPool() = default;

    ConnectionPool(const ConnectionPool &) = delete;
    ConnectionPool &operator=(const ConnectionPool &) = delete;

    std::pair<Connection, xerrors::Error>
    acquire(const ConnectionConfig &cfg, const std::string &log_prefix);

    size_t size() const;
    size_t available_count(const std::string &endpoint) const;

private:
    struct PoolEntry {
        std::shared_ptr<UA_Client> client;
        bool in_use;
    };

    mutable std::mutex mutex_;
    std::unordered_map<std::string, std::vector<PoolEntry>> connections_;

    void release(const std::string &key, std::shared_ptr<UA_Client> client);
    friend class Connection;
};

}
