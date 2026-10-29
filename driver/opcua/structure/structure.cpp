// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <new>
#include <string>
#include <utility>
#include <vector>

#include "absl/log/log.h"
#include "open62541/types.h"
#include "open62541/util.h"

#include "driver/opcua/structure/structure.h"
#include "driver/opcua/telem/telem.h"
#include "driver/opcua/types/types.h"

namespace driver::opcua::structure {
namespace {
/// @brief the most rounds of member types that Types::read follows.
constexpr int MAX_DEPTH = 16;
/// @brief the most types that Types::read defines for one server.
constexpr std::size_t MAX_TYPES = 1024;

x::errors::Error invalid(const std::string &message) {
    return x::errors::Error(x::errors::VALIDATION, message);
}

/// @brief a member of a structure type, and its place in a value of the type.
struct Place {
    const UA_DataTypeMember *member = nullptr;
    std::size_t offset = 0;
};

/// @returns the member named name of a structure type.
std::pair<Place, x::errors::Error>
find_member(const UA_DataType &type, const std::string &name) {
    Place place;
    const UA_DataType *member_type = nullptr;
    UA_Boolean is_array = false;
    if (!UA_DataType_getStructMember(
            &type,
            name.c_str(),
            &place.offset,
            &member_type,
            &is_array
        ))
        return {{}, invalid(std::string(type.typeName) + " has no member " + name)};
    if (is_array) return {{}, invalid(name + " is an array")};
    for (std::size_t i = 0; i < type.membersSize; ++i)
        if (name == type.members[i].memberName) place.member = &type.members[i];
    return {place, x::errors::NIL};
}

/// @returns one value of a type as JSON text.
std::pair<std::string, x::errors::Error>
json(const void *data, const UA_DataType *type) {
    UA_EncodeJsonOptions options{};
    options.stringNodeIds = true;
    UA_ByteString out = UA_BYTESTRING_NULL;
    if (const auto status = UA_encodeJson(data, type, &out, &options);
        status != UA_STATUSCODE_GOOD)
        return {
            "",
            invalid(
                std::string("the value does not encode: ") + UA_StatusCode_name(status)
            )
        };
    std::string text(reinterpret_cast<const char *>(out.data), out.length);
    UA_ByteString_clear(&out);
    return {std::move(text), x::errors::NIL};
}

/// @returns the value as JSON text, where an array is a JSON array of its elements.
std::pair<std::string, x::errors::Error> json(const UA_Variant &value) {
    if (value.type == nullptr) return {"null", x::errors::NIL};
    if (UA_Variant_isScalar(&value)) return json(value.data, value.type);
    std::string text = "[";
    const auto *data = static_cast<const char *>(value.data);
    for (std::size_t i = 0; i < value.arrayLength; ++i) {
        auto [element, err] = json(data + i * value.type->memSize, value.type);
        if (err) return {"", err};
        if (i > 0) text += ", ";
        text += element;
    }
    return {text + "]", x::errors::NIL};
}

bool contains(const std::vector<types::NodeId> &ids, const UA_NodeId &id) {
    for (const auto &seen: ids)
        if (UA_NodeId_equal(&seen.get(), &id)) return true;
    return false;
}

/// @brief the definition of a type on the server, borrowed from a read response.
struct Definition {
    std::size_t id;
    const UA_Variant *definition;
    const UA_QualifiedName *name;
};

/// @brief owns a structure value that Types::sample decoded.
struct Decoded {
    void *data = nullptr;
    const UA_DataType *type = nullptr;

    Decoded() = default;
    Decoded(const Decoded &) = delete;
    Decoded &operator=(const Decoded &) = delete;
    ~Decoded() {
        if (data != nullptr) UA_delete(data, type);
    }
};
}

std::pair<Path, x::errors::Error> parse(const std::string &field) {
    Path path;
    std::string name;
    const auto empty = [&field] {
        return std::pair{Path{}, invalid("the field " + field + " has an empty name")};
    };
    for (std::size_t i = 0; i < field.size(); ++i) {
        const char c = field[i];
        if (c == '&') {
            if (++i == field.size())
                return {Path{}, invalid("the field " + field + " ends in &")};
            name.push_back(field[i]);
        } else if (c == '.') {
            if (name.empty()) return empty();
            path.push_back(std::move(name));
            name.clear();
        } else
            name.push_back(c);
    }
    if (name.empty()) return empty();
    path.push_back(std::move(name));
    return {std::move(path), x::errors::NIL};
}

std::string to_string(const Path &path) {
    std::string field;
    for (std::size_t i = 0; i < path.size(); ++i) {
        if (i > 0) field.push_back('.');
        for (const char c: path[i]) {
            if (c == '.' || c == '&') field.push_back('&');
            field.push_back(c);
        }
    }
    return field;
}

const UA_DataTypeArray *Types::head() const {
    if (this->entries.empty()) return nullptr;
    return &this->entries.back()->array;
}

Types::~Types() {
    for (auto &entry: this->entries)
        UA_DataType_clear(&entry->type);
}

Types::Types(Types &&other) noexcept: entries(std::move(other.entries)) {
    other.entries.clear();
}

Types &Types::operator=(Types &&other) noexcept {
    if (this != &other) {
        for (auto &entry: this->entries)
            UA_DataType_clear(&entry->type);
        this->entries = std::move(other.entries);
        other.entries.clear();
    }
    return *this;
}

std::pair<Types, x::errors::Error> Types::read(
    const connection::Pool::Connection &conn,
    const std::vector<UA_NodeId> &nodes
) {
    Types out;
    if (nodes.empty()) return {std::move(out), x::errors::NIL};
    types::ReadRequestBuilder type_req;
    for (const auto &node: nodes)
        type_req.add_node(node, UA_ATTRIBUTEID_DATATYPE);
    auto [node_types, err] = conn.read(type_req.build());
    if (err) return {std::move(out), err};

    std::vector<types::NodeId> seen;
    std::vector<std::size_t> round;
    const auto visit = [&](const UA_NodeId &id) {
        if (UA_findDataType(&id) != nullptr || contains(seen, id) ||
            seen.size() >= MAX_TYPES)
            return;
        seen.emplace_back(id);
        round.push_back(seen.size() - 1);
    };
    for (std::size_t i = 0; i < node_types.get().resultsSize; ++i)
        if (const auto &value = node_types.get().results[i].value;
            UA_Variant_hasScalarType(&value, &UA_TYPES[UA_TYPES_NODEID]))
            visit(*static_cast<const UA_NodeId *>(value.data));

    std::vector<types::ReadResponse> responses;
    std::vector<Definition> definitions;
    for (int depth = 0; !round.empty() && depth < MAX_DEPTH; ++depth) {
        const auto current = std::exchange(round, {});
        types::ReadRequestBuilder def_req;
        for (const auto idx: current) {
            def_req.add_node(seen[idx].get(), UA_ATTRIBUTEID_DATATYPEDEFINITION);
            def_req.add_node(seen[idx].get(), UA_ATTRIBUTEID_BROWSENAME);
        }
        auto [res, def_err] = conn.read(def_req.build());
        if (def_err) return {std::move(out), def_err};
        const auto &raw = res.get();
        for (std::size_t k = 0; k < current.size(); ++k) {
            const auto &def = raw.results[2 * k];
            const auto &name = raw.results[2 * k + 1];
            if (!def.hasValue || !UA_Variant_hasScalarType(
                                     &name.value,
                                     &UA_TYPES[UA_TYPES_QUALIFIEDNAME]
                                 ))
                continue;
            if (UA_Variant_hasScalarType(
                    &def.value,
                    &UA_TYPES[UA_TYPES_STRUCTUREDEFINITION]
                )) {
                const auto *sd = static_cast<const UA_StructureDefinition *>(
                    def.value.data
                );
                for (std::size_t f = 0; f < sd->fieldsSize; ++f)
                    visit(sd->fields[f].dataType);
            } else if (!UA_Variant_hasScalarType(
                           &def.value,
                           &UA_TYPES[UA_TYPES_ENUMDEFINITION]
                       ))
                continue;
            definitions.push_back({
                .id = current[k],
                .definition = &def.value,
                .name = static_cast<const UA_QualifiedName *>(name.value.data),
            });
        }
        responses.push_back(std::move(res));
    }

    // A type builds once the types of its members exist, so each pass builds the
    // types whose members the earlier passes built.
    std::vector<bool> done(definitions.size(), false);
    for (bool progress = true; progress;) {
        progress = false;
        for (std::size_t i = 0; i < definitions.size(); ++i) {
            if (done[i]) continue;
            const auto &d = definitions[i];
            // Namespace 0 keeps the index out of the type name in error messages.
            const UA_QualifiedName name{0, d.name->name};
            UA_ExtensionObject description;
            UA_StructureDescription structure;
            UA_EnumDescription enumeration;
            if (d.definition->type == &UA_TYPES[UA_TYPES_STRUCTUREDEFINITION]) {
                UA_StructureDescription_init(&structure);
                structure.dataTypeId = seen[d.id].get();
                structure.name = name;
                structure
                    .structureDefinition = *static_cast<const UA_StructureDefinition *>(
                    d.definition->data
                );
                UA_ExtensionObject_setValue(
                    &description,
                    &structure,
                    &UA_TYPES[UA_TYPES_STRUCTUREDESCRIPTION]
                );
            } else {
                UA_EnumDescription_init(&enumeration);
                enumeration.dataTypeId = seen[d.id].get();
                enumeration.name = name;
                enumeration.enumDefinition = *static_cast<const UA_EnumDefinition *>(
                    d.definition->data
                );
                enumeration.builtInType = UA_DATATYPEKIND_INT32 + 1;
                UA_ExtensionObject_setValue(
                    &description,
                    &enumeration,
                    &UA_TYPES[UA_TYPES_ENUMDESCRIPTION]
                );
            }
            auto entry = std::make_unique<Entry>();
            const auto status = UA_DataType_fromDescription(
                &entry->type,
                &description,
                out.head()
            );
            if (status == UA_STATUSCODE_BADNOTFOUND) continue;
            done[i] = true;
            if (status != UA_STATUSCODE_GOOD) {
                VLOG(1) << "[opc] cannot define the data type "
                        << types::NodeId::to_string(seen[d.id].get()) << ": "
                        << UA_StatusCode_name(status);
                continue;
            }
            entry->array = {
                .next = const_cast<UA_DataTypeArray *>(out.head()),
                .typesSize = 1,
                .types = &entry->type,
                .cleanup = false,
            };
            out.entries.push_back(std::move(entry));
            progress = true;
        }
    }
    return {std::move(out), x::errors::NIL};
}

const UA_DataType *Types::find(const UA_NodeId &data_type) const {
    return UA_findDataTypeWithCustom(&data_type, this->head());
}

std::vector<Member> Types::members(const UA_DataType &type) {
    std::vector<Member> members;
    if (type.typeKind != UA_DATATYPEKIND_STRUCTURE &&
        type.typeKind != UA_DATATYPEKIND_OPTSTRUCT)
        return members;
    for (std::size_t i = 0; i < type.membersSize; ++i) {
        const auto &m = type.members[i];
        members.push_back({
            .name = m.memberName,
            .type = m.memberType,
            .is_array = static_cast<bool>(m.isArray),
        });
    }
    return members;
}

std::pair<const UA_DataType *, x::errors::Error>
Types::member_type(const UA_DataType &type, const Path &path) {
    const UA_DataType *current = &type;
    for (const auto &name: path) {
        auto [place, err] = find_member(*current, name);
        if (err) return {nullptr, err};
        current = place.member->memberType;
    }
    return {current, x::errors::NIL};
}

template<typename R, typename F>
std::pair<R, x::errors::Error>
Types::visit(const UA_Variant &value, const Path &path, const F &fn) const {
    if (value.type == nullptr || value.data == nullptr)
        return {R{}, invalid("the value is empty")};
    if (!UA_Variant_isScalar(&value))
        return {R{}, invalid("the value is an array of structures")};
    const void *data = value.data;
    const UA_DataType *type = value.type;
    Decoded decoded;
    if (type->typeKind == UA_DATATYPEKIND_EXTENSIONOBJECT) {
        const auto &eo = *static_cast<const UA_ExtensionObject *>(data);
        if (eo.encoding >= UA_EXTENSIONOBJECT_DECODED) {
            data = eo.content.decoded.data;
            type = eo.content.decoded.type;
        } else if (eo.encoding == UA_EXTENSIONOBJECT_ENCODED_BYTESTRING) {
            const UA_DataType *encoded = nullptr;
            for (const auto &entry: this->entries)
                if (UA_NodeId_equal(
                        &entry->type.binaryEncodingId,
                        &eo.content.encoded.typeId
                    ))
                    encoded = &entry->type;
            if (encoded == nullptr)
                return {
                    R{},
                    invalid("the server did not define the type of the value")
                };
            decoded.data = UA_new(encoded);
            if (decoded.data == nullptr) throw std::bad_alloc();
            decoded.type = encoded;
            UA_DecodeBinaryOptions options{};
            options.customTypes = this->head();
            if (const auto status = UA_decodeBinary(
                    &eo.content.encoded.body,
                    decoded.data,
                    encoded,
                    &options
                );
                status != UA_STATUSCODE_GOOD)
                return {
                    R{},
                    invalid(
                        std::string("the value does not decode: ") +
                        UA_StatusCode_name(status)
                    )
                };
            data = decoded.data;
            type = encoded;
        } else
            return {R{}, invalid("the value is encoded as XML")};
    }
    for (const auto &name: path) {
        auto [place, err] = find_member(*type, name);
        if (err) return {R{}, err};
        const auto *ptr = static_cast<const char *>(data) + place.offset;
        if (place.member->isOptional) {
            ptr = *reinterpret_cast<const char *const *>(ptr);
            if (ptr == nullptr) return {R{}, invalid("the value has no " + name)};
        }
        data = ptr;
        type = place.member->memberType;
    }
    UA_Variant member;
    UA_Variant_setScalar(&member, const_cast<void *>(data), type);
    return fn(member);
}

std::pair<x::telem::SampleValue, x::errors::Error> Types::sample(
    const UA_Variant &value,
    const Path &path,
    const x::telem::DataType &dt
) const {
    return this->visit<x::telem::SampleValue>(
        value,
        path,
        [&dt](const UA_Variant &member) { return telem::to_sample(member, dt); }
    );
}

std::pair<std::string, x::errors::Error>
Types::print(const UA_Variant &value, const Path &path) const {
    if (path.empty() && (value.type == nullptr || !UA_Variant_isScalar(&value) ||
                         value.type->typeKind != UA_DATATYPEKIND_EXTENSIONOBJECT))
        return json(value);
    return this->visit<std::string>(value, path, [](const UA_Variant &member) {
        return json(member);
    });
}
}
