// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/xmemory/local_shared.h"
#include "x/cpp/xtest/xtest.h"

using namespace xmemory;

TEST(LocalShared, DefaultConstruction) {
    local_shared<int> ptr;
    EXPECT_EQ(ptr.get(), nullptr);
    EXPECT_EQ(ptr.use_count(), 0);
    EXPECT_FALSE(ptr);
}

TEST(LocalShared, ValueConstruction) {
    auto ptr = make_local_shared<int>(42);
    EXPECT_NE(ptr.get(), nullptr);
    EXPECT_EQ(*ptr, 42);
    EXPECT_EQ(ptr.use_count(), 1);
    EXPECT_TRUE(ptr);
}

TEST(LocalShared, CopyConstruction) {
    auto ptr1 = make_local_shared<int>(42);
    EXPECT_EQ(ptr1.use_count(), 1);

    auto ptr2 = ptr1;
    EXPECT_EQ(ptr1.use_count(), 2);
    EXPECT_EQ(ptr2.use_count(), 2);
    EXPECT_EQ(*ptr1, 42);
    EXPECT_EQ(*ptr2, 42);
    EXPECT_EQ(ptr1, ptr2);
}

TEST(LocalShared, MoveConstruction) {
    auto ptr1 = make_local_shared<int>(42);
    EXPECT_EQ(ptr1.use_count(), 1);

    auto ptr2 = std::move(ptr1);
    EXPECT_EQ(ptr1.get(), nullptr);
    EXPECT_EQ(ptr1.use_count(), 0);
    EXPECT_EQ(ptr2.use_count(), 1);
    EXPECT_EQ(*ptr2, 42);
}

TEST(LocalShared, CopyAssignment) {
    auto ptr1 = make_local_shared<int>(42);
    auto ptr2 = make_local_shared<int>(100);

    EXPECT_EQ(ptr1.use_count(), 1);
    EXPECT_EQ(ptr2.use_count(), 1);

    ptr2 = ptr1;

    EXPECT_EQ(ptr1.use_count(), 2);
    EXPECT_EQ(ptr2.use_count(), 2);
    EXPECT_EQ(*ptr1, 42);
    EXPECT_EQ(*ptr2, 42);
}

TEST(LocalShared, MoveAssignment) {
    auto ptr1 = make_local_shared<int>(42);
    auto ptr2 = make_local_shared<int>(100);

    ptr2 = std::move(ptr1);

    EXPECT_EQ(ptr1.get(), nullptr);
    EXPECT_EQ(ptr2.use_count(), 1);
    EXPECT_EQ(*ptr2, 42);
}

TEST(LocalShared, Reset) {
    auto ptr = make_local_shared<int>(42);
    EXPECT_EQ(ptr.use_count(), 1);

    ptr.reset();

    EXPECT_EQ(ptr.get(), nullptr);
    EXPECT_EQ(ptr.use_count(), 0);
}

TEST(LocalShared, Swap) {
    auto ptr1 = make_local_shared<int>(42);
    auto ptr2 = make_local_shared<int>(100);

    ptr1.swap(ptr2);

    EXPECT_EQ(*ptr1, 100);
    EXPECT_EQ(*ptr2, 42);
}

TEST(LocalShared, MultipleReferences) {
    auto ptr1 = make_local_shared<int>(42);
    auto ptr2 = ptr1;
    auto ptr3 = ptr1;

    EXPECT_EQ(ptr1.use_count(), 3);
    EXPECT_EQ(ptr2.use_count(), 3);
    EXPECT_EQ(ptr3.use_count(), 3);

    ptr2.reset();
    EXPECT_EQ(ptr1.use_count(), 2);
    EXPECT_EQ(ptr3.use_count(), 2);

    ptr3.reset();
    EXPECT_EQ(ptr1.use_count(), 1);
}

TEST(LocalShared, StructWithMembers) {
    struct TestStruct {
        int x;
        double y;
        std::string z;

        TestStruct(int x_, double y_, std::string z_): x(x_), y(y_), z(std::move(z_)) {}
    };

    auto ptr = make_local_shared<TestStruct>(42, 3.14, "test");
    EXPECT_EQ(ptr->x, 42);
    EXPECT_EQ(ptr->y, 3.14);
    EXPECT_EQ(ptr->z, "test");

    auto ptr2 = ptr;
    EXPECT_EQ(ptr2->x, 42);
    EXPECT_EQ(ptr.use_count(), 2);
}

TEST(LocalShared, NullptrComparison) {
    local_shared<int> ptr;
    EXPECT_EQ(ptr, nullptr);
    EXPECT_FALSE(ptr != nullptr);

    ptr = make_local_shared<int>(42);
    EXPECT_NE(ptr, nullptr);
    EXPECT_FALSE(ptr == nullptr);
}

struct DestructorCounter {
    int *counter;

    explicit DestructorCounter(int *c): counter(c) {}

    ~DestructorCounter() {
        if (counter) ++(*counter);
    }
};

TEST(LocalShared, DestructorCalled) {
    int counter = 0;

    {
        auto ptr1 = make_local_shared<DestructorCounter>(&counter);
        EXPECT_EQ(counter, 0);

        {
            auto ptr2 = ptr1;
            EXPECT_EQ(counter, 0);
        }

        EXPECT_EQ(counter, 0); // Still one reference
    }

    EXPECT_EQ(counter, 1); // Now destroyed
}
