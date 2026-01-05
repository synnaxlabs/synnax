// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "arc/cpp/types/types.h"
#include "arc/go/types/arc/go/types/types.pb.h"

/// @brief it should correctly round-trip a simple Type through protobuf
TEST(TypesTest, testTypeProtobufRoundTrip) {
    arc::types::Type original(arc::types::Kind::F32);

    arc::v1::types::PBType pb;
    original.to_proto(&pb);

    arc::types::Type reconstructed(pb);

    ASSERT_EQ(reconstructed.kind, arc::types::Kind::F32);
    ASSERT_EQ(reconstructed.elem, nullptr);
}

/// @brief it should correctly round-trip a Type with elem through protobuf
TEST(TypesTest, testTypeWithElemProtobufRoundTrip) {
    arc::types::Type elem_type(arc::types::Kind::U64);
    arc::types::Type original(arc::types::Kind::Series, std::move(elem_type));

    arc::v1::types::PBType pb;
    original.to_proto(&pb);

    arc::types::Type reconstructed(pb);

    ASSERT_EQ(reconstructed.kind, arc::types::Kind::Series);
    ASSERT_NE(reconstructed.elem, nullptr);
    ASSERT_EQ(reconstructed.elem->kind, arc::types::Kind::U64);
}

/// @brief it should correctly convert all Kind enum values
TEST(TypesTest, testAllKindValues) {
    const arc::types::Kind kinds[] = {
        arc::types::Kind::Invalid,
        arc::types::Kind::U8,
        arc::types::Kind::U16,
        arc::types::Kind::U32,
        arc::types::Kind::U64,
        arc::types::Kind::I8,
        arc::types::Kind::I16,
        arc::types::Kind::I32,
        arc::types::Kind::I64,
        arc::types::Kind::F32,
        arc::types::Kind::F64,
        arc::types::Kind::String,
        arc::types::Kind::Chan,
        arc::types::Kind::Series,
    };

    for (const auto kind: kinds) {
        arc::types::Type original(kind);
        arc::v1::types::PBType pb;
        original.to_proto(&pb);
        arc::types::Type reconstructed(pb);
        ASSERT_EQ(reconstructed.kind, kind);
    }
}

/// @brief it should correctly round-trip a Type with unit through protobuf
TEST(TypesTest, testTypeWithUnitProtobufRoundTrip) {
    arc::types::Dimensions dims;
    dims.time = 1;
    arc::types::Unit unit(dims, 1.0, "ns");
    arc::types::Type original(arc::types::Kind::I64, std::move(unit));

    arc::v1::types::PBType pb;
    original.to_proto(&pb);

    arc::types::Type reconstructed(pb);

    ASSERT_EQ(reconstructed.kind, arc::types::Kind::I64);
    ASSERT_NE(reconstructed.unit, nullptr);
    ASSERT_EQ(reconstructed.unit->name, "ns");
    ASSERT_EQ(reconstructed.unit->scale, 1.0);
    ASSERT_EQ(reconstructed.unit->dimensions.time, 1);
    ASSERT_TRUE(reconstructed.is_timestamp());
}
