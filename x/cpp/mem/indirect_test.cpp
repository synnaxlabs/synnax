// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "x/cpp/mem/indirect.h"
#include <gtest/gtest.h>
#include <string>

namespace {

struct Point {
    int x;
    int y;
};

// Forward declaration to test incomplete type support
struct Node;

struct Node {
    int value;
    x::mem::indirect<Node> left;
    x::mem::indirect<Node> right;
};

TEST(IndirectTest, DefaultConstruction) {
    x::mem::indirect<int> i;
    EXPECT_FALSE(i);
    EXPECT_FALSE(i.has_value());
    EXPECT_EQ(i, nullptr);
}

TEST(IndirectTest, NullptrConstruction) {
    x::mem::indirect<int> i(nullptr);
    EXPECT_FALSE(i);
    EXPECT_EQ(i.get(), nullptr);
}

TEST(IndirectTest, ValueConstruction) {
    x::mem::indirect<int> i(42);
    EXPECT_TRUE(i);
    EXPECT_TRUE(i.has_value());
    EXPECT_EQ(*i, 42);
    EXPECT_NE(i, nullptr);
}

TEST(IndirectTest, StructConstruction) {
    x::mem::indirect<Point> p(Point{10, 20});
    EXPECT_TRUE(p);
    EXPECT_EQ(p->x, 10);
    EXPECT_EQ(p->y, 20);
}

TEST(IndirectTest, CopyConstruction) {
    x::mem::indirect<int> a(42);
    x::mem::indirect<int> b(a);

    EXPECT_TRUE(b);
    EXPECT_EQ(*b, 42);

    // Verify deep copy - modifying one doesn't affect the other
    *a = 100;
    EXPECT_EQ(*a, 100);
    EXPECT_EQ(*b, 42);
}

TEST(IndirectTest, CopyConstructionEmpty) {
    x::mem::indirect<int> a;
    x::mem::indirect<int> b(a);

    EXPECT_FALSE(b);
}

TEST(IndirectTest, MoveConstruction) {
    x::mem::indirect<int> a(42);
    x::mem::indirect<int> b(std::move(a));

    EXPECT_TRUE(b);
    EXPECT_EQ(*b, 42);
    EXPECT_FALSE(a); // moved-from state
}

TEST(IndirectTest, CopyAssignment) {
    x::mem::indirect<int> a(42);
    x::mem::indirect<int> b;

    b = a;

    EXPECT_TRUE(b);
    EXPECT_EQ(*b, 42);

    // Verify deep copy
    *a = 100;
    EXPECT_EQ(*b, 42);
}

TEST(IndirectTest, MoveAssignment) {
    x::mem::indirect<int> a(42);
    x::mem::indirect<int> b;

    b = std::move(a);

    EXPECT_TRUE(b);
    EXPECT_EQ(*b, 42);
}

TEST(IndirectTest, NullptrAssignment) {
    x::mem::indirect<int> i(42);
    EXPECT_TRUE(i);

    i = nullptr;
    EXPECT_FALSE(i);
}

TEST(IndirectTest, ValueAssignment) {
    x::mem::indirect<int> i;
    i = 42;

    EXPECT_TRUE(i);
    EXPECT_EQ(*i, 42);
}

TEST(IndirectTest, ValueOr) {
    x::mem::indirect<int> empty;
    x::mem::indirect<int> present(42);

    EXPECT_EQ(empty.value_or(100), 100);
    EXPECT_EQ(present.value_or(100), 42);
}

TEST(IndirectTest, Reset) {
    x::mem::indirect<int> i(42);
    EXPECT_TRUE(i);

    i.reset();
    EXPECT_FALSE(i);
}

TEST(IndirectTest, Swap) {
    x::mem::indirect<int> a(42);
    x::mem::indirect<int> b(100);

    a.swap(b);

    EXPECT_EQ(*a, 100);
    EXPECT_EQ(*b, 42);
}

TEST(IndirectTest, SelfReferentialStruct) {
    Node root;
    root.value = 1;
    root.left = Node{2, nullptr, nullptr};
    root.right = Node{3, nullptr, nullptr};

    EXPECT_EQ(root.value, 1);
    EXPECT_TRUE(root.left);
    EXPECT_EQ(root.left->value, 2);
    EXPECT_TRUE(root.right);
    EXPECT_EQ(root.right->value, 3);
    EXPECT_FALSE(root.left->left);
    EXPECT_FALSE(root.left->right);
}

TEST(IndirectTest, DeepCopyRecursive) {
    Node original;
    original.value = 1;
    original.left = Node{2, nullptr, nullptr};

    Node copy = original;

    EXPECT_EQ(copy.value, 1);
    EXPECT_TRUE(copy.left);
    EXPECT_EQ(copy.left->value, 2);

    // Modify original, copy should be unaffected
    original.left->value = 100;
    EXPECT_EQ(copy.left->value, 2);
}

TEST(IndirectTest, MakeIndirect) {
    auto i = x::mem::make_indirect<Point>(10, 20);
    EXPECT_TRUE(i);
    EXPECT_EQ(i->x, 10);
    EXPECT_EQ(i->y, 20);
}

TEST(IndirectTest, StringType) {
    x::mem::indirect<std::string> s(std::string("hello"));
    EXPECT_TRUE(s);
    EXPECT_EQ(*s, "hello");

    auto copy = s;
    EXPECT_EQ(*copy, "hello");

    *s = "world";
    EXPECT_EQ(*s, "world");
    EXPECT_EQ(*copy, "hello");
}

} // namespace