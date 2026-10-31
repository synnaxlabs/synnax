// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <optional>
#include <utility>
#include <vector>

#include "absl/log/log.h"
#include "open62541/client_highlevel.h"

#include "driver/opcua/connection/connection.h"
#include "driver/opcua/errors/errors.h"

namespace driver::opcua::connection {
const x::errors::Error BREAKER_OPEN_ERROR(
    opcua::errors::NO_CONNECTION,
    "The Driver paused new connections to the server after too many failed attempts "
    "in a row."
);

namespace {
/// @brief encodes one field of a key with its length first, so different field values
/// never produce the same key.
std::string field(const std::string &value) {
    return std::to_string(value.size()) + ":" + value;
}

/// @brief identifies the configurations whose connections differ only in the client
/// certificate. The endpoint comes first for available_count.
std::string settings_key(const Config &cfg) {
    std::string key = field(cfg.endpoint) + field(cfg.username) + field(cfg.password) +
                      field(cfg.security_mode) + field(cfg.security_policy);
    if (!cfg.secured()) return key;
    return key + field(cfg.client_cert) + field(cfg.client_private_key) +
           field(cfg.certificate_validation_disabled ? "unvalidated" : "validated");
}

/// @brief identifies the connections that one configuration and client certificate
/// can share. The trusted server certificate is not part of the key, so a connection
/// that trusted the first certificate serves the same device after the Driver pins
/// that certificate.
std::string pool_key(const std::string &settings, const Security &sec) {
    if (!sec.identity.has_value()) return settings;
    return settings + field(sec.identity->certificate.thumbprint());
}

/// @returns true if the client trusts the server certificate that the configuration
/// trusts, or the configuration trusts any certificate.
bool trusts(UA_Client *client, const Config &cfg) {
    if (!cfg.secured() || cfg.certificate_validation_disabled ||
        !cfg.server_certificate.has_value())
        return true;
    return trusted_certificate(client) == cfg.server_certificate;
}

/// @brief handles the client's pending network events, then reads the server's clock to
/// check that the server still answers.
x::errors::Error probe(UA_Client *client) {
    if (const auto status = UA_Client_run_iterate(client, 0);
        status != UA_STATUSCODE_GOOD)
        return errors::parse(status);
    if (const auto err = lost(client)) return err;
    UA_Variant value;
    UA_Variant_init(&value);
    const auto status = UA_Client_readValueAttribute(
        client,
        UA_NODEID_NUMERIC(0, UA_NS0ID_SERVER_SERVERSTATUS_CURRENTTIME),
        &value
    );
    UA_Variant_clear(&value);
    return errors::parse(status);
}
}

x::errors::Error Pool::Connection::check(const UA_StatusCode result) const {
    auto err = errors::parse(result);
    if (err.matches(errors::UNREACHABLE))
        UA_Client_disconnectSecureChannel(this->get());
    else
        this->pool_->replied(this->key_);
    return err;
}

std::pair<types::ReadResponse, x::errors::Error>
Pool::Connection::read(const UA_ReadRequest &req) const {
    if (const auto err = lost(this->get())) return {types::ReadResponse(), err};
    types::ReadResponse res(UA_Client_Service_read(this->get(), req));
    auto err = this->check(res.get().responseHeader.serviceResult);
    return {std::move(res), err};
}

std::pair<types::WriteResponse, x::errors::Error>
Pool::Connection::write(const UA_WriteRequest &req) const {
    if (const auto err = lost(this->get())) return {types::WriteResponse(), err};
    types::WriteResponse res(UA_Client_Service_write(this->get(), req));
    auto err = this->check(res.get().responseHeader.serviceResult);
    return {std::move(res), err};
}

Pool::Pool(std::shared_ptr<certificate::Store> store): Pool(std::move(store), {}) {}

Pool::Pool(std::shared_ptr<certificate::Store> store, const Timing timing):
    store(std::move(store)), timing(timing), evictor([this] { this->evict(); }) {}

Pool::~Pool() {
    {
        std::lock_guard lock(this->mutex_);
        this->closing = true;
    }
    this->idle_cv.notify_all();
    this->evictor.join();
}

x::errors::Error Pool::BreakerState::open_error() const {
    return this->last_error.matches(errors::SECURITY) ? this->last_error
                                                      : BREAKER_OPEN_ERROR;
}

bool Pool::BreakerState::open() const {
    return this->consecutive_failures >= BREAKER_THRESHOLD &&
           std::chrono::steady_clock::now() < this->cooldown_until;
}

void Pool::BreakerState::failed(
    const x::errors::Error &err,
    const Timing &timing,
    const std::string &log_prefix
) {
    this->last_error = err;
    if (++this->consecutive_failures < BREAKER_THRESHOLD) return;
    const auto pause = std::max(this->cooldown, timing.min_cooldown);
    this->cooldown_until = std::chrono::steady_clock::now() + pause;
    LOG(WARNING) << log_prefix << "pausing connections to the server for "
                 << pause.count() << " ms after " << this->consecutive_failures
                 << " failed attempts in a row";
    this->cooldown = std::min(pause * 2, timing.max_cooldown);
}

void Pool::BreakerState::succeeded() {
    this->consecutive_failures = 0;
    this->cooldown = {};
    this->last_error = x::errors::NIL;
}

std::pair<Pool::Connection, x::errors::Error>
Pool::acquire(const Config &cfg, const std::string &log_prefix) {
    auto [sec, sec_err] = resolve_security(cfg, *this->store);
    if (sec_err) return {Connection(nullptr, nullptr, ""), sec_err};
    sec.first_certificate_trusted = true;
    const std::string settings = settings_key(cfg);
    const std::string key = pool_key(settings, sec);

    std::shared_ptr<UA_Client> client;
    std::vector<std::shared_ptr<UA_Client>> retired;
    {
        std::lock_guard lock(this->mutex_);
        retired = this->retire(settings, key);
        client = this->take(key, cfg);
    }
    // Some servers allow few sessions, so retired ones close before a new one opens.
    if (!retired.empty()) {
        VLOG(1) << log_prefix
                << "Closing connections with a replaced client certificate";
        retired.clear();
    }

    // The probe runs outside the lock, since it waits on the server.
    if (client != nullptr && !lost(client.get())) {
        if (const auto err = probe(client.get()); !err) {
            {
                std::lock_guard lock(this->mutex_);
                this->breakers[key].succeeded();
            }
            this->replied(key);
            return {Connection(client, this, key), x::errors::NIL};
        }
        VLOG(1) << log_prefix << "Idle connection stopped answering";
    }

    std::unique_lock lock(this->mutex_);
    // References to map values stay valid as the map grows.
    auto &breaker = this->breakers[key];
    if (breaker.trusted != cfg.server_certificate) {
        breaker.trusted = cfg.server_certificate;
        breaker.consecutive_failures = 0;
        breaker.cooldown = {};
    }
    while (true) {
        if (breaker.open()) {
            if (client != nullptr) this->put_back(key, client);
            VLOG(1) << log_prefix << "Connections to the server are paused";
            return {Connection(nullptr, nullptr, ""), breaker.open_error()};
        }
        if (!breaker.connecting) break;
        VLOG(1) << log_prefix << "Waiting for another connection attempt";
        this->connect_cv.wait(lock, [&breaker] { return !breaker.connecting; });
        auto other = this->take(key, cfg);
        if (other != nullptr && !lost(other.get())) {
            if (client != nullptr) this->put_back(key, client);
            return {Connection(other, this, key), x::errors::NIL};
        }
        if (other == nullptr) continue;
        if (client == nullptr)
            client = std::move(other);
        else
            this->put_back(key, other);
    }
    breaker.connecting = true;
    lock.unlock();

    const bool created = client == nullptr;
    x::errors::Error err;
    if (created) {
        auto [c, connect_err] = connect(cfg, sec, log_prefix);
        client = std::move(c);
        err = connect_err;
    } else {
        VLOG(1) << log_prefix << "Connecting a lost connection again";
        err = reconnect(client, cfg.endpoint);
    }
    if (!err) err = probe(client.get());
    // A kept connection must read as lost so the next attempt connects again, and a
    // server that stopped answering gets no CloseSession to wait on.
    if (err && (!created || err.matches(errors::UNREACHABLE)))
        UA_Client_disconnectSecureChannel(client.get());

    lock.lock();
    breaker.connecting = false;
    if (err) {
        breaker.failed(err, this->timing, log_prefix);
        // A lost connection keeps its entry, so its Session is activated again on the
        // next attempt instead of being left on the server.
        if (!created) this->put_back(key, client);
        lock.unlock();
        this->connect_cv.notify_all();
        return {Connection(nullptr, nullptr, ""), err};
    }
    breaker.succeeded();
    if (created) {
        // Another caller can replace the certificate while this one connects.
        const bool replaced = this->current_keys[settings] != key;
        this->connections_[key].push_back(
            {.client = client, .in_use = true, .retired = replaced, .idle_since = {}}
        );
    }
    lock.unlock();
    this->connect_cv.notify_all();
    this->replied(key);
    VLOG(1) << log_prefix << "Connected to " << cfg.endpoint;
    return {Connection(client, this, key), x::errors::NIL};
}

std::pair<bool, x::errors::Error>
Pool::refresh(Connection &conn, const Config &cfg, const std::string &log_prefix) {
    auto [sec, sec_err] = resolve_security(cfg, *this->store);
    if (sec_err) return {false, sec_err};
    if (pool_key(settings_key(cfg), sec) == conn.key_) return {false, x::errors::NIL};
    auto replacement = cfg;
    if (const auto trusted = trusted_certificate(conn.get()))
        replacement.server_certificate = trusted;
    LOG(INFO) << log_prefix << "Connecting again with the new client certificate";
    auto [next, err] = this->acquire(replacement, log_prefix);
    if (err) return {false, err};
    conn = std::move(next);
    return {true, x::errors::NIL};
}

void Pool::evict() {
    std::unique_lock lock(this->mutex_);
    while (!this->closing) {
        const auto now = std::chrono::steady_clock::now();
        std::optional<std::chrono::steady_clock::time_point> next;
        std::vector<Entry> closed;
        for (auto it = this->connections_.begin(); it != this->connections_.end();) {
            std::vector<Entry> kept;
            for (auto &entry: it->second) {
                const auto expiry = entry.idle_since + this->timing.idle_timeout;
                if (!entry.in_use && (entry.retired || expiry <= now)) {
                    closed.push_back(std::move(entry));
                    continue;
                }
                if (!entry.in_use && (!next.has_value() || expiry < *next))
                    next = expiry;
                kept.push_back(std::move(entry));
            }
            it->second = std::move(kept);
            if (it->second.empty())
                it = this->connections_.erase(it);
            else
                ++it;
        }
        if (!closed.empty()) {
            lock.unlock();
            VLOG(1) << "[opc.pool] Closing " << closed.size() << " idle connections";
            closed.clear();
            lock.lock();
            continue;
        }
        if (next.has_value())
            this->idle_cv.wait_until(lock, *next);
        else
            this->idle_cv.wait(lock);
    }
}

x::telem::TimeStamp Pool::last_reply(const Config &cfg) const {
    auto [sec, sec_err] = resolve_security(cfg, *this->store);
    if (sec_err) return {};
    const auto key = pool_key(settings_key(cfg), sec);
    std::lock_guard lock(this->mutex_);
    const auto it = this->replies.find(key);
    if (it == this->replies.end()) return {};
    return it->second;
}

void Pool::replied(const std::string &key) {
    const auto now = x::telem::TimeStamp::now();
    std::lock_guard lock(this->mutex_);
    this->replies[key] = now;
}

std::shared_ptr<UA_Client> Pool::take(const std::string &key, const Config &cfg) {
    const auto it = this->connections_.find(key);
    if (it == this->connections_.end()) return nullptr;
    Entry *dropped = nullptr;
    for (auto &entry: it->second) {
        if (entry.in_use || entry.retired || !trusts(entry.client.get(), cfg)) continue;
        if (!lost(entry.client.get())) {
            entry.in_use = true;
            return entry.client;
        }
        if (dropped == nullptr) dropped = &entry;
    }
    if (dropped == nullptr) return nullptr;
    dropped->in_use = true;
    return dropped->client;
}

void Pool::put_back(const std::string &key, const std::shared_ptr<UA_Client> &client) {
    const auto it = this->connections_.find(key);
    if (it == this->connections_.end()) return;
    for (auto &entry: it->second)
        if (entry.client == client) {
            entry.in_use = false;
            entry.idle_since = std::chrono::steady_clock::now();
            this->idle_cv.notify_all();
            return;
        }
}

void Pool::release(const std::string &key, const std::shared_ptr<UA_Client> &client) {
    std::lock_guard lock(this->mutex_);
    const auto it = this->connections_.find(key);
    if (it == this->connections_.end()) return;
    // The caller still holds the client, so a removed one closes outside the lock.
    std::erase_if(it->second, [&client](const Entry &e) {
        return e.client == client && e.retired;
    });
    this->put_back(key, client);
    if (it->second.empty()) this->connections_.erase(it);
}

std::vector<std::shared_ptr<UA_Client>>
Pool::retire(const std::string &settings, const std::string &key) {
    auto &current = this->current_keys[settings];
    if (current == key) return {};
    const auto replaced = std::exchange(current, key);
    std::vector<std::shared_ptr<UA_Client>> retired;
    const auto it = connections_.find(replaced);
    if (it == connections_.end()) return retired;
    for (auto &entry: it->second)
        if (entry.in_use)
            entry.retired = true;
        else
            retired.push_back(std::move(entry.client));
    std::erase_if(it->second, [](const Entry &e) { return !e.client; });
    if (it->second.empty()) connections_.erase(it);
    return retired;
}

size_t Pool::size() const {
    std::lock_guard lock(this->mutex_);
    size_t total = 0;
    for (const auto &[_, entries]: connections_)
        total += entries.size();
    return total;
}

size_t Pool::available_count(const std::string &endpoint) const {
    std::lock_guard lock(this->mutex_);
    size_t count = 0;
    for (const auto &[key, entries]: connections_)
        if (key.starts_with(field(endpoint)))
            for (const auto &entry: entries)
                if (!entry.in_use && entry.client) count++;
    return count;
}
}
