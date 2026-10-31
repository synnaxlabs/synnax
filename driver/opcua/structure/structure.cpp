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
        if (const auto raw = seen.resolve({}).value(); UA_NodeId_equal(&raw, &id))
            return true;
    return false;
}

/// @brief the definition of a type on the server, borrowed from a read response.
struct Definition {
    std::size_t id;
    const UA_Variant *definition;
    const UA_QualifiedName *name;
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

std::pair<Types, x::errors::Error> Types::read(
    const connection::Pool::Connection &conn,
    const std::vector<UA_NodeId> &data_types
) {
    Types out;
    // The server's own IDs name no namespace URI, so each resolves to itself.
    std::vector<types::NodeId> seen;
    const auto seen_id = [&seen](const std::size_t i) {
        return seen[i].resolve({}).value();
    };
    std::vector<std::size_t> round;
    bool capped = false;
    const auto visit = [&](const UA_NodeId &id) {
        if (UA_findDataType(&id) != nullptr || contains(seen, id)) return true;
        if (seen.size() >= MAX_TYPES) {
            capped = true;
            return false;
        }
        seen.emplace_back(id);
        round.push_back(seen.size() - 1);
        return true;
    };
    for (const auto &id: data_types)
        if (!visit(id))
            out.failures.emplace_back(
                types::NodeId(id),
                "the server defines more than " + std::to_string(MAX_TYPES) + " types"
            );

    std::vector<types::ReadResponse> responses;
    std::vector<Definition> definitions;
    for (int depth = 0; !round.empty(); ++depth) {
        if (depth == MAX_DEPTH) {
            capped = true;
            break;
        }
        const auto current = std::exchange(round, {});
        types::ReadRequestBuilder req;
        for (const auto idx: current) {
            req.add_node(seen_id(idx), UA_ATTRIBUTEID_DATATYPEDEFINITION);
            req.add_node(seen_id(idx), UA_ATTRIBUTEID_BROWSENAME);
        }
        auto [res, err] = conn.read(req.build());
        if (err) return {std::move(out), err};
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

    const auto fail = [&](const Definition &d, const std::string &reason) {
        const auto &name = d.name->name;
        out.failures.emplace_back(
            types::NodeId(seen_id(d.id)),
            "cannot decode " +
                std::string(reinterpret_cast<const char *>(name.data), name.length) +
                ": " + reason
        );
    };
    // A type builds once the types of its members exist, so each pass builds the types
    // whose members the earlier passes built.
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
                structure.dataTypeId = seen_id(d.id);
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
                enumeration.dataTypeId = seen_id(d.id);
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
                fail(d, UA_StatusCode_name(status));
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
    for (std::size_t i = 0; i < definitions.size(); ++i)
        if (!done[i])
            fail(
                definitions[i],
                capped ? "its members use more than " + std::to_string(MAX_TYPES) +
                             " types or nest deeper than " + std::to_string(MAX_DEPTH) +
                             " levels"
                       : "the server did not define the type of a member"
            );
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

std::pair<Field, x::errors::Error>
Types::field(const UA_NodeId &data_type, const Path &path) const {
    Field out;
    out.root = this->find(data_type);
    if (out.root == nullptr) {
        for (const auto &[id, reason]: this->failures)
            if (const auto raw = id.resolve({}).value();
                UA_NodeId_equal(&raw, &data_type))
                return {{}, invalid(reason)};
        return {
            {},
            x::errors::Error(
                UNDEFINED,
                "the server gives no definition of the data type of the node"
            )
        };
    }
    out.member = out.root;
    for (const auto &name: path) {
        auto [place, err] = find_member(*out.member, name);
        if (err) return {{}, err};
        out.steps.push_back({
            .offset = place.offset,
            .optional = static_cast<bool>(place.member->isOptional),
            .name = name,
        });
        out.member = place.member->memberType;
    }
    return {std::move(out), x::errors::NIL};
}

std::pair<types::Variant, x::errors::Error>
Types::decode(const UA_Variant &value) const {
    types::Variant out;
    if (value.type == nullptr || value.data == nullptr)
        return {std::move(out), invalid("the value is empty")};
    if (!UA_Variant_isScalar(&value))
        return {std::move(out), invalid("the value is an array of structures")};
    void *data = value.data;
    const UA_DataType *type = value.type;
    if (type->typeKind == UA_DATATYPEKIND_EXTENSIONOBJECT) {
        const auto &eo = *static_cast<const UA_ExtensionObject *>(data);
        if (eo.encoding == UA_EXTENSIONOBJECT_ENCODED_XML)
            return {std::move(out), invalid("the value is encoded as XML")};
        if (eo.encoding >= UA_EXTENSIONOBJECT_DECODED) {
            data = eo.content.decoded.data;
            type = eo.content.decoded.type;
        } else {
            const UA_DataType *encoded = nullptr;
            for (const auto &entry: this->entries)
                if (UA_NodeId_equal(
                        &entry->type.binaryEncodingId,
                        &eo.content.encoded.typeId
                    ))
                    encoded = &entry->type;
            if (encoded == nullptr)
                return {
                    std::move(out),
                    invalid("the server did not define the type of the value")
                };
            void *decoded = UA_new(encoded);
            if (decoded == nullptr) throw std::bad_alloc();
            UA_Variant_setScalar(out.ptr(), decoded, encoded);
            UA_DecodeBinaryOptions options{};
            options.customTypes = this->head();
            if (const auto status = UA_decodeBinary(
                    &eo.content.encoded.body,
                    decoded,
                    encoded,
                    &options
                );
                status != UA_STATUSCODE_GOOD)
                return {
                    types::Variant(),
                    invalid(
                        std::string("the value does not decode: ") +
                        UA_StatusCode_name(status)
                    )
                };
            return {std::move(out), x::errors::NIL};
        }
    }
    UA_Variant_setScalar(out.ptr(), data, type);
    out.ptr()->storageType = UA_VARIANT_DATA_NODELETE;
    return {std::move(out), x::errors::NIL};
}

std::pair<UA_Variant, x::errors::Error>
Types::member(const UA_Variant &structure, const Field &field) {
    UA_Variant member{};
    if (!UA_NodeId_equal(&structure.type->typeId, &field.root->typeId))
        return {
            member,
            invalid(
                std::string("the value is a ") + structure.type->typeName + ", not a " +
                field.root->typeName
            )
        };
    const auto *data = static_cast<const char *>(structure.data);
    for (const auto &step: field.steps) {
        data += step.offset;
        if (step.optional) {
            data = *reinterpret_cast<const char *const *>(data);
            if (data == nullptr)
                return {member, invalid("the value has no " + step.name)};
        }
    }
    UA_Variant_setScalar(&member, const_cast<char *>(data), field.member);
    return {member, x::errors::NIL};
}

std::pair<x::telem::SampleValue, x::errors::Error> Types::sample(
    const UA_Variant &structure,
    const Field &field,
    const x::telem::DataType &dt
) {
    const auto [member, err] = Types::member(structure, field);
    if (err) return {{}, err};
    return telem::to_sample(member, dt);
}

std::pair<std::string, x::errors::Error>
Types::print(const UA_Variant &value, const Path &path) const {
    if (path.empty() && (value.type == nullptr || !UA_Variant_isScalar(&value) ||
                         value.type->typeKind != UA_DATATYPEKIND_EXTENSIONOBJECT))
        return json(value);
    const auto [structure, err] = this->decode(value);
    if (err) return {"", err};
    if (path.empty()) return json(structure.get());
    const auto [field, field_err] = this->field(structure.get().type->typeId, path);
    if (field_err) return {"", field_err};
    const auto [member, member_err] = Types::member(structure.get(), field);
    if (member_err) return {"", member_err};
    return json(member);
}
}
