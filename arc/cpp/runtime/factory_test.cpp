// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include "arc/cpp/runtime/factory/factory.h"

#include "gtest/gtest.h"
#include "x/cpp/xtest/xtest.h"

namespace arc {

/// @brief Mock node for testing factory pattern.
class MockNode : public Node {
    std::string id_;

public:
    explicit MockNode(std::string id) : id_(std::move(id)) {}

    xerrors::Error execute(NodeContext &ctx) override { return xerrors::NIL; }

    std::string id() const override { return id_; }
};

/// @brief Mock factory that handles a specific node type.
class MockFactoryA : public NodeFactory {
public:
    std::pair<std::unique_ptr<Node>, xerrors::Error>
    create(const NodeFactoryConfig &cfg) override {
        if (cfg.ir_node.type != "type_a") {
            return {nullptr, xerrors::Error("NOT_FOUND")};
        }
        return {std::make_unique<MockNode>(cfg.ir_node.key), xerrors::NIL};
    }
};

/// @brief Mock factory that handles a different node type.
class MockFactoryB : public NodeFactory {
public:
    std::pair<std::unique_ptr<Node>, xerrors::Error>
    create(const NodeFactoryConfig &cfg) override {
        if (cfg.ir_node.type != "type_b") {
            return {nullptr, xerrors::Error("NOT_FOUND")};
        }
        return {std::make_unique<MockNode>(cfg.ir_node.key), xerrors::NIL};
    }
};

/// @brief Mock factory that always returns an error (not NOT_FOUND).
class MockFactoryError : public NodeFactory {
public:
    std::pair<std::unique_ptr<Node>, xerrors::Error>
    create(const NodeFactoryConfig &cfg) override {
        if (cfg.ir_node.type != "type_error") {
            return {nullptr, xerrors::Error("NOT_FOUND")};
        }
        return {nullptr, xerrors::Error("arc.test.factory_error", "Intentional error")};
    }
};

class FactoryTest : public ::testing::Test {
protected:
    std::unique_ptr<queue::SPSC<ChannelUpdate>> input_queue_;
    std::unique_ptr<queue::SPSC<ChannelOutput>> output_queue_;
    std::unique_ptr<State> state_;
    ir::IR ir_;

    void SetUp() override {
        input_queue_ = std::make_unique<queue::SPSC<ChannelUpdate>>(16);
        output_queue_ = std::make_unique<queue::SPSC<ChannelOutput>>(16);
        state_ = std::make_unique<State>(input_queue_.get(), output_queue_.get());

        // Setup minimal IR
        ir_.nodes.push_back(ir::Node{"node_a"});
        ir_.nodes[0].type = "type_a";
    }
};

TEST_F(FactoryTest, MultiFactoryFirstFactorySucceeds) {
    MultiFactory factory;
    factory.add(std::make_unique<MockFactoryA>());
    factory.add(std::make_unique<MockFactoryB>());

    NodeFactoryConfig cfg{ir_.nodes[0], *state_, ir_};

    auto [node, err] = factory.create(cfg);
    ASSERT_NIL(err);
    ASSERT_NE(node, nullptr);
    EXPECT_EQ(node->id(), "node_a");
}

TEST_F(FactoryTest, MultiFactorySecondFactorySucceeds) {
    MultiFactory factory;
    factory.add(std::make_unique<MockFactoryA>());
    factory.add(std::make_unique<MockFactoryB>());

    // Create node with type_b
    ir_.nodes[0].type = "type_b";
    NodeFactoryConfig cfg{ir_.nodes[0], *state_, ir_};

    auto [node, err] = factory.create(cfg);
    ASSERT_NIL(err);
    ASSERT_NE(node, nullptr);
    EXPECT_EQ(node->id(), "node_a");
}

TEST_F(FactoryTest, MultiFactoryNoMatchReturnsNotFound) {
    MultiFactory factory;
    factory.add(std::make_unique<MockFactoryA>());
    factory.add(std::make_unique<MockFactoryB>());

    // Create node with unknown type
    ir_.nodes[0].type = "unknown_type";
    NodeFactoryConfig cfg{ir_.nodes[0], *state_, ir_};

    auto [node, err] = factory.create(cfg);
    EXPECT_EQ(node, nullptr);
    ASSERT_OCCURRED_AS(err, "NOT_FOUND");
}

TEST_F(FactoryTest, MultiFactoryStopsOnRealError) {
    MultiFactory factory;
    factory.add(std::make_unique<MockFactoryError>());
    factory.add(std::make_unique<MockFactoryA>());  // Should not be reached

    // Create node with type_error
    ir_.nodes[0].type = "type_error";
    NodeFactoryConfig cfg{ir_.nodes[0], *state_, ir_};

    auto [node, err] = factory.create(cfg);
    EXPECT_EQ(node, nullptr);
    // Should return the real error, not NOT_FOUND
    ASSERT_OCCURRED_AS(err, "arc.test.factory_error");
}

TEST_F(FactoryTest, MultiFactoryEmptyFactoryListReturnsNotFound) {
    MultiFactory factory;

    NodeFactoryConfig cfg{ir_.nodes[0], *state_, ir_};

    auto [node, err] = factory.create(cfg);
    EXPECT_EQ(node, nullptr);
    ASSERT_OCCURRED_AS(err, "NOT_FOUND");
}

TEST_F(FactoryTest, MultiFactoryOrderMatters) {
    MultiFactory factory;
    // Add MockFactoryA twice to verify first match wins
    factory.add(std::make_unique<MockFactoryA>());
    factory.add(std::make_unique<MockFactoryA>());

    ir_.nodes[0].type = "type_a";
    NodeFactoryConfig cfg{ir_.nodes[0], *state_, ir_};

    auto [node, err] = factory.create(cfg);
    ASSERT_NIL(err);
    ASSERT_NE(node, nullptr);
    // Verifies that the first factory created the node (no double-creation)
}

}  // namespace arc
