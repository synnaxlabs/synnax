// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std
#include <algorithm>
#include <thread>

/// external
#include "glog/logging.h"
#include "open62541/client_config_default.h"
#include "open62541/client_highlevel.h"

/// internal
#include "driver/opc/connection/connection.h"
#include "driver/opc/errors/errors.h"

namespace driver::opc::connection {
std::pair<Pool::Connection, x::errors::Error>
Pool::acquire(const Config &cfg, const std::string &log_prefix) {
    const std::string key = cfg.endpoint + "|" + cfg.username + "|" +
                            cfg.security_mode + "|" + cfg.security_policy;

    // Try to find a cached connection. We mark it in_use under the lock,
    // then release the lock before running the blocking health probe.
    std::shared_ptr<UA_Client> candidate;
    {
        std::lock_guard<std::mutex> lock(mutex_);
        auto it = connections_.find(key);
        if (it != connections_.end()) {
            for (auto &entry: it->second) {
                if (!entry.in_use) {
                    UA_SessionState session_state;
                    UA_SecureChannelState channel_state;
                    UA_Client_getState(
                        entry.client.get(),
                        &channel_state,
                        &session_state,
                        nullptr
                    );
                    if (session_state == UA_SESSIONSTATE_ACTIVATED) {
                        entry.in_use = true;
                        candidate = entry.client;
                        break;
                    } else {
                        VLOG(1) << log_prefix << "Removing stale connection from pool";
                        entry.client.reset();
                    }
                }
            }

            it->second.erase(
                std::remove_if(
                    it->second.begin(),
                    it->second.end(),
                    [](const Entry &e) { return !e.client; }
                ),
                it->second.end()
            );
        }
    }

    // Health probe runs outside the lock so other threads can acquire
    // different connections concurrently.
    if (candidate) {
        if (const auto err = run_iterate_checked(candidate, log_prefix)) {
            VLOG(1) << log_prefix << "Cached connection failed maintenance, discarding";
            std::lock_guard<std::mutex> lock(mutex_);
            auto it = connections_.find(key);
            if (it != connections_.end()) {
                for (auto &entry: it->second) {
                    if (entry.client == candidate) {
                        entry.client.reset();
                        break;
                    }
                }
                it->second.erase(
                    std::remove_if(
                        it->second.begin(),
                        it->second.end(),
                        [](const Entry &e) { return !e.client; }
                    ),
                    it->second.end()
                );
            }
        } else {
            VLOG(1) << log_prefix << "Reusing connection from pool for "
                    << cfg.endpoint;
            // Reset breaker on successful reuse.
            {
                std::lock_guard<std::mutex> lock(mutex_);
                breakers[key].consecutive_failures = 0;
            }
            return {Connection(candidate, this, key), x::errors::NIL};
        }
    }

    // Serialized connection creation: only one thread connects at a time
    // per endpoint. Other threads wait for the connecting thread to finish,
    // then try to reuse the newly created connection.
    {
        std::unique_lock<std::mutex> lock(mutex_);
        auto &breaker = breakers[key];

        // Check circuit breaker before cleanup_stale_entries: if tripped,
        // fail fast without blocking on UA_Client_disconnect network calls.
        if (breaker.consecutive_failures >= BREAKER_THRESHOLD) {
            if (std::chrono::steady_clock::now() < breaker.cooldown_until) {
                VLOG(1) << log_prefix << "Circuit breaker open, rejecting connection";
                return {
                    Connection(nullptr, nullptr, ""),
                    x::errors::Error(
                        opc::errors::NO_CONNECTION,
                        "circuit breaker open — too many consecutive "
                        "connection failures"
                    )
                };
            }
            // Cooldown expired — allow one probe attempt (half-open).
            VLOG(1) << log_prefix << "Circuit breaker half-open, allowing probe";
        }

        // Breaker is not tripped (or is half-open) — clean up stale entries.
        // Release the lock first: cleanup_stale_entries takes mutex_ internally
        // and calls UA_Client_disconnect which blocks on network I/O.
        lock.unlock();
        cleanup_stale_entries(key, log_prefix);
        lock.lock();
        // Re-check breaker after reacquiring: another thread may have tripped
        // it while we were cleaning up.
        if (breaker.consecutive_failures >= BREAKER_THRESHOLD &&
            std::chrono::steady_clock::now() < breaker.cooldown_until) {
            VLOG(1) << log_prefix << "Circuit breaker open after cleanup, rejecting";
            return {
                Connection(nullptr, nullptr, ""),
                x::errors::Error(
                    opc::errors::NO_CONNECTION,
                    "circuit breaker open — too many consecutive "
                    "connection failures"
                )
            };
        }

        // If another thread is already connecting, wait for it to finish.
        // Then try to reuse its connection instead of creating a duplicate.
        if (breaker.connecting) {
            VLOG(1) << log_prefix << "Waiting for another thread to finish connecting";
            connect_cv.wait(lock, [&breaker] { return !breaker.connecting; });

            // Try to grab the newly created connection.
            auto it = connections_.find(key);
            if (it != connections_.end()) {
                for (auto &entry: it->second) {
                    if (!entry.in_use) {
                        UA_SessionState ss;
                        UA_SecureChannelState cs;
                        UA_Client_getState(entry.client.get(), &cs, &ss, nullptr);
                        if (ss == UA_SESSIONSTATE_ACTIVATED) {
                            entry.in_use = true;
                            VLOG(1) << log_prefix
                                    << "Reusing connection created by "
                                       "another thread";
                            return {
                                Connection(entry.client, this, key),
                                x::errors::NIL
                            };
                        }
                    }
                }
            }

            // No reusable connection available. If breaker is still tripped,
            // fail fast. Otherwise fall through to connect ourselves.
            if (breaker.consecutive_failures >= BREAKER_THRESHOLD) {
                return {
                    Connection(nullptr, nullptr, ""),
                    x::errors::Error(
                        opc::errors::NO_CONNECTION,
                        "circuit breaker open — connecting thread failed, "
                        "too many consecutive connection failures"
                    )
                };
            }
        }

        breaker.connecting = true;
    }

    // Connect outside the lock. Only one thread reaches here per endpoint.
    auto [client, err] = connect(cfg, log_prefix);

    if (err) {
        std::lock_guard<std::mutex> lock(mutex_);
        auto &breaker = breakers[key];
        breaker.connecting = false;
        breaker.consecutive_failures++;
        if (breaker.consecutive_failures >= BREAKER_THRESHOLD) {
            breaker.cooldown_until = std::chrono::steady_clock::now() +
                                     BREAKER_COOLDOWN;
            LOG(WARNING) << log_prefix << "Circuit breaker tripped after "
                         << breaker.consecutive_failures << " consecutive failures";
        }
        connect_cv.notify_all();
        return {Connection(nullptr, nullptr, ""), err};
    }

    // Perform initial connection maintenance with error checking.
    if (const auto iterate_err = run_iterate_checked(client, log_prefix)) {
        LOG(WARNING) << log_prefix << "New connection failed initial maintenance";
        std::lock_guard<std::mutex> lock(mutex_);
        auto &breaker = breakers[key];
        breaker.connecting = false;
        breaker.consecutive_failures++;
        if (breaker.consecutive_failures >= BREAKER_THRESHOLD) {
            breaker.cooldown_until = std::chrono::steady_clock::now() +
                                     BREAKER_COOLDOWN;
            LOG(WARNING) << log_prefix << "Circuit breaker tripped after "
                         << breaker.consecutive_failures << " consecutive failures";
        }
        connect_cv.notify_all();
        return {Connection(nullptr, nullptr, ""), iterate_err};
    }

    // Success — add to pool, reset breaker, notify waiters.
    {
        std::lock_guard<std::mutex> lock(mutex_);
        connections_[key].push_back({client, true});
        auto &breaker = breakers[key];
        breaker.connecting = false;
        breaker.consecutive_failures = 0;
    }
    connect_cv.notify_all();

    VLOG(1) << log_prefix << "Created new connection for " << cfg.endpoint;
    return {Connection(client, this, key), x::errors::NIL};
}

void Pool::release(const std::string &key, std::shared_ptr<UA_Client> client) {
    std::lock_guard<std::mutex> lock(mutex_);

    auto it = connections_.find(key);
    if (it == connections_.end()) { return; }

    for (auto &entry: it->second) {
        if (entry.client == client) {
            UA_SessionState session_state;
            UA_SecureChannelState channel_state;
            UA_Client_getState(client.get(), &channel_state, &session_state, nullptr);
            if (session_state == UA_SESSIONSTATE_ACTIVATED) {
                entry.in_use = false;
                VLOG(1) << "[conn_pool] Returned connection to pool";
            } else {
                entry.client.reset();
                VLOG(1) << "[conn_pool] Discarding disconnected connection";
            }
            break;
        }
    }

    it->second.erase(
        std::remove_if(
            it->second.begin(),
            it->second.end(),
            [](const Entry &e) { return !e.client; }
        ),
        it->second.end()
    );

    if (it->second.empty()) connections_.erase(it);
}

void Pool::cleanup_stale_entries(
    const std::string &key,
    const std::string &log_prefix
) {
    std::vector<std::shared_ptr<UA_Client>> stale;
    {
        std::lock_guard<std::mutex> lock(mutex_);
        auto it = connections_.find(key);
        if (it == connections_.end()) return;
        for (auto &entry: it->second) {
            if (entry.in_use || !entry.client) continue;
            UA_SessionState ss;
            UA_SecureChannelState cs;
            UA_Client_getState(entry.client.get(), &cs, &ss, nullptr);
            if (ss != UA_SESSIONSTATE_ACTIVATED) {
                stale.push_back(entry.client);
                entry.client.reset();
            }
        }
        it->second.erase(
            std::remove_if(
                it->second.begin(),
                it->second.end(),
                [](const Entry &e) { return !e.client; }
            ),
            it->second.end()
        );
        if (it->second.empty()) connections_.erase(it);
    }
    // Disconnect stale clients outside the lock to send CloseSession
    // to the server, freeing server-side session slots.
    for (auto &c: stale) {
        VLOG(1) << log_prefix << "Sending CloseSession for orphaned connection";
        UA_Client_disconnect(c.get());
    }
}

size_t Pool::size() const {
    std::lock_guard<std::mutex> lock(mutex_);
    size_t total = 0;
    for (const auto &[_, entries]: connections_)
        total += entries.size();
    return total;
}

size_t Pool::available_count(const std::string &endpoint) const {
    std::lock_guard<std::mutex> lock(mutex_);
    size_t count = 0;
    for (const auto &[key, entries]: connections_)
        if (key.find(endpoint) == 0)
            for (const auto &entry: entries)
                if (!entry.in_use && entry.client) count++;
    return count;
}

x::errors::Error Pool::run_iterate_checked(
    const std::shared_ptr<UA_Client> &client,
    const std::string &log_prefix
) {
    const UA_StatusCode status = UA_Client_run_iterate(client.get(), 0);
    if (status != UA_STATUSCODE_GOOD) {
        LOG(WARNING) << log_prefix
                     << "run_iterate failed: " << UA_StatusCode_name(status);
        return driver::opc::errors::parse(status);
    }

    UA_SessionState session_state;
    UA_SecureChannelState channel_state;
    UA_Client_getState(client.get(), &channel_state, &session_state, nullptr);

    if (session_state != UA_SESSIONSTATE_ACTIVATED) {
        LOG(WARNING) << log_prefix << "Session no longer activated after run_iterate";
        return x::errors::Error(
            opc::errors::NO_CONNECTION,
            "session deactivated during maintenance"
        );
    }

    UA_Variant value;
    UA_Variant_init(&value);
    const UA_StatusCode read_status = UA_Client_readValueAttribute(
        client.get(),
        UA_NODEID_NUMERIC(0, UA_NS0ID_SERVER_SERVERSTATUS_CURRENTTIME),
        &value
    );
    UA_Variant_clear(&value);

    if (read_status != UA_STATUSCODE_GOOD) {
        LOG(WARNING) << log_prefix
                     << "Health probe failed: " << UA_StatusCode_name(read_status);
        return driver::opc::errors::parse(read_status);
    }

    return x::errors::NIL;
}

}
