// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cstring>
#include <memory>
#include <new>
#include <stdexcept>
#include <string>
#include <utility>
#include <vector>

#include "absl/log/log.h"
#include "open62541/client_highlevel.h"
#include "open62541/client_subscriptions.h"

#include "driver/opcua/connection/connection.h"
#include "driver/opcua/errors/errors.h"

namespace driver::opcua::connection {
const x::errors::Error BREAKER_OPEN_ERROR(
    opcua::errors::NO_CONNECTION,
    "The Driver paused new connections to the server after too many failed attempts "
    "in a row."
);

namespace {
/// @brief identifies the configurations whose connections differ only in the client
/// certificate. The endpoint comes first for available_count.
std::string settings_key(const Config &cfg) {
    std::string key = cfg.endpoint + "|" + cfg.username + "|" + cfg.password + "|" +
                      cfg.security_mode + "|" + cfg.security_policy;
    if (!cfg.secured()) return key;
    return key + "|" + cfg.client_cert + "|" + cfg.client_private_key + "|" +
           (cfg.certificate_validation_disabled ? "unvalidated" : "validated");
}

/// @brief identifies the connections that one configuration and client certificate
/// can share. The trusted server certificate is not part of the key, so a connection
/// that trusted the first certificate serves the same device after the Driver pins
/// that certificate.
std::string pool_key(const std::string &settings, const Security &sec) {
    if (!sec.identity.has_value()) return settings;
    return settings + "|" + sec.identity->certificate.thumbprint();
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

/// @returns the UInt32 value of a read result, or 0 if the server has no such value.
uint32_t uint32_value(const UA_DataValue &result) {
    if (!result.hasValue ||
        !UA_Variant_hasScalarType(&result.value, &UA_TYPES[UA_TYPES_UINT32]))
        return 0;
    return *static_cast<const UA_UInt32 *>(result.value.data);
}

/// @returns an error if the number of results does not match the number of nodes.
x::errors::Error check_results(const size_t results, const size_t nodes) {
    if (results == nodes) return x::errors::NIL;
    return x::errors::Error(
        errors::CRITICAL,
        "the server returned " + std::to_string(results) + " results for " +
            std::to_string(nodes) + " nodes"
    );
}

/// @brief reads the value of one node in namespace 0, in a request of its own.
std::pair<types::ReadResponse, x::errors::Error>
read_value(UA_Client *client, const UA_UInt32 node) {
    if (const auto err = lost(client)) return {types::ReadResponse(), err};
    UA_ReadValueId id;
    UA_ReadValueId_init(&id);
    id.nodeId = UA_NODEID_NUMERIC(0, node);
    id.attributeId = UA_ATTRIBUTEID_VALUE;
    UA_ReadRequest req;
    UA_ReadRequest_init(&req);
    req.nodesToRead = &id;
    req.nodesToReadSize = 1;
    types::ReadResponse res(UA_Client_Service_read(client, req));
    const auto &raw = res.get();
    auto err = errors::parse(raw.responseHeader.serviceResult);
    if (!err) err = check_results(raw.resultsSize, 1);
    return {std::move(res), err};
}

/// @brief reads what the Driver needs to know about the client's server, which also
/// checks that the server answers. A server that lacks an operation limit has none.
/// Each node gets its own request, since a server can allow fewer nodes per request
/// than the Driver reads here.
std::pair<Server, x::errors::Error> read_server(UA_Client *client) {
    if (const auto status = UA_Client_run_iterate(client, 0);
        status != UA_STATUSCODE_GOOD)
        return {Server{}, errors::parse(status)};
    Server server;
    const auto [namespaces, ns_err] = read_value(
        client,
        UA_NS0ID_SERVER_NAMESPACEARRAY
    );
    if (ns_err) return {Server{}, ns_err};
    if (const auto &value = namespaces.get().results[0];
        value.hasValue &&
        UA_Variant_hasArrayType(&value.value, &UA_TYPES[UA_TYPES_STRING])) {
        const auto *uris = static_cast<const UA_String *>(value.value.data);
        for (size_t i = 0; i < value.value.arrayLength; ++i)
            server.namespaces.emplace_back(
                reinterpret_cast<const char *>(uris[i].data),
                uris[i].length
            );
    }
    const auto [read_limit, read_err] = read_value(
        client,
        UA_NS0ID_SERVER_SERVERCAPABILITIES_OPERATIONLIMITS_MAXNODESPERREAD
    );
    if (read_err) return {Server{}, read_err};
    server.max_nodes_per_read = uint32_value(read_limit.get().results[0]);
    const auto [write_limit, write_err] = read_value(
        client,
        UA_NS0ID_SERVER_SERVERCAPABILITIES_OPERATIONLIMITS_MAXNODESPERWRITE
    );
    if (write_err) return {Server{}, write_err};
    server.max_nodes_per_write = uint32_value(write_limit.get().results[0]);
    const auto [browse_limit, browse_err] = read_value(
        client,
        UA_NS0ID_SERVER_SERVERCAPABILITIES_OPERATIONLIMITS_MAXNODESPERBROWSE
    );
    if (browse_err) return {Server{}, browse_err};
    server.max_nodes_per_browse = uint32_value(browse_limit.get().results[0]);
    return {std::move(server), x::errors::NIL};
}

/// @brief moves the results of a response to one part of a request into results.
template<typename T>
void move_results(T *results, T *&part, size_t &part_size) {
    std::memcpy(results, part, part_size * sizeof(T));
    UA_free(part);
    part = nullptr;
    part_size = 0;
}

/// @brief owns a response from the client and clears it on destruction.
template<typename T>
class Response {
    T value_;
    const UA_DataType *type_;

public:
    Response(T value, const UA_DataType *type): value_(value), type_(type) {}
    ~Response() { UA_clear(&value_, type_); }
    Response(const Response &) = delete;
    Response &operator=(const Response &) = delete;
    Response(Response &&) = delete;
    Response &operator=(Response &&) = delete;

    T &get() { return value_; }
};

/// @brief the continuation points of one BrowseNext request, with the index of the
/// browse result that each one continues.
struct Pending {
    UA_BrowseNextRequest req{};
    std::vector<size_t> results;

    /// @brief takes the continuation points of the count results in raw, where raw[i]
    /// continues the browse result at indexes[i].
    Pending(
        UA_BrowseResult *raw,
        const size_t count,
        const std::vector<size_t> &indexes
    ) {
        UA_BrowseNextRequest_init(&req);
        std::vector<size_t> positions;
        for (size_t i = 0; i < count; ++i)
            if (raw[i].continuationPoint.length > 0) positions.push_back(i);
        if (positions.empty()) return;
        req.continuationPoints = static_cast<UA_ByteString *>(
            UA_Array_new(positions.size(), &UA_TYPES[UA_TYPES_BYTESTRING])
        );
        if (req.continuationPoints == nullptr) throw std::bad_alloc();
        req.continuationPointsSize = positions.size();
        for (size_t j = 0; j < positions.size(); ++j) {
            auto &point = raw[positions[j]].continuationPoint;
            req.continuationPoints[j] = point;
            UA_ByteString_init(&point);
            results.push_back(indexes[positions[j]]);
        }
    }

    ~Pending() { UA_BrowseNextRequest_clear(&req); }
    Pending(const Pending &) = delete;
    Pending &operator=(const Pending &) = delete;
    Pending(Pending &&) = delete;
    Pending &operator=(Pending &&) = delete;

    [[nodiscard]] bool empty() const { return results.empty(); }

    /// @brief frees the continuation points on the server, since each one counts
    /// against the server's limit until the session closes. Does nothing when the
    /// connection is lost, since the server then frees them itself.
    void release(UA_Client *client) {
        if (this->empty() || lost(client)) return;
        req.releaseContinuationPoints = true;
        const Response<UA_BrowseNextResponse> res(
            UA_Client_Service_browseNext(client, req),
            &UA_TYPES[UA_TYPES_BROWSENEXTRESPONSE]
        );
    }
};

/// @brief adds the children that raw lists to out, with the namespace indexes of the
/// server. A failed browse sets the error of out and drops its children.
void add_references(
    const UA_BrowseResult &raw,
    types::BrowseResult &out,
    const std::vector<std::string> &namespaces
) {
    if (auto err = errors::parse(raw.statusCode)) {
        out.err = err;
        out.references.clear();
        return;
    }
    if (out.err) return;
    for (size_t i = 0; i < raw.referencesSize; ++i) {
        const auto &ref = raw.references[i];
        if (ref.nodeId.serverIndex != 0) continue;
        UA_NodeId id = ref.nodeId.nodeId;
        if (const auto &uri = ref.nodeId.namespaceUri; uri.length > 0) {
            const auto it = std::ranges::find(
                namespaces,
                std::string(reinterpret_cast<const char *>(uri.data), uri.length)
            );
            if (it == namespaces.end()) continue;
            id.namespaceIndex = static_cast<UA_UInt16>(it - namespaces.begin());
        }
        const auto &name = ref.browseName.name;
        out.references.push_back({
            .node = types::NodeId(id),
            .browse_name = std::string(
                reinterpret_cast<const char *>(name.data),
                name.length
            ),
            .node_class = ref.nodeClass,
        });
    }
}
}

std::pair<types::ReadResponse, x::errors::Error>
Pool::Connection::read(const UA_ReadRequest &req) const {
    const size_t limit = this->server_->max_nodes_per_read;
    const size_t total = req.nodesToReadSize;
    if (limit == 0 || total <= limit) {
        if (const auto err = lost(this->get())) return {types::ReadResponse(), err};
        types::ReadResponse res(UA_Client_Service_read(this->get(), req));
        const auto &raw = res.get();
        if (auto err = errors::parse(raw.responseHeader.serviceResult))
            return {std::move(res), err};
        this->pool_->replied(this->key_);
        auto err = check_results(raw.resultsSize, total);
        return {std::move(res), err};
    }
    types::ReadResponse merged;
    auto &out = *merged.ptr();
    out.results = static_cast<UA_DataValue *>(
        UA_Array_new(total, &UA_TYPES[UA_TYPES_DATAVALUE])
    );
    if (out.results == nullptr) throw std::bad_alloc();
    out.resultsSize = total;
    for (size_t offset = 0; offset < total; offset += limit) {
        UA_ReadRequest part = req;
        part.nodesToRead = req.nodesToRead + offset;
        part.nodesToReadSize = std::min(limit, total - offset);
        auto [res, err] = this->read(part);
        if (err) return {types::ReadResponse(), err};
        auto &raw = *res.ptr();
        move_results(out.results + offset, raw.results, raw.resultsSize);
    }
    return {std::move(merged), x::errors::NIL};
}

std::pair<types::WriteResponse, x::errors::Error>
Pool::Connection::write(const UA_WriteRequest &req) const {
    const size_t limit = this->server_->max_nodes_per_write;
    const size_t total = req.nodesToWriteSize;
    if (limit == 0 || total <= limit) {
        if (const auto err = lost(this->get())) return {types::WriteResponse(), err};
        types::WriteResponse res(UA_Client_Service_write(this->get(), req));
        const auto &raw = res.get();
        if (auto err = errors::parse(raw.responseHeader.serviceResult))
            return {std::move(res), err};
        this->pool_->replied(this->key_);
        auto err = check_results(raw.resultsSize, total);
        return {std::move(res), err};
    }
    types::WriteResponse merged;
    auto &out = *merged.ptr();
    out.results = static_cast<UA_StatusCode *>(
        UA_Array_new(total, &UA_TYPES[UA_TYPES_STATUSCODE])
    );
    if (out.results == nullptr) throw std::bad_alloc();
    out.resultsSize = total;
    for (size_t offset = 0; offset < total; offset += limit) {
        UA_WriteRequest part = req;
        part.nodesToWrite = req.nodesToWrite + offset;
        part.nodesToWriteSize = std::min(limit, total - offset);
        auto [res, err] = this->write(part);
        if (err) return {types::WriteResponse(), err};
        auto &raw = *res.ptr();
        move_results(out.results + offset, raw.results, raw.resultsSize);
    }
    return {std::move(merged), x::errors::NIL};
}

std::pair<std::vector<types::BrowseResult>, x::errors::Error>
Pool::Connection::browse(const std::vector<UA_NodeId> &nodes) const {
    std::vector<types::BrowseResult> results(nodes.size());
    const size_t limit = this->server_->max_nodes_per_browse;
    const size_t step = limit == 0 ? std::max<size_t>(nodes.size(), 1) : limit;
    for (size_t offset = 0; offset < nodes.size(); offset += step)
        if (auto err = this->browse(
                nodes.data() + offset,
                std::min(step, nodes.size() - offset),
                results.data() + offset
            ))
            return {{}, err};
    return {std::move(results), x::errors::NIL};
}

x::errors::Error Pool::Connection::browse(
    const UA_NodeId *nodes,
    const size_t count,
    types::BrowseResult *results
) const {
    std::vector<UA_BrowseDescription> descriptions(count);
    for (size_t i = 0; i < count; ++i) {
        auto &d = descriptions[i];
        UA_BrowseDescription_init(&d);
        d.nodeId = nodes[i];
        d.browseDirection = UA_BROWSEDIRECTION_FORWARD;
        d.referenceTypeId = UA_NODEID_NUMERIC(0, UA_NS0ID_HIERARCHICALREFERENCES);
        d.includeSubtypes = true;
        d.resultMask = UA_BROWSERESULTMASK_BROWSENAME | UA_BROWSERESULTMASK_NODECLASS;
    }
    UA_BrowseRequest req;
    UA_BrowseRequest_init(&req);
    req.nodesToBrowse = descriptions.data();
    req.nodesToBrowseSize = count;
    if (auto err = lost(this->get())) return err;
    Response<UA_BrowseResponse> res(
        UA_Client_Service_browse(this->get(), req),
        &UA_TYPES[UA_TYPES_BROWSERESPONSE]
    );
    auto &raw = res.get();
    if (auto err = errors::parse(raw.responseHeader.serviceResult)) return err;
    this->pool_->replied(this->key_);
    const auto &namespaces = this->server_->namespaces;
    std::vector<size_t> indexes(count);
    for (size_t i = 0; i < count; ++i)
        indexes[i] = i;
    const size_t received = std::min(raw.resultsSize, count);
    auto pending = std::make_unique<Pending>(raw.results, received, indexes);
    if (auto err = check_results(raw.resultsSize, count)) {
        pending->release(this->get());
        return err;
    }
    for (size_t i = 0; i < count; ++i)
        add_references(raw.results[i], results[i], namespaces);
    while (!pending->empty()) {
        if (auto err = lost(this->get())) return err;
        Response<UA_BrowseNextResponse> next(
            UA_Client_Service_browseNext(this->get(), pending->req),
            &UA_TYPES[UA_TYPES_BROWSENEXTRESPONSE]
        );
        auto &next_raw = next.get();
        if (auto err = errors::parse(next_raw.responseHeader.serviceResult)) {
            pending->release(this->get());
            return err;
        }
        this->pool_->replied(this->key_);
        const size_t asked = pending->results.size();
        const size_t got = std::min(next_raw.resultsSize, asked);
        auto following = std::make_unique<Pending>(
            next_raw.results,
            got,
            pending->results
        );
        if (auto err = check_results(next_raw.resultsSize, asked)) {
            following->release(this->get());
            return err;
        }
        bool found = false;
        for (size_t j = 0; j < got; ++j) {
            found = found || next_raw.results[j].referencesSize > 0;
            add_references(
                next_raw.results[j],
                results[pending->results[j]],
                namespaces
            );
        }
        pending = std::move(following);
        if (!found && !pending->empty()) {
            pending->release(this->get());
            return x::errors::Error(
                errors::CRITICAL,
                "the server returned continuation points with no children"
            );
        }
    }
    return x::errors::NIL;
}

Pool::Pool(
    std::shared_ptr<certificate::Store> store,
    const std::chrono::milliseconds idle_timeout
):
    store(std::move(store)), idle_timeout(idle_timeout) {}

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
    const std::string &log_prefix
) {
    this->last_error = err;
    if (++this->consecutive_failures < BREAKER_THRESHOLD) return;
    this->cooldown_until = std::chrono::steady_clock::now() + this->cooldown;
    LOG(WARNING) << log_prefix << "pausing connections to the server for "
                 << this->cooldown.count() << " s after " << this->consecutive_failures
                 << " failed attempts in a row";
    this->cooldown = std::min(this->cooldown * 2, MAX_COOLDOWN);
}

void Pool::BreakerState::succeeded() {
    this->consecutive_failures = 0;
    this->cooldown = MIN_COOLDOWN;
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
            std::shared_ptr<const Server> server;
            {
                std::lock_guard lock(this->mutex_);
                this->breakers[key].succeeded();
                server = this->entry(key, client).server;
            }
            this->replied(key);
            return {Connection(client, this, key, std::move(server)), x::errors::NIL};
        }
        VLOG(1) << log_prefix << "Idle connection stopped answering";
    }

    std::unique_lock lock(this->mutex_);
    // References to map values stay valid as the map grows.
    auto &breaker = this->breakers[key];
    if (breaker.trusted != cfg.server_certificate) {
        breaker.trusted = cfg.server_certificate;
        breaker.consecutive_failures = 0;
        breaker.cooldown = MIN_COOLDOWN;
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
            auto server = this->entry(key, other).server;
            return {Connection(other, this, key, std::move(server)), x::errors::NIL};
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
    std::shared_ptr<const Server> server;
    if (!err) {
        auto [read, read_err] = read_server(client.get());
        err = read_err;
        server = std::make_shared<const Server>(std::move(read));
    }

    lock.lock();
    breaker.connecting = false;
    if (err) {
        breaker.failed(err, log_prefix);
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
        this->connections_[key].push_back({
            .client = client,
            .server = server,
            .in_use = true,
            .retired = replaced,
            .idle_since = {},
        });
    } else
        this->entry(key, client).server = server;
    lock.unlock();
    this->connect_cv.notify_all();
    this->replied(key);
    VLOG(1) << log_prefix << "Connected to " << cfg.endpoint;
    return {Connection(client, this, key, std::move(server)), x::errors::NIL};
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

void Pool::close_idle() {
    std::vector<std::shared_ptr<UA_Client>> closed;
    {
        std::lock_guard lock(this->mutex_);
        const auto deadline = std::chrono::steady_clock::now() - this->idle_timeout;
        for (auto it = this->connections_.begin(); it != this->connections_.end();) {
            for (auto &entry: it->second)
                if (!entry.in_use && (entry.retired || entry.idle_since <= deadline))
                    closed.push_back(std::move(entry.client));
            std::erase_if(it->second, [](const Entry &e) { return !e.client; });
            if (it->second.empty())
                it = this->connections_.erase(it);
            else
                ++it;
        }
    }
    if (!closed.empty())
        VLOG(1) << "[opc.pool] Closing " << closed.size() << " idle connections";
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
            return;
        }
}

Pool::Entry &
Pool::entry(const std::string &key, const std::shared_ptr<UA_Client> &client) {
    for (auto &e: this->connections_.at(key))
        if (e.client == client) return e;
    throw std::logic_error("no pool entry for an acquired connection");
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
        if (key.starts_with(endpoint))
            for (const auto &entry: entries)
                if (!entry.in_use && entry.client) count++;
    return count;
}
namespace {
/// @brief how often the server publishes the events of a subscription.
constexpr double EVENT_PUBLISHING_INTERVAL_MS = 250;
/// @brief the publishing intervals with no events after which the server sends a
/// keep-alive.
constexpr UA_UInt32 EVENT_KEEP_ALIVE_COUNT = 10;
/// @brief the publishing intervals with no publish request after which the server ends
/// the subscription: three keep-alives.
constexpr UA_UInt32 EVENT_LIFETIME_COUNT = 3 * EVENT_KEEP_ALIVE_COUNT;
/// @brief the most events that the server holds for one notifier between publishes.
constexpr UA_UInt32 EVENT_QUEUE_SIZE = 1000;

void on_event(
    UA_Client *,
    UA_UInt32,
    void *subscription,
    UA_UInt32,
    void *,
    const UA_KeyValueMap fields
) {
    static_cast<Subscription *>(subscription)->handle(fields);
}

void on_status_change(
    UA_Client *,
    UA_UInt32,
    void *subscription,
    UA_StatusChangeNotification *notification
) {
    if (notification != nullptr && UA_StatusCode_isBad(notification->status))
        static_cast<Subscription *>(subscription)->status = notification->status;
}
}

std::pair<types::CallResponse, x::errors::Error>
Pool::Connection::call(const UA_CallMethodRequest &method) const {
    if (const auto err = lost(this->get())) return {types::CallResponse(), err};
    UA_CallRequest req;
    UA_CallRequest_init(&req);
    req.methodsToCall = const_cast<UA_CallMethodRequest *>(&method);
    req.methodsToCallSize = 1;
    types::CallResponse res(UA_Client_Service_call(this->get(), req));
    const auto &raw = res.get();
    if (auto err = errors::parse(raw.responseHeader.serviceResult))
        return {std::move(res), err};
    this->pool_->replied(this->key_);
    auto err = check_results(raw.resultsSize, 1);
    return {std::move(res), err};
}

std::pair<std::vector<x::errors::Error>, x::errors::Error> Pool::Connection::subscribe(
    const std::vector<UA_NodeId> &notifiers,
    const UA_EventFilter &filter,
    EventHandler handle
) {
    this->unsubscribe();
    if (const auto err = lost(this->get())) return {{}, err};
    auto sub = std::make_unique<Subscription>();
    sub->handle = std::move(handle);
    auto req = UA_CreateSubscriptionRequest_default();
    req.requestedPublishingInterval = EVENT_PUBLISHING_INTERVAL_MS;
    req.requestedMaxKeepAliveCount = EVENT_KEEP_ALIVE_COUNT;
    req.requestedLifetimeCount = EVENT_LIFETIME_COUNT;
    const auto res = UA_Client_Subscriptions_create(
        this->get(),
        req,
        sub.get(),
        on_status_change,
        nullptr
    );
    if (auto err = errors::parse(res.responseHeader.serviceResult)) return {{}, err};
    sub->id = res.subscriptionId;
    this->subscription_ = std::move(sub);
    std::vector<x::errors::Error> results;
    results.reserve(notifiers.size());
    for (const auto &notifier: notifiers) {
        if (const auto err = lost(this->get())) {
            this->unsubscribe();
            return {{}, err};
        }
        UA_MonitoredItemCreateRequest item;
        UA_MonitoredItemCreateRequest_init(&item);
        item.itemToMonitor.nodeId = notifier;
        item.itemToMonitor.attributeId = UA_ATTRIBUTEID_EVENTNOTIFIER;
        item.monitoringMode = UA_MONITORINGMODE_REPORTING;
        item.requestedParameters.queueSize = EVENT_QUEUE_SIZE;
        item.requestedParameters.discardOldest = true;
        item.requestedParameters.filter.encoding = UA_EXTENSIONOBJECT_DECODED;
        item.requestedParameters.filter.content.decoded
            .type = &UA_TYPES[UA_TYPES_EVENTFILTER];
        item.requestedParameters.filter.content.decoded
            .data = const_cast<UA_EventFilter *>(&filter);
        auto created = UA_Client_MonitoredItems_createEvent(
            this->get(),
            this->subscription_->id,
            UA_TIMESTAMPSTORETURN_BOTH,
            item,
            nullptr,
            on_event,
            nullptr
        );
        results.push_back(errors::parse(created.statusCode));
        UA_MonitoredItemCreateResult_clear(&created);
    }
    return {std::move(results), x::errors::NIL};
}

UA_UInt32 Pool::Connection::subscription_id() const {
    return this->subscription_ == nullptr ? 0 : this->subscription_->id;
}

x::errors::Error Pool::Connection::iterate(const x::telem::TimeSpan timeout) const {
    if (const auto err = lost(this->get())) return err;
    const auto status = UA_Client_run_iterate(
        this->get(),
        static_cast<UA_UInt32>(timeout.milliseconds())
    );
    if (const auto err = lost(this->get())) return err;
    if (status != UA_STATUSCODE_GOOD) return errors::parse(status);
    if (this->subscription_ != nullptr &&
        UA_StatusCode_isBad(this->subscription_->status))
        return errors::parse(this->subscription_->status);
    return x::errors::NIL;
}

void Pool::Connection::close() {
    this->unsubscribe();
    if (this->pool_ && this->client_) this->pool_->release(this->key_, this->client_);
}

void Pool::Connection::unsubscribe() {
    if (this->subscription_ == nullptr) return;
    // The handle stays alive until the client lets go of it, since a delete processes
    // the notifications that arrive before its response.
    const auto sub = std::move(this->subscription_);
    if (this->client_ == nullptr) return;
    if (!lost(this->get())) UA_Client_Subscriptions_deleteSingle(this->get(), sub->id);
    void *context = nullptr;
    if (UA_Client_Subscriptions_getContext(this->get(), sub->id, &context) !=
        UA_STATUSCODE_GOOD)
        return;
    LOG(WARNING) << "[opc.connection] closing the Session, since the server did not "
                    "delete subscription "
                 << sub->id;
    UA_Client_disconnect(this->get());
}
}
