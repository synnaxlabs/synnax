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

/// external
#include "glog/logging.h"
#include "open62541/client_config_default.h"
#include "open62541/client_highlevel.h"

/// internal
#include "driver/opc/connection/connection.h"
#include "driver/opc/errors/errors.h"

namespace opc::connection {
std::pair<Pool::Connection, xerrors::Error>
Pool::acquire(const Config &cfg, const std::string &log_prefix) {
    const std::string key = cfg.endpoint + "|" + cfg.username + "|" +
                            cfg.security_mode + "|" + cfg.security_policy;

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
                        // Perform connection maintenance with error checking
                        if (const auto err = run_iterate_checked(
                                entry.client,
                                log_prefix
                            )) {
                            VLOG(1)
                                << log_prefix
                                << "Cached connection failed maintenance, trying next";
                            entry.client.reset();
                            continue;
                        }
                        entry.in_use = true;
                        VLOG(1) << log_prefix << "Reusing connection from pool for "
                                << cfg.endpoint;
                        return {Connection(entry.client, this, key), xerrors::NIL};
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

    auto [client, err] = connect(cfg, log_prefix);
    if (err) { return {Connection(nullptr, nullptr, ""), err}; }

    // Perform initial connection maintenance with error checking
    if (const auto iterate_err = run_iterate_checked(client, log_prefix)) {
        LOG(WARNING) << log_prefix << "New connection failed initial maintenance";
        return {Connection(nullptr, nullptr, ""), iterate_err};
    }

    {
        std::lock_guard<std::mutex> lock(mutex_);
        connections_[key].push_back({client, true});
    }

    VLOG(1) << log_prefix << "Created new connection for " << cfg.endpoint;
    return {Connection(client, this, key), xerrors::NIL};
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

xerrors::Error Pool::run_iterate_checked(
    const std::shared_ptr<UA_Client> &client,
    const std::string &log_prefix
) {
    const UA_StatusCode status = UA_Client_run_iterate(client.get(), 0);
    if (status != UA_STATUSCODE_GOOD) {
        LOG(WARNING) << log_prefix
                     << "run_iterate failed: " << UA_StatusCode_name(status);
        return opc::errors::parse(status);
    }

    // Re-verify session state after run_iterate to catch any state transition
    UA_SessionState session_state;
    UA_SecureChannelState channel_state;
    UA_Client_getState(client.get(), &channel_state, &session_state, nullptr);

    if (session_state != UA_SESSIONSTATE_ACTIVATED) {
        LOG(WARNING) << log_prefix << "Session no longer activated after run_iterate";
        return xerrors::Error(
            opc::errors::NO_CONNECTION,
            "session deactivated during maintenance"
        );
    }

    // Active health probe: read server time to verify connection is truly alive.
    // This catches cases where the server died but the client's local state
    // hasn't been updated yet.
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
        return opc::errors::parse(read_status);
    }

    return xerrors::NIL;
}

}
