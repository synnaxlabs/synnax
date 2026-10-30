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
#include <functional>
#include <memory>
#include <new>
#include <optional>
#include <stdexcept>
#include <string>
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
                      field(cfg.security_mode) + field(cfg.security_policy) +
                      field(std::to_string(cfg.secure_channel_lifetime_ms)) +
                      field(std::to_string(cfg.session_timeout_ms)) +
                      field(std::to_string(cfg.client_timeout_ms));
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

/// @brief moves the results of a response to one part of a request into results.
template<typename T>
void move_results(T *results, T *&part, size_t &part_size) {
    std::memcpy(results, part, part_size * sizeof(T));
    UA_free(part);
    part = nullptr;
    part_size = 0;
}

/// @brief the continuation points of one BrowseNext request, with the index of the
/// browse result that each one continues.
struct Pending {
    UA_BrowseNextRequest req{};
    std::vector<size_t> results;

    /// @brief takes the continuation points of the count results in raw, where raw[i]
    /// continues the browse result at indexes[i]. Takes only the points whose index
    /// keep accepts, when keep is set.
    Pending(
        UA_BrowseResult *raw,
        const size_t count,
        const std::vector<size_t> &indexes,
        const std::function<bool(size_t)> &keep = nullptr
    ) {
        UA_BrowseNextRequest_init(&req);
        std::vector<size_t> positions;
        for (size_t i = 0; i < count; ++i)
            if (raw[i].continuationPoint.length > 0 &&
                (keep == nullptr || keep(indexes[i])))
                positions.push_back(i);
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
    Pending(Pending &&other) noexcept:
        req(other.req), results(std::move(other.results)) {
        UA_BrowseNextRequest_init(&other.req);
    }
    Pending &operator=(Pending &&other) noexcept {
        if (this != &other) {
            UA_BrowseNextRequest_clear(&req);
            req = other.req;
            results = std::move(other.results);
            UA_BrowseNextRequest_init(&other.req);
        }
        return *this;
    }

    [[nodiscard]] bool empty() const { return results.empty(); }

    /// @brief frees the continuation points on the server, since each one counts
    /// against the server's limit until the session closes. Does nothing when the
    /// connection is lost, since the server then frees them itself.
    void release(UA_Client *client) {
        if (this->empty() || lost(client)) return;
        req.releaseContinuationPoints = true;
        const types::BrowseNextResponse res(UA_Client_Service_browseNext(client, req));
    }
};

/// @brief adds the children that raw lists to out, with the namespace indexes of the
/// server, and takes one from room for each. Sets out.truncated when raw lists a child
/// after room runs out. A failed browse sets the error of out and drops its children.
void add_references(
    const UA_BrowseResult &raw,
    types::BrowseResult &out,
    size_t &room,
    const std::vector<std::string> &namespaces
) {
    if (auto err = errors::parse(raw.statusCode)) {
        out.err = err;
        out.references.clear();
        return;
    }
    if (out.err) return;
    for (size_t i = 0; i < raw.referencesSize; ++i) {
        if (room == 0) {
            out.truncated = true;
            return;
        }
        --room;
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

/// @brief adds the children that the count results in raw list to results, where raw[i]
/// continues the browse result at indexes[i]. Releases the continuation points of the
/// results that room cut short.
/// @returns the continuation points to follow.
Pending add_page(
    UA_Client *client,
    UA_BrowseResult *raw,
    const size_t count,
    const std::vector<size_t> &indexes,
    types::BrowseResult *results,
    size_t &room,
    const std::vector<std::string> &namespaces
) {
    for (size_t i = 0; i < count; ++i)
        add_references(raw[i], results[indexes[i]], room, namespaces);
    Pending(raw, count, indexes, [results](const size_t i) {
        return results[i].truncated;
    }).release(client);
    return {raw, count, indexes};
}

/// @brief reads the value of each node in namespace 0, in requests of at most limit
/// nodes, or in one request when limit is 0.
/// @returns one result for each node, in order.
std::pair<types::ReadResponse, x::errors::Error> read_values(
    UA_Client *client,
    const std::vector<UA_UInt32> &nodes,
    const size_t limit
) {
    std::vector<UA_ReadValueId> ids(nodes.size());
    for (size_t i = 0; i < nodes.size(); ++i) {
        UA_ReadValueId_init(&ids[i]);
        ids[i].nodeId = UA_NODEID_NUMERIC(0, nodes[i]);
        ids[i].attributeId = UA_ATTRIBUTEID_VALUE;
    }
    types::ReadResponse merged;
    auto &out = *merged.ptr();
    out.results = static_cast<UA_DataValue *>(
        UA_Array_new(ids.size(), &UA_TYPES[UA_TYPES_DATAVALUE])
    );
    if (out.results == nullptr) throw std::bad_alloc();
    out.resultsSize = ids.size();
    const size_t step = limit == 0 ? ids.size() : limit;
    for (size_t offset = 0; offset < ids.size(); offset += step) {
        if (const auto err = lost(client)) return {types::ReadResponse(), err};
        UA_ReadRequest req;
        UA_ReadRequest_init(&req);
        req.nodesToRead = ids.data() + offset;
        req.nodesToReadSize = std::min(step, ids.size() - offset);
        types::ReadResponse res(UA_Client_Service_read(client, req));
        auto &raw = *res.ptr();
        auto err = errors::parse(raw.responseHeader.serviceResult);
        if (!err) err = check_results(raw.resultsSize, req.nodesToReadSize);
        if (err) return {types::ReadResponse(), err};
        move_results(out.results + offset, raw.results, raw.resultsSize);
    }
    return {std::move(merged), x::errors::NIL};
}

/// @returns the namespace URIs in value, a read of the NamespaceArray node, or
/// errors::CRITICAL if value holds no array of strings. Every server has the node.
std::pair<std::vector<std::string>, x::errors::Error>
parse_namespaces(const UA_DataValue &value) {
    if (UA_StatusCode_isBad(value.status))
        return {
            {},
            x::errors::Error(
                errors::CRITICAL,
                std::string("the server failed to read its namespace array: ") +
                    UA_StatusCode_name(value.status)
            )
        };
    if (!value.hasValue ||
        !UA_Variant_hasArrayType(&value.value, &UA_TYPES[UA_TYPES_STRING]))
        return {
            {},
            x::errors::Error(errors::CRITICAL, "the server has no namespace array")
        };
    std::vector<std::string> namespaces;
    namespaces.reserve(value.value.arrayLength);
    const auto *uris = static_cast<const UA_String *>(value.value.data);
    for (size_t i = 0; i < value.value.arrayLength; ++i)
        namespaces.emplace_back(
            reinterpret_cast<const char *>(uris[i].data),
            uris[i].length
        );
    return {std::move(namespaces), x::errors::NIL};
}

/// @brief reads what the Driver needs to know about the client's server, which also
/// checks that the server answers. A server that lacks an operation limit has none.
/// @returns errors::CRITICAL if the server has no namespace array.
std::pair<Server, x::errors::Error> read_server(UA_Client *client) {
    if (const auto status = UA_Client_run_iterate(client, 0);
        status != UA_STATUSCODE_GOOD)
        return {Server{}, errors::parse(status)};
    Server server;
    // The read limit comes alone, since the other reads must keep to it.
    const auto [limit, limit_err] = read_values(
        client,
        {UA_NS0ID_SERVER_SERVERCAPABILITIES_OPERATIONLIMITS_MAXNODESPERREAD},
        0
    );
    if (limit_err) return {Server{}, limit_err};
    server.max_nodes_per_read = uint32_value(limit.get().results[0]);
    const auto [facts, err] = read_values(
        client,
        {
            UA_NS0ID_SERVER_NAMESPACEARRAY,
            UA_NS0ID_SERVER_SERVERCAPABILITIES_OPERATIONLIMITS_MAXNODESPERWRITE,
            UA_NS0ID_SERVER_SERVERCAPABILITIES_OPERATIONLIMITS_MAXNODESPERBROWSE,
        },
        server.max_nodes_per_read
    );
    if (err) return {Server{}, err};
    const auto *results = facts.get().results;
    auto [namespaces, ns_err] = parse_namespaces(results[0]);
    if (ns_err) return {Server{}, ns_err};
    server.namespaces = std::move(namespaces);
    server.max_nodes_per_write = uint32_value(results[1]);
    server.max_nodes_per_browse = uint32_value(results[2]);
    return {std::move(server), x::errors::NIL};
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

std::pair<bool, x::errors::Error>
Pool::Connection::update_namespaces(const UA_DataValue &value) {
    auto [namespaces, err] = parse_namespaces(value);
    if (err) return {false, err};
    if (namespaces == this->server_->namespaces) return {false, x::errors::NIL};
    auto server = std::make_shared<Server>(*this->server_);
    server->namespaces = std::move(namespaces);
    this->server_ = std::move(server);
    std::lock_guard lock(this->pool_->mutex_);
    this->pool_->entry(this->key_, this->client_).server = this->server_;
    return {true, x::errors::NIL};
}

std::pair<types::ReadResponse, x::errors::Error>
Pool::Connection::read(const UA_ReadRequest &req) const {
    const size_t limit = this->server_->max_nodes_per_read;
    const size_t total = req.nodesToReadSize;
    if (limit == 0 || total <= limit) {
        if (const auto err = lost(this->get())) return {types::ReadResponse(), err};
        types::ReadResponse res(UA_Client_Service_read(this->get(), req));
        const auto &raw = res.get();
        if (auto err = this->check(raw.responseHeader.serviceResult))
            return {std::move(res), err};
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
        if (auto err = this->check(raw.responseHeader.serviceResult))
            return {std::move(res), err};
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

std::pair<std::vector<types::BrowseResult>, x::errors::Error> Pool::Connection::browse(
    const std::vector<UA_NodeId> &nodes,
    const std::chrono::steady_clock::time_point deadline
) const {
    std::vector<types::BrowseResult> results(nodes.size());
    const size_t limit = this->server_->max_nodes_per_browse;
    const size_t step = limit == 0 ? std::max<size_t>(nodes.size(), 1) : limit;
    size_t room = MAX_CHILDREN;
    for (size_t offset = 0; offset < nodes.size(); offset += step) {
        if (offset > 0 && std::chrono::steady_clock::now() >= deadline) {
            for (size_t i = offset; i < nodes.size(); ++i)
                results[i].truncated = true;
            break;
        }
        if (auto err = this->browse(
                nodes.data() + offset,
                std::min(step, nodes.size() - offset),
                results.data() + offset,
                room,
                deadline
            ))
            return {std::vector<types::BrowseResult>(), err};
    }
    return {std::move(results), x::errors::NIL};
}

x::errors::Error Pool::Connection::browse(
    const UA_NodeId *nodes,
    const size_t count,
    types::BrowseResult *results,
    size_t &room,
    const std::chrono::steady_clock::time_point deadline
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
    types::BrowseResponse res(UA_Client_Service_browse(this->get(), req));
    auto &raw = *res.ptr();
    if (auto err = this->check(raw.responseHeader.serviceResult)) return err;
    const auto &namespaces = this->server_->namespaces;
    std::vector<size_t> indexes(count);
    for (size_t i = 0; i < count; ++i)
        indexes[i] = i;
    if (auto err = check_results(raw.resultsSize, count)) {
        const size_t received = std::min(raw.resultsSize, count);
        Pending(raw.results, received, indexes).release(this->get());
        return err;
    }
    auto pending = add_page(
        this->get(),
        raw.results,
        count,
        indexes,
        results,
        room,
        namespaces
    );
    while (!pending.empty()) {
        if (std::chrono::steady_clock::now() >= deadline) {
            for (const auto i: pending.results)
                results[i].truncated = true;
            pending.release(this->get());
            return x::errors::NIL;
        }
        if (auto err = lost(this->get())) return err;
        types::BrowseNextResponse next(
            UA_Client_Service_browseNext(this->get(), pending.req)
        );
        auto &next_raw = *next.ptr();
        if (auto err = this->check(next_raw.responseHeader.serviceResult)) {
            pending.release(this->get());
            return err;
        }
        const size_t asked = pending.results.size();
        if (auto err = check_results(next_raw.resultsSize, asked)) {
            const size_t got = std::min(next_raw.resultsSize, asked);
            Pending(next_raw.results, got, pending.results).release(this->get());
            return err;
        }
        bool found = false;
        for (size_t j = 0; j < asked; ++j)
            found = found || next_raw.results[j].referencesSize > 0;
        pending = add_page(
            this->get(),
            next_raw.results,
            asked,
            pending.results,
            results,
            room,
            namespaces
        );
        if (!found && !pending.empty()) {
            pending.release(this->get());
            return x::errors::Error(
                errors::CRITICAL,
                "the server returned continuation points with no children"
            );
        }
    }
    return x::errors::NIL;
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
    // A kept connection must read as lost so the next attempt connects again and reads
    // the server's facts again, and a server that stopped answering gets no
    // CloseSession to wait on.
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

std::pair<Pool::Connection, x::errors::Error> Pool::refresh(
    const Connection &conn,
    const Config &cfg,
    const std::string &log_prefix
) {
    auto [sec, sec_err] = resolve_security(cfg, *this->store);
    if (sec_err) return {Connection(nullptr, nullptr, ""), sec_err};
    if (pool_key(settings_key(cfg), sec) == conn.key_)
        return {Connection(nullptr, nullptr, ""), x::errors::NIL};
    auto replacement = cfg;
    if (const auto trusted = trusted_certificate(conn.get()))
        replacement.server_certificate = trusted;
    LOG(INFO) << log_prefix << "Connecting again with the new client certificate";
    return this->acquire(replacement, log_prefix);
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
        if (key.starts_with(field(endpoint)))
            for (const auto &entry: entries)
                if (!entry.in_use && entry.client) count++;
    return count;
}
}
