#pragma once

#include "x/cpp/telem/telem.h"
#include "x/cpp/xjson/xjson.h"

#include "arc/go/types/arc/go/types/types.pb.h"

namespace arc::types {
using ChannelKey = std::uint32_t;

enum class Kind : uint8_t {
    Invalid = 0,
    U8 = 1,
    U16 = 2,
    U32 = 3,
    U64 = 4,
    I8 = 5,
    I16 = 6,
    I32 = 7,
    I64 = 8,
    F32 = 9,
    F64 = 10,
    String = 11,
    TimeStamp = 12,
    TimeSpan = 13,
    Chan = 14,
    Series = 15,
};

struct Type {
    Kind kind = Kind::Invalid;
    std::unique_ptr<Type> elem;

    explicit Type(xjson::Parser parser) {
        this->kind = parser.field<Kind>("kind");
        const auto elem_parser = parser.optional_child("elem");
        if (elem_parser.ok()) this->elem = std::make_unique<Type>(elem_parser);
    }

    [[nodiscard]] nlohmann::json to_json() const {
        nlohmann::json j;
        j["kind"] = static_cast<uint8_t>(kind);
        if (elem) j["elem"] = elem->to_json();
        return j;
    }

    explicit Type(const arc::v1::types::PBType &pb) {
        this->kind = static_cast<Kind>(pb.kind());
        if (pb.has_elem()) this->elem = std::make_unique<Type>(pb.elem());
    }

    void to_proto(arc::v1::types::PBType *pb) const {
        pb->set_kind(static_cast<arc::v1::types::PBKind>(kind));
        if (elem) elem->to_proto(pb->mutable_elem());
    }

    Type() = default;
    explicit Type(const Kind k): kind(k) {}
    Type(const Kind k, Type elem_type):
        kind(k), elem(std::make_unique<Type>(std::move(elem_type))) {}

    Type(const Type &other): kind(other.kind) {
        if (other.elem) { elem = std::make_unique<Type>(*other.elem); }
    }

    Type &operator=(const Type &other) {
        if (this != &other) {
            kind = other.kind;
            if (other.elem)
                elem = std::make_unique<Type>(*other.elem);
            else
                elem.reset();
        }
        return *this;
    }

    Type(Type &&) = default;
    Type &operator=(Type &&) = default;

    [[nodiscard]] size_t density() const {
        switch (kind) {
            case Kind::U8:
            case Kind::I8:
                return 1;
            case Kind::U16:
            case Kind::I16:
                return 2;
            case Kind::U32:
            case Kind::I32:
            case Kind::F32:
                return 4;
            case Kind::U64:
            case Kind::I64:
            case Kind::F64:
            case Kind::TimeStamp:
            case Kind::TimeSpan:
                return 8;
            default:
                return 0;
        }
    }

    [[nodiscard]] bool is_valid() const { return kind != Kind::Invalid; }

    [[nodiscard]] telem::DataType telem() const;
};
}
