// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <array>
#include <chrono>
#include <memory>
#include <optional>
#include <ranges>
#include <set>
#include <span>
#include <string>
#include <tuple>
#include <utility>
#include <vector>

#include "absl/log/log.h"
#include "open62541/client.h"
#include "open62541/client_highlevel.h"
#include "open62541/types.h"

#include "x/cpp/json/json.h"

#include "driver/opcua/device/device.h"
#include "driver/opcua/errors/errors.h"
#include "driver/opcua/method/method.h"
#include "driver/opcua/scan_task.h"
#include "driver/opcua/structure/structure.h"
#include "driver/opcua/telem/telem.h"
#include "driver/opcua/types/types.h"

namespace driver::opcua {
namespace {
/// @returns the status message for a failed connection, or the fallback for a failure
/// that is not about certificates.
std::string failure_message(
    const x::errors::Error &err,
    const connection::Config &cfg,
    const std::string &fallback
) {
    if (err.matches(errors::SERVER_CERTIFICATE_UNTRUSTED))
        return "Server certificate not trusted";
    if (err.matches(errors::SERVER_CERTIFICATE_TIME_INVALID))
        return "Server certificate expired";
    if (err.matches(errors::CLIENT_CERTIFICATE_REJECTED))
        return cfg.uses_files() ? "Server rejected the client certificate"
                                : "Server rejected the Driver certificate";
    if (err.matches(errors::CLIENT_CERTIFICATE_INVALID))
        return "Failed to read client certificate";
    if (err.matches(errors::DRIVER_CERTIFICATE_FAILED))
        return "Failed to load the Driver certificate";
    return fallback;
}

/// @returns an error status for the reply to cmd, which each handler fills in.
synnax::task::Status
command_status(const synnax::task::Task &task, const synnax::task::Command &cmd) {
    return {
        .key = synnax::task::status_key(task),
        .name = task.name,
        .variant = synnax::status::VARIANT_ERROR,
        .details = synnax::task::StatusDetails{
            .task = task.key,
            .running = true,
            .cmd = cmd.key,
            .config_hash = task.config_hash,
        }
    };
}
}

Scanner::Scanner(
    std::shared_ptr<task::Context> ctx,
    synnax::task::Task task,
    std::shared_ptr<connection::Pool> conn_pool,
    std::shared_ptr<certificate::Store> store
):
    ctx(std::move(ctx)),
    task(std::move(task)),
    conn_pool(std::move(conn_pool)),
    store(std::move(store)) {}

common::ScannerConfig Scanner::config() const {
    return common::ScannerConfig{
        .make = INTEGRATION_NAME,
        .log_prefix = SCAN_LOG_PREFIX
    };
}

std::pair<std::vector<synnax::device::Device>, x::errors::Error>
Scanner::scan(const common::ScannerContext &scan_ctx) {
    std::vector<synnax::device::Device> devices;
    if (scan_ctx.devices == nullptr) return {devices, x::errors::NIL};
    for (const auto &dev: *scan_ctx.devices | std::views::values)
        devices.push_back(dev);
    const auto renew_err = this->renew_certificate(devices);
    for (auto &dev: devices)
        if (const auto err = this->check_device_health(dev); err)
            LOG(WARNING) << SCAN_LOG_PREFIX << "health check failed for " << dev.name
                         << ": " << err;
    if (renew_err)
        return {
            devices,
            x::errors::Error(
                common::PARTIAL_SCAN_ERROR,
                "Failed to renew the Driver certificate: " + renew_err.data
            )
        };
    return {devices, x::errors::NIL};
}

bool Scanner::exec(
    synnax::task::Command &cmd,
    const synnax::task::Task &,
    const std::shared_ptr<task::Context> &
) {
    if (cmd.type == BROWSE_CMD_TYPE) {
        this->browse_nodes(cmd);
        return true;
    }
    if (cmd.type == BROWSE_VARIABLES_CMD_TYPE) {
        this->browse_variables(cmd);
        return true;
    }
    if (cmd.type == READ_NODE_CMD_TYPE) {
        this->read_node(cmd);
        return true;
    }
    if (cmd.type == TEST_CONNECTION_CMD_TYPE) {
        this->test_connection(cmd);
        return true;
    }
    if (cmd.type == GET_ENDPOINTS_CMD_TYPE) {
        this->endpoints(cmd);
        return true;
    }
    if (cmd.type == GET_CERTIFICATE_CMD_TYPE) {
        this->certificates(cmd, false);
        return true;
    }
    if (cmd.type == REGENERATE_CERTIFICATE_CMD_TYPE) {
        this->certificates(cmd, true);
        return true;
    }
    return false; // Not handled
}

x::errors::Error Scanner::check_device_health(synnax::device::Device &dev) {
    const auto rack_key = this->task.rack;
    const auto parser = x::json::Parser(dev.properties);
    const auto props = device::Properties(parser);
    synnax::device::Status status{
        .key = synnax::device::status_key(dev),
        .name = dev.name,
        .variant = synnax::status::VARIANT_SUCCESS,
        .message = "Server connected",
        .time = ::x::telem::TimeStamp::now(),
        .details = {.rack = rack_key, .device = dev.key},
    };
    if (parser.error()) {
        status.variant = synnax::status::VARIANT_WARNING;
        status.message = "Invalid device properties";
        status.description = common::config_err_description(parser.error());
        dev.status = status;
        return parser.error();
    }
    const auto &cfg = props.connection;
    const auto previous = this->checked[dev.key];
    const bool pin = cfg.secured() && !cfg.certificate_validation_disabled &&
                     !cfg.server_certificate.has_value();
    if (pin || this->conn_pool->last_reply(cfg) <= previous) {
        auto [conn, conn_err] = this->conn_pool->acquire(cfg, SCAN_LOG_PREFIX);
        // Marked after the acquire, so its own reply does not skip the next probe.
        this->checked[dev.key] = x::telem::TimeStamp::now();
        if (conn_err) {
            status.message = failure_message(conn_err, cfg, "Failed to reach server");
            status.variant = conn_err.matches(errors::SECURITY)
                               ? synnax::status::VARIANT_ERROR
                               : synnax::status::VARIANT_WARNING;
            status.description = conn_err.data;
            dev.status = status;
            return x::errors::NIL;
        }
        if (pin)
            if (const auto cert = connection::server_certificate(conn.get())) {
                LOG(INFO) << SCAN_LOG_PREFIX << "trusting server certificate "
                          << cert->thumbprint() << " for " << dev.name;
                dev.properties["connection"]["server_certificate"] = cert->pem();
            }
    } else
        this->checked[dev.key] = x::telem::TimeStamp::now();

    const auto accepted = this->renewal.accepted.find(dev.key);
    if (cfg.secured() && cfg.certificate_validation_disabled) {
        status.variant = synnax::status::VARIANT_WARNING;
        status.message = "Connected without certificate validation";
        status.description = "The Driver accepts any server certificate for this "
                             "device, so it cannot detect a server that impersonates "
                             "this one. Turn validation on again when you finish "
                             "troubleshooting.";
    } else if (accepted != this->renewal.accepted.end() && !accepted->second) {
        auto [current, current_err] = this->store->current();
        if (current_err) return current_err;
        status.variant = synnax::status::VARIANT_WARNING;
        status.message = "Server does not trust the next certificate";
        status.description = "Approve the next Driver certificate " +
                             this->renewal.thumbprint + " on the server before " +
                             current.certificate.not_after().iso8601() +
                             ", when the current certificate expires.";
    }
    dev.status = status;
    return x::errors::NIL;
}

x::errors::Error
Scanner::renew_certificate(const std::vector<synnax::device::Device> &devices) {
    if (const auto err = this->store->renew()) return err;
    auto [next, next_err] = this->store->next();
    if (next_err) return next_err;
    if (!next.has_value()) {
        this->renewal = Renewal{};
        return x::errors::NIL;
    }
    const auto thumbprint = next->certificate.thumbprint();
    if (this->renewal.thumbprint != thumbprint)
        this->renewal = {.thumbprint = thumbprint};
    const auto now = x::telem::TimeStamp::now();
    if (this->renewal.last_attempt != x::telem::TimeStamp() &&
        now - this->renewal.last_attempt < RENEWAL_ATTEMPT_INTERVAL)
        return x::errors::NIL;
    this->renewal.last_attempt = now;
    this->renewal.accepted.clear();
    bool all_accepted = true;
    for (const auto &dev: devices) {
        const auto parser = x::json::Parser(dev.properties);
        const auto props = device::Properties(parser);
        if (parser.error() || !props.connection.secured() ||
            props.connection.uses_files())
            continue;
        const connection::Security sec{
            .identity = next,
            .first_certificate_trusted = true,
        };
        const auto err = connection::connect(props.connection, sec, SCAN_LOG_PREFIX)
                             .second;
        if (err && !err.matches(errors::CLIENT_CERTIFICATE_REJECTED)) {
            all_accepted = false;
            continue;
        }
        this->renewal.accepted[dev.key] = !err;
        all_accepted = all_accepted && !err;
    }
    if (!all_accepted) return x::errors::NIL;
    if (const auto err = this->store->promote()) return err;
    LOG(INFO) << SCAN_LOG_PREFIX << "switched to the next certificate " << thumbprint;
    this->renewal = Renewal{};
    return x::errors::NIL;
}

void Scanner::certificates(
    const synnax::task::Command &cmd,
    const bool regenerate
) const {
    auto status = command_status(this->task, cmd);
    x::errors::Error err = x::errors::NIL;
    if (regenerate) err = this->store->regenerate().second;
    auto [current, current_err] = this->store->current();
    auto [next, next_err] = this->store->next();
    if (!err) err = current_err ? current_err : next_err;
    if (err) {
        status.message = regenerate ? "Failed to regenerate the Driver certificate"
                                    : "Failed to load the Driver certificate";
        status.description = err.data;
        ctx->set_status(status);
        return;
    }
    x::json::json data = {{"current", current.certificate.info().to_json()}};
    if (next.has_value()) data["next"] = next->certificate.info().to_json();
    status.variant = synnax::status::VARIANT_SUCCESS;
    status.message = regenerate ? "Regenerated the Driver certificate"
                                : "Retrieved the Driver certificate";
    status.details.data = data;
    ctx->set_status(status);
}

namespace {
std::string text(const UA_String &s) {
    return std::string(reinterpret_cast<const char *>(s.data), s.length);
}

/// @returns the scalar of type T that the result holds, or nothing.
template<typename T>
std::optional<T> scalar(const UA_DataValue &result, const UA_DataType &type) {
    if (!result.hasValue || !UA_Variant_hasScalarType(&result.value, &type))
        return std::nullopt;
    return *static_cast<const T *>(result.value.data);
}

/// @brief the data type of a variable and whether it holds an array.
struct VariableType {
    x::telem::DataType data_type = x::telem::UNKNOWN_T;
    bool is_array = false;
};

/// @brief finds the type of a variable from the results of reading its DataType and
/// ValueRank. When the DataType is abstract or unknown, or the ValueRank allows both a
/// scalar and an array, value tells the rest. value is nullptr when it was not read.
/// @returns the type, and whether the DataType and ValueRank alone set it.
std::pair<VariableType, bool> variable_type(
    const UA_DataValue &data_type,
    const UA_DataValue &value_rank,
    const UA_DataValue *value
) {
    VariableType type;
    if (const auto id = scalar<UA_NodeId>(data_type, UA_TYPES[UA_TYPES_NODEID]))
        type.data_type = telem::to_data_type(*id);
    const auto rank = scalar<UA_Int32>(value_rank, UA_TYPES[UA_TYPES_INT32]);
    const bool rank_known = rank.has_value() && *rank >= UA_VALUERANK_SCALAR;
    type.is_array = rank_known && *rank != UA_VALUERANK_SCALAR;
    const bool declared = type.data_type != x::telem::UNKNOWN_T && rank_known;
    if (value == nullptr || !value->hasValue) return {type, declared};
    if (type.data_type == x::telem::UNKNOWN_T)
        type.data_type = telem::to_data_type(value->value.type);
    if (!rank_known) type.is_array = !UA_Variant_isScalar(&value->value);
    return {type, declared};
}

/// @brief finds the types of the variables that refs name from their DataType and
/// ValueRank. When a DataType is abstract or unknown, or a ValueRank allows both a
/// scalar and an array, the current value tells the type.
/// @returns one type for each reference, unknown for a node that is not a variable.
std::pair<std::vector<VariableType>, x::errors::Error> variable_types(
    const connection::Pool::Connection &conn,
    const std::span<const types::Reference> refs
) {
    std::vector<VariableType> out(refs.size());
    std::vector<size_t> variables;
    types::ReadRequestBuilder req;
    for (size_t i = 0; i < refs.size(); ++i) {
        if (refs[i].node_class != UA_NODECLASS_VARIABLE) continue;
        req.add_node(refs[i].node.resolve({}).value(), UA_ATTRIBUTEID_DATATYPE);
        req.add_node(refs[i].node.resolve({}).value(), UA_ATTRIBUTEID_VALUERANK);
        variables.push_back(i);
    }
    if (variables.empty()) return {std::move(out), x::errors::NIL};
    auto [res, err] = conn.read(req.build());
    if (err) return {{}, err};
    const auto *results = res.get().results;
    std::vector<size_t> unknown;
    types::ReadRequestBuilder values;
    for (size_t j = 0; j < variables.size(); ++j) {
        bool declared = false;
        std::tie(
            out[variables[j]],
            declared
        ) = variable_type(results[2 * j], results[2 * j + 1], nullptr);
        if (declared) continue;
        values.add_node(
            refs[variables[j]].node.resolve({}).value(),
            UA_ATTRIBUTEID_VALUE
        );
        unknown.push_back(j);
    }
    if (unknown.empty()) return {std::move(out), x::errors::NIL};
    auto [value_res, value_err] = conn.read(values.build());
    if (value_err) return {{}, value_err};
    for (size_t k = 0; k < unknown.size(); ++k) {
        const auto j = unknown[k];
        out[variables[j]] = variable_type(
                                results[2 * j],
                                results[2 * j + 1],
                                &value_res.get().results[k]
        )
                                .first;
    }
    return {std::move(out), x::errors::NIL};
}

/// @brief the most variables in one step of reading types, when the server has no limit
/// on the nodes in a Read request.
constexpr size_t TYPE_BATCH = 1000;

/// @returns the nodes that refs name, with the types of the variables. Reads the types
/// in steps of about one Read request, and stops between steps once deadline passes, so
/// it can return only the first nodes of refs. Always reads the first step.
std::pair<std::vector<types::Node>, x::errors::Error> to_nodes(
    const connection::Pool::Connection &conn,
    const std::vector<types::Reference> &refs,
    const std::chrono::steady_clock::time_point deadline
) {
    // Each variable takes two nodes of a request: its DataType and its ValueRank.
    const size_t limit = conn.server().max_nodes_per_read;
    const size_t step = limit == 0 ? TYPE_BATCH : std::max<size_t>(limit / 2, 1);
    std::vector<types::Node> nodes;
    nodes.reserve(refs.size());
    for (size_t head = 0; head < refs.size(); head += step) {
        if (head > 0 && std::chrono::steady_clock::now() >= deadline) break;
        const std::span part(refs.data() + head, std::min(step, refs.size() - head));
        auto [variables, err] = variable_types(conn, part);
        if (err) return {{}, err};
        for (size_t i = 0; i < part.size(); ++i)
            nodes.emplace_back(
                variables[i].data_type,
                part[i].browse_name,
                types::NodeId::to_string(
                    part[i].node.resolve({}).value(),
                    conn.server().namespaces
                ),
                types::node_class_to_string(part[i].node_class),
                variables[i].is_array
            );
    }
    return {std::move(nodes), x::errors::NIL};
}

/// @brief lists the members of the structure value of a variable, at path. A node that
/// holds no structure has no members.
/// @returns x::errors::VALIDATION if the structure type of the node does not build, or
/// if path is not empty and does not name a structure member of the node's value.
std::pair<std::vector<types::Node>, x::errors::Error> structure_members(
    const connection::Pool::Connection &conn,
    const UA_NodeId &id,
    const std::string &node_id,
    const structure::Path &path
) {
    const auto none =
        [&path]() -> std::pair<std::vector<types::Node>, x::errors::Error> {
        if (path.empty()) return {{}, x::errors::NIL};
        return {
            {},
            x::errors::Error(
                x::errors::VALIDATION,
                "the node does not hold a structure"
            )
        };
    };
    types::ReadRequestBuilder req;
    req.add_node(id, UA_ATTRIBUTEID_NODECLASS);
    req.add_node(id, UA_ATTRIBUTEID_DATATYPE);
    auto [res, err] = conn.read(req.build());
    if (err) return {{}, err};
    const auto &cls = res.get().results[0].value;
    const auto &dt = res.get().results[1].value;
    // The client decodes the NodeClass enumeration as its Int32 encoding.
    if (!UA_Variant_hasScalarType(&cls, &UA_TYPES[UA_TYPES_INT32]) ||
        *static_cast<const UA_Int32 *>(cls.data) != UA_NODECLASS_VARIABLE ||
        !UA_Variant_hasScalarType(&dt, &UA_TYPES[UA_TYPES_NODEID]))
        return none();
    const auto &data_type = *static_cast<const UA_NodeId *>(dt.data);
    auto [types, types_err] = structure::Types::read(conn, {data_type});
    if (types_err) return {{}, types_err};
    auto [field, field_err] = types.field(data_type, path);
    if (field_err.matches(structure::UNDEFINED) && path.empty())
        return {{}, x::errors::NIL};
    if (field_err) return {{}, field_err};
    std::vector<types::Node> members;
    for (const auto &m: structure::Types::members(field.type())) {
        auto member_path = path;
        member_path.push_back(m.name);
        members.emplace_back(
            telem::to_data_type(m.type),
            m.name,
            node_id,
            types::node_class_to_string(UA_NODECLASS_VARIABLE),
            m.is_array,
            structure::to_string(member_path)
        );
    }
    return {std::move(members), x::errors::NIL};
}

/// @brief a variable that a search under a node found.
struct Found {
    types::Reference ref;
    /// @brief the browse names from below the node where the search started.
    std::vector<std::string> path;
};

/// @brief what a search for the variables under a node found.
struct Search {
    std::vector<Found> variables;
    /// @brief whether the search stopped at a limit before it found every variable.
    bool truncated = false;
    /// @brief why each object that failed to browse was skipped.
    std::vector<x::errors::Error> skipped;
};

/// @brief the most objects in one Browse request of a search for variables.
constexpr size_t SEARCH_BATCH = 100;
/// @brief how long a browse_variables command runs before it returns what it found,
/// which is less than the 30 s that the Console waits for the reply.
constexpr auto SEARCH_DEADLINE = std::chrono::seconds(20);
/// @brief how long a browse command runs before it returns the nodes that it typed,
/// which is less than the 10 s that the Console waits for the reply.
constexpr auto BROWSE_DEADLINE = std::chrono::seconds(7);

/// @brief finds the variables under a node, through objects at any depth, breadth
/// first, until deadline. Does not look under a variable. A node that fails to browse
/// is skipped, and its error goes in the result.
std::pair<Search, x::errors::Error> find_variables(
    const connection::Pool::Connection &conn,
    const UA_NodeId &start,
    const std::chrono::steady_clock::time_point deadline
) {
    struct Object {
        types::NodeId node;
        std::vector<std::string> path;
    };
    Search search;
    std::set<std::string> visited{types::NodeId::to_string(start)};
    std::vector<Object> objects;
    objects.push_back({.node = types::NodeId(start), .path = {}});
    const size_t limit = conn.server().max_nodes_per_browse;
    const size_t batch = limit == 0 ? SEARCH_BATCH : std::min(limit, SEARCH_BATCH);
    for (size_t head = 0; head < objects.size();) {
        if (search.variables.size() >= MAX_VARIABLES ||
            std::chrono::steady_clock::now() >= deadline) {
            search.truncated = true;
            break;
        }
        const size_t count = std::min(batch, objects.size() - head);
        std::vector<UA_NodeId> ids;
        ids.reserve(count);
        for (size_t i = head; i < head + count; ++i)
            ids.push_back(objects[i].node.resolve({}).value());
        auto [results, err] = conn.browse(ids, deadline);
        if (err) return {Search{}, err};
        for (size_t i = 0; i < count; ++i) {
            if (const auto &browse_err = results[i].err) {
                search.skipped.emplace_back(
                    browse_err,
                    objects[head + i].node.to_string() + ": " + browse_err.data
                );
                continue;
            }
            search.truncated = search.truncated || results[i].truncated;
            for (auto &ref: results[i].references) {
                if (ref.node_class != UA_NODECLASS_VARIABLE &&
                    ref.node_class != UA_NODECLASS_OBJECT)
                    continue;
                const bool object = ref.node_class == UA_NODECLASS_OBJECT;
                if ((object && objects.size() >= MAX_OBJECTS) ||
                    (!object && search.variables.size() >= MAX_VARIABLES)) {
                    search.truncated = true;
                    continue;
                }
                if (!visited.insert(ref.node.to_string()).second) continue;
                auto path = objects[head + i].path;
                path.push_back(ref.browse_name);
                if (object)
                    objects.push_back(
                        {.node = std::move(ref.node), .path = std::move(path)}
                    );
                else
                    search.variables.push_back(
                        {.ref = std::move(ref), .path = std::move(path)}
                    );
            }
        }
        head += count;
    }
    return {std::move(search), x::errors::NIL};
}

/// @returns the DateTime as ISO 8601 text, or nothing when the server did not set it or
/// a timestamp cannot hold it.
std::optional<std::string> date_text(const UA_DateTime time) {
    if (time == 0) return std::nullopt;
    const auto [ts, err] = telem::to_timestamp(time);
    if (err) return std::nullopt;
    return ts.iso8601();
}

/// @returns the name of the data type, reading its BrowseName when the client does not
/// know it.
std::pair<std::string, x::errors::Error>
type_name(const connection::Pool::Connection &conn, const UA_NodeId &data_type) {
    if (const auto *type = UA_findDataType(&data_type))
        return {type->typeName, x::errors::NIL};
    types::ReadRequestBuilder req;
    req.add_node(data_type, UA_ATTRIBUTEID_BROWSENAME);
    auto [res, err] = conn.read(req.build());
    if (err) return {"", err};
    const auto name = scalar<UA_QualifiedName>(
        res.get().results[0],
        UA_TYPES[UA_TYPES_QUALIFIEDNAME]
    );
    if (!name.has_value()) return {types::NodeId::to_string(data_type), x::errors::NIL};
    return {text(name->name), x::errors::NIL};
}

/// @returns the attributes and current value of a node as the reply to a read node
/// command. A field reads a member of the structure value of the node.
/// @returns x::errors::VALIDATION if field does not name a member of the node's value.
std::pair<x::json::json, x::errors::Error> read_details(
    const connection::Pool::Connection &conn,
    const UA_NodeId &id,
    const structure::Path &field
) {
    constexpr std::array attributes{
        UA_ATTRIBUTEID_NODECLASS,
        UA_ATTRIBUTEID_BROWSENAME,
        UA_ATTRIBUTEID_DISPLAYNAME,
        UA_ATTRIBUTEID_DESCRIPTION,
        UA_ATTRIBUTEID_DATATYPE,
        UA_ATTRIBUTEID_VALUERANK,
        UA_ATTRIBUTEID_ARRAYDIMENSIONS,
        UA_ATTRIBUTEID_USERACCESSLEVEL,
        UA_ATTRIBUTEID_HISTORIZING,
        UA_ATTRIBUTEID_MINIMUMSAMPLINGINTERVAL,
        UA_ATTRIBUTEID_VALUE,
    };
    types::ReadRequestBuilder req;
    for (const auto attribute: attributes)
        req.add_node(id, attribute);
    auto raw_req = req.build();
    raw_req.timestampsToReturn = UA_TIMESTAMPSTORETURN_BOTH;
    auto [res, read_err] = conn.read(raw_req);
    if (read_err) return {{}, read_err};
    const auto *r = res.get().results;
    if (const auto node_err = errors::parse(r[0].status)) return {{}, node_err};
    // The client decodes the NodeClass enumeration as its Int32 encoding.
    const auto cls = static_cast<UA_NodeClass>(
        scalar<UA_Int32>(r[0], UA_TYPES[UA_TYPES_INT32]).value_or(0)
    );
    const auto browse_name = scalar<UA_QualifiedName>(
        r[1],
        UA_TYPES[UA_TYPES_QUALIFIEDNAME]
    );
    const std::string name = browse_name.has_value() ? text(browse_name->name) : "";
    x::json::json out = {
        {"node_class", types::node_class_to_string(cls)},
        {"browse_name", name},
    };
    if (const auto v = scalar<UA_LocalizedText>(r[2], UA_TYPES[UA_TYPES_LOCALIZEDTEXT]))
        out["display_name"] = text(v->text);
    if (const auto v = scalar<UA_LocalizedText>(r[3], UA_TYPES[UA_TYPES_LOCALIZEDTEXT]))
        out["description"] = text(v->text);
    VariableType type;
    if (cls == UA_NODECLASS_METHOD) {
        auto [inputs, inputs_err] = method::read_inputs(conn, {id});
        if (inputs_err) return {{}, inputs_err};
        if (const auto &declared = inputs[0]; declared.has_value()) {
            auto args = x::json::json::array();
            for (const auto &arg: *declared) {
                auto [arg_type, name_err] = type_name(
                    conn,
                    arg.data_type.resolve({}).value()
                );
                if (name_err) return {{}, name_err};
                args.push_back({
                    {"name", arg.name},
                    {"type_name", arg_type},
                    {"data_type",
                     telem::to_data_type(arg.data_type.resolve({}).value()).name()},
                });
            }
            out["input_arguments"] = args;
        }
    }
    if (cls == UA_NODECLASS_VARIABLE) {
        type = variable_type(r[4], r[5], &r[10]).first;
        const auto type_id = scalar<UA_NodeId>(r[4], UA_TYPES[UA_TYPES_NODEID]);
        const auto rank = scalar<UA_Int32>(r[5], UA_TYPES[UA_TYPES_INT32]);
        if (type_id.has_value()) {
            auto [type_text, name_err] = type_name(conn, *type_id);
            if (name_err) return {{}, name_err};
            out["type_name"] = type_text;
        }
        if (rank.has_value()) out["value_rank"] = *rank;
        if (r[6].hasValue &&
            UA_Variant_hasArrayType(&r[6].value, &UA_TYPES[UA_TYPES_UINT32])) {
            const auto *dims = static_cast<const UA_UInt32 *>(r[6].value.data);
            out["array_dimensions"] = std::vector<UA_UInt32>(
                dims,
                dims + r[6].value.arrayLength
            );
        }
        if (const auto access = scalar<UA_Byte>(r[7], UA_TYPES[UA_TYPES_BYTE])) {
            out["readable"] = (*access & UA_ACCESSLEVELMASK_READ) != 0;
            out["writable"] = (*access & UA_ACCESSLEVELMASK_WRITE) != 0;
            out["history_readable"] = (*access & UA_ACCESSLEVELMASK_HISTORYREAD) != 0;
        }
        if (const auto v = scalar<UA_Boolean>(r[8], UA_TYPES[UA_TYPES_BOOLEAN]))
            out["historizing"] = *v;
        if (const auto v = scalar<UA_Double>(r[9], UA_TYPES[UA_TYPES_DOUBLE]))
            out["minimum_sampling_interval"] = *v;
        const auto &value = r[10];
        out["status"] = UA_StatusCode_name(value.status);
        if (value.hasSourceTimestamp)
            if (const auto t = date_text(value.sourceTimestamp))
                out["source_timestamp"] = *t;
        if (value.hasServerTimestamp)
            if (const auto t = date_text(value.serverTimestamp))
                out["server_timestamp"] = *t;
        structure::Types types;
        if (type_id.has_value() &&
            (!field.empty() ||
             (value.value.type != nullptr &&
              value.value.type->typeKind == UA_DATATYPEKIND_EXTENSIONOBJECT))) {
            auto [read_types, types_err] = structure::Types::read(conn, {*type_id});
            if (types_err) return {{}, types_err};
            types = std::move(read_types);
        }
        if (!field.empty()) {
            if (!type_id.has_value())
                return {
                    {},
                    x::errors::Error(
                        x::errors::VALIDATION,
                        "the node does not hold a structure"
                    )
                };
            auto [member, member_err] = types.field(*type_id, field);
            if (member_err) return {{}, member_err};
            out["type_name"] = member.type().typeName;
            type = {
                .data_type = telem::to_data_type(&member.type()),
                .is_array = false,
            };
        }
        if (value.hasValue) {
            auto [value_text, text_err] = types.print(value.value, field);
            if (text_err) return {{}, text_err};
            out["value"] = value_text;
        }
    }
    out["node"] = types::Node(
                      type.data_type,
                      field.empty() ? name : field.back(),
                      types::NodeId::to_string(id, conn.server().namespaces),
                      types::node_class_to_string(cls),
                      type.is_array,
                      structure::to_string(field)
    )
                      .to_json();
    return {out, x::errors::NIL};
}
}

std::optional<std::pair<connection::Pool::Connection, UA_NodeId>> Scanner::target(
    const x::json::Parser &parser,
    const ScanCommandArgs &args,
    synnax::task::Status &status,
    const std::string &failure
) const {
    if (!parser.ok()) {
        status.message = "Failed to parse scan command";
        status.description = common::config_err_description(parser.error());
        status.details.data = parser.error_json();
        ctx->set_status(status);
        return std::nullopt;
    }
    auto [connection, err] = conn_pool->acquire(args.connection, SCAN_LOG_PREFIX);
    if (err) {
        status.message = "Failed to connect to server";
        status.description = err.data;
        ctx->set_status(status);
        return std::nullopt;
    }
    // A namespace that the server added since the connection opened would print as an
    // index, which a restart can move.
    types::ReadRequestBuilder ns_req;
    ns_req.add_node(UA_NODEID_NUMERIC(0, UA_NS0ID_SERVER_NAMESPACEARRAY));
    auto [ns_res, ns_err] = connection.read(ns_req.build());
    if (!ns_err) ns_err = connection.update_namespaces(ns_res.get().results[0]).second;
    if (ns_err) {
        status.message = failure;
        status.description = ns_err.data;
        ctx->set_status(status);
        return std::nullopt;
    }
    const auto node = args.node.resolve(connection.server().namespaces);
    if (!node.has_value()) {
        status.message = failure;
        status.description = "The server has no namespace " +
                             args.node.namespace_uri() + ".";
        ctx->set_status(status);
        return std::nullopt;
    }
    return std::pair{std::move(connection), node.value()};
}

void Scanner::browse_nodes(const synnax::task::Command &cmd) const {
    const auto deadline = std::chrono::steady_clock::now() + BROWSE_DEADLINE;
    x::json::Parser parser(cmd.args);
    const ScanCommandArgs args(parser);
    auto status = command_status(this->task, cmd);
    auto target = this->target(parser, args, status, "Failed to browse nodes");
    if (!target.has_value()) return;
    auto &[connection, start] = *target;

    x::errors::Error err;
    std::vector<types::Node> nodes;
    bool truncated = false;
    if (args.field.empty()) {
        auto [results, browse_err] = connection.browse({start}, deadline);
        err = browse_err ? browse_err : results[0].err;
        if (!err) {
            const auto &refs = results[0].references;
            std::tie(nodes, err) = to_nodes(connection, refs, deadline);
            truncated = results[0].truncated || nodes.size() < refs.size();
        }
    }
    x::errors::Error members_err;
    if (!err) {
        auto [members, list_err] = structure_members(
            connection,
            start,
            args.node_id,
            args.field
        );
        for (auto &m: members)
            nodes.push_back(std::move(m));
        // A member has no children besides its own members.
        if (args.field.empty())
            members_err = list_err;
        else
            err = list_err;
    }
    if (err) {
        status.message = "Failed to browse nodes";
        status.description = err.data;
        ctx->set_status(status);
        return;
    }

    x::json::json channels = x::json::json::array();
    for (const auto &node: nodes)
        channels.push_back(node.to_json());
    status.message = "Browsed nodes";
    status.variant = synnax::status::VARIANT_SUCCESS;
    if (members_err) {
        status.message = "Failed to list structure members";
        status.variant = synnax::status::VARIANT_WARNING;
        status.description = members_err.data;
    }
    status.details.data = {{"channels", channels}, {"truncated", truncated}};
    ctx->set_status(status);
}

void Scanner::browse_variables(const synnax::task::Command &cmd) const {
    const auto deadline = std::chrono::steady_clock::now() + SEARCH_DEADLINE;
    x::json::Parser parser(cmd.args);
    const ScanCommandArgs args(parser);
    auto status = command_status(this->task, cmd);
    auto target = this->target(parser, args, status, "Failed to find variables");
    if (!target.has_value()) return;
    auto &[connection, start] = *target;
    auto [search, search_err] = find_variables(connection, start, deadline);
    std::vector<types::Reference> refs;
    refs.reserve(search.variables.size());
    for (auto &found: search.variables)
        refs.push_back(std::move(found.ref));
    std::vector<types::Node> nodes;
    auto err = search_err;
    if (!err) std::tie(nodes, err) = to_nodes(connection, refs, deadline);
    if (err) {
        status.message = "Failed to find variables";
        status.description = err.data;
        ctx->set_status(status);
        return;
    }
    x::json::json skipped = x::json::json::array();
    for (const auto &skip_err: search.skipped)
        skipped.push_back(skip_err.data);
    x::json::json channels = x::json::json::array();
    for (size_t i = 0; i < nodes.size(); ++i) {
        auto node = nodes[i].to_json();
        node["path"] = search.variables[i].path;
        channels.push_back(node);
    }
    status.message = "Found variables";
    status.variant = synnax::status::VARIANT_SUCCESS;
    status.details.data = {
        {"channels", channels},
        {"truncated", search.truncated || nodes.size() < refs.size()},
        {"skipped", skipped},
    };
    ctx->set_status(status);
}

void Scanner::read_node(const synnax::task::Command &cmd) const {
    x::json::Parser parser(cmd.args);
    const ScanCommandArgs args(parser);
    auto status = command_status(this->task, cmd);
    auto target = this->target(parser, args, status, "Failed to read node");
    if (!target.has_value()) return;
    auto &[connection, id] = *target;
    auto [data, details_err] = read_details(connection, id, args.field);
    if (details_err) {
        status.message = "Failed to read node";
        status.description = details_err.data;
        ctx->set_status(status);
        return;
    }
    status.message = "Read node";
    status.variant = synnax::status::VARIANT_SUCCESS;
    status.details.data = data;
    ctx->set_status(status);
}

void Scanner::test_connection(const synnax::task::Command &cmd) const {
    x::json::Parser parser(cmd.args);
    const ScanCommandArgs args(parser);
    auto status = command_status(this->task, cmd);
    if (!parser.ok()) {
        status.message = "Failed to parse test command";
        status.description = common::config_err_description(parser.error());
        status.details.data = parser.error_json();
        ctx->set_status(status);
        return;
    }
    auto [sec, sec_err] = connection::resolve_security(args.connection, *this->store);
    if (sec_err) {
        status.message = failure_message(
            sec_err,
            args.connection,
            "Failed to connect to server"
        );
        status.description = sec_err.data;
        ctx->set_status(status);
        return;
    }
    auto [client, err] = connection::connect(args.connection, sec, SCAN_LOG_PREFIX);
    if (err) {
        status.message = failure_message(
            err,
            args.connection,
            "Failed to connect to server"
        );
        status.description = err.data;
        if (err.matches(errors::SERVER_CERTIFICATE_UNTRUSTED))
            if (const auto cert = connection::server_certificate(client.get()))
                status.details.data = {{"server_certificate", cert->info().to_json()}};
        ctx->set_status(status);
        return;
    }
    status.variant = synnax::status::VARIANT_SUCCESS;
    status.message = "Connected to server";
    ctx->set_status(status);
}

void Scanner::endpoints(const synnax::task::Command &cmd) const {
    x::json::Parser parser(cmd.args);
    const connection::Config cfg(parser.child("connection"));
    auto status = command_status(this->task, cmd);
    if (!parser.ok()) {
        status.message = "Failed to parse endpoints command";
        status.description = common::config_err_description(parser.error());
        status.details.data = parser.error_json();
        ctx->set_status(status);
        return;
    }
    auto [endpoints, err] = connection::endpoints(cfg);
    if (err) {
        status.message = "Failed to get endpoints";
        status.description = err.data;
        ctx->set_status(status);
        return;
    }
    x::json::json data = x::json::json::array();
    for (const auto &endpoint: endpoints)
        data.push_back(endpoint.to_json());
    status.variant = synnax::status::VARIANT_SUCCESS;
    status.message = "Received endpoints";
    status.details.data = {{"endpoints", data}};
    ctx->set_status(status);
}
}
