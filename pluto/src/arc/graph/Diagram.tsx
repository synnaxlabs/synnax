// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { type record } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { Node } from "@/arc/graph/node";
import { useSelectNodeConfig, useSingleDispatch } from "@/arc/queries";
import { Component } from "@/component";
import { Diagram as Base } from "@/vis/diagram";

// nodeChangesToActions converts diagram node gestures into Arc actions. Dimension
// changes are dropped: Arc graph nodes carry no measured field, so the diagram
// re-measures them on mount.
export const nodeChangesToActions = (changes: Base.NodeChange[]): arc.Action[] => {
  const actions: arc.Action[] = [];
  changes.forEach((ch) => {
    switch (ch.type) {
      case "position":
        actions.push(arc.setNodePosition({ key: ch.key, position: ch.position }));
        return;
      case "remove":
        actions.push(arc.removeNode({ key: ch.key }));
    }
  });
  return actions;
};

// edgeChangesToActions converts diagram edge gestures into Arc actions. The
// diagram edge key is the edge's stable identity, carried through to the graph.
export const edgeChangesToActions = (changes: Base.EdgeChange[]): arc.Action[] =>
  changes.flatMap((ch) => {
    switch (ch.type) {
      case "add":
        return [
          arc.addEdge({
            edge: {
              key: ch.edge.key,
              source: ch.edge.source,
              target: ch.edge.target,
              kind: arc.ir.EdgeKind.continuous,
            },
          }),
        ];
      case "remove":
        return [arc.removeEdge({ key: ch.key })];
      case "reconnect":
        return [
          arc.reconnectEdge({ key: ch.key, source: ch.source, target: ch.target }),
        ];
      default:
        return [];
    }
  });

const NodeRenderer = ({
  nodeKey,
  position,
  selected,
  draggable,
}: Base.NodeProps): ReactElement | null => {
  const config = useSelectNodeConfig({ nodeKey });
  const dispatch = useSingleDispatch();
  const handleChange = useCallback(
    (config: Partial<record.Unknown>) =>
      dispatch(arc.setNodeConfig({ key: nodeKey, config })),
    [nodeKey, dispatch],
  );
  if (config == null) return null;
  const Render = Node.resolveSpec(config.type).Symbol;
  return (
    <Render
      nodeKey={nodeKey}
      position={position}
      selected={selected}
      draggable={draggable}
      config={config}
      onConfigChange={handleChange}
    />
  );
};

export const Diagram = Base.create({ node: Component.renderProp(NodeRenderer) });
