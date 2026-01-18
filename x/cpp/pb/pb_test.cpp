// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/pb/pb.h"
#include "x/cpp/test/test.h"

namespace x::pb {

struct MockProto {
    int value;
    bool should_fail;
};

struct MockElement {
    int value;

    static std::pair<MockElement, errors::Error>
    from_proto(const MockProto &pb) {
        if (pb.should_fail)
            return {{}, errors::Error("test.error", "mock conversion failed")};
        return {MockElement{pb.value}, errors::NIL};
    }
};

TEST(testpb, testFromProtoRepeatedEmptyContainer) {
    std::vector<MockProto> src;
    std::vector<MockElement> dst;
    auto err = from_proto_repeated<MockElement>(dst, src);
    ASSERT_NIL(err);
    ASSERT_EQ(dst.size(), 0);
}

TEST(testpb, testFromProtoRepeatedSingleItem) {
    std::vector<MockProto> src = {{42, false}};
    std::vector<MockElement> dst;
    auto err = from_proto_repeated<MockElement>(dst, src);
    ASSERT_NIL(err);
    ASSERT_EQ(dst.size(), 1);
    ASSERT_EQ(dst[0].value, 42);
}

TEST(testpb, testFromProtoRepeatedMultipleItems) {
    std::vector<MockProto> src = {{1, false}, {2, false}, {3, false}};
    std::vector<MockElement> dst;
    auto err = from_proto_repeated<MockElement>(dst, src);
    ASSERT_NIL(err);
    ASSERT_EQ(dst.size(), 3);
    ASSERT_EQ(dst[0].value, 1);
    ASSERT_EQ(dst[1].value, 2);
    ASSERT_EQ(dst[2].value, 3);
}

TEST(testpb, testFromProtoRepeatedErrorOnFirstItem) {
    std::vector<MockProto> src = {{1, true}, {2, false}, {3, false}};
    std::vector<MockElement> dst;
    auto err = from_proto_repeated<MockElement>(dst, src);
    ASSERT_TRUE(err);
    ASSERT_MATCHES(err, errors::Error("test.error", ""));
    ASSERT_EQ(dst.size(), 0);
}

TEST(testpb, testFromProtoRepeatedErrorOnMiddleItem) {
    std::vector<MockProto> src = {{1, false}, {2, true}, {3, false}};
    std::vector<MockElement> dst;
    auto err = from_proto_repeated<MockElement>(dst, src);
    ASSERT_TRUE(err);
    ASSERT_MATCHES(err, errors::Error("test.error", ""));
    ASSERT_EQ(dst.size(), 1);
    ASSERT_EQ(dst[0].value, 1);
}

}
