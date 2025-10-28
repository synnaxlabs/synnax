// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <string>
#include <vector>

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/scheduler.h"
#include "arc/cpp/runtime/state.h"
#include "x/cpp/xerrors/errors.h"

namespace arc {

// Forward declaration
class Runtime;

/// @brief Configuration for node factory creation.
///
/// Provides dependencies needed by ALL factories for constructing nodes.
/// Factory-specific dependencies (e.g., WASM runtime) are passed to factory constructors.
struct NodeFactoryConfig {
    const ir::Node& ir_node;  ///< IR definition for this node
    State& state;             ///< Runtime state reference
    const ir::IR& ir;         ///< Full IR for context lookups
};

/// @brief Factory interface for creating nodes from IR definitions.
///
/// Implements the Chain of Responsibility pattern:
/// - Return {node, NIL} if the factory can handle the node type
/// - Return {nullptr, NOT_FOUND} if the factory cannot handle the type
/// - Return {nullptr, error} for real errors during node construction
///
/// This pattern allows composing multiple factories via MultiFactory without
/// modifying existing code (Open/Closed Principle).
///
/// Example implementation:
/// @code
/// class MyNodeFactory : public NodeFactory {
///     std::pair<std::unique_ptr<Node>, xerrors::Error>
///     create(const NodeFactoryConfig& cfg) override {
///         if (cfg.ir_node.type != "my_type") {
///             return {nullptr, xerrors::Error("NOT_FOUND")};
///         }
///         auto node = std::make_unique<MyNode>(...);
///         return {std::move(node), xerrors::NIL};
///     }
/// };
/// @endcode
class NodeFactory {
public:
    virtual ~NodeFactory() = default;

    /// @brief Create a node from IR definition.
    ///
    /// @param cfg Configuration with IR node and dependencies.
    /// @return Pair of {node, error}:
    ///         - {node, NIL} on success
    ///         - {nullptr, NOT_FOUND} if this factory cannot handle the type
    ///         - {nullptr, error} on construction failure
    virtual std::pair<std::unique_ptr<Node>, xerrors::Error>
    create(const NodeFactoryConfig& cfg) = 0;
};

/// @brief Composite factory that tries multiple factories in sequence.
///
/// Implements Chain of Responsibility pattern by delegating to child factories
/// until one succeeds. Stops on first non-NOT_FOUND error.
///
/// Usage:
/// @code
/// MultiFactory factory;
/// factory.add(std::make_unique<IntervalNodeFactory>());
/// factory.add(std::make_unique<WASMNodeFactory>());
///
/// auto [node, err] = factory.create(cfg);
/// if (err) {
///     // No factory could handle this node type
/// }
/// @endcode
class MultiFactory : public NodeFactory {
    std::vector<std::unique_ptr<NodeFactory>> factories_;

public:
    MultiFactory() = default;

    /// @brief Add a factory to the chain.
    ///
    /// Factories are tried in the order they are added. Place more specific
    /// factories before more general ones (e.g., IntervalFactory before WASMFactory).
    ///
    /// @param factory Factory to add (ownership transferred).
    void add(std::unique_ptr<NodeFactory> factory);

    /// @brief Try each factory until one succeeds.
    ///
    /// Iterates through factories in order:
    /// - If factory returns {node, NIL}: return immediately with success
    /// - If factory returns {nullptr, NOT_FOUND}: try next factory
    /// - If factory returns {nullptr, other_error}: return error immediately
    /// - If no factory succeeds: return NOT_FOUND error
    ///
    /// @param cfg Configuration for node creation.
    /// @return Pair of {node, error} (see create() documentation).
    std::pair<std::unique_ptr<Node>, xerrors::Error>
    create(const NodeFactoryConfig& cfg) override;
};

}  // namespace arc
