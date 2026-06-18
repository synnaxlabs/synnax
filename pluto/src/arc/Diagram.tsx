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

import { create } from "@/arc/Arc";
import { Stage } from "@/arc/functions";
import { useDispatch, useSelectNodeConfig } from "@/arc/queries";
import { parseEdgeKey } from "@/arc/translate";
import { Component } from "@/component";
import { Key } from "@/key";
import { type Diagram as BaseDiagram } from "@/vis/diagram";

// nodeChangesToActions converts diagram node gestures into Arc actions. Dimension
// changes are dropped: Arc graph nodes carry no measured field, so the diagram
// re-measures them on mount.
export const nodeChangesToActions = (
  changes: BaseDiagram.NodeChange[],
): arc.Action[] => {
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

// edgeChangesToActions converts diagram edge gestures into Arc actions. Removal
// recovers the endpoints from the diagram edge key, since Arc edges carry no key
// on the wire.
export const edgeChangesToActions = (changes: BaseDiagram.EdgeChange[]): arc.Action[] =>
  changes.flatMap((ch) => {
    switch (ch.type) {
      case "add":
        return [
          arc.addEdge({
            edge: {
              source: ch.edge.source,
              target: ch.edge.target,
              kind: arc.ir.EdgeKind.continuous,
            },
          }),
        ];
      case "remove":
        return [arc.removeEdge(parseEdgeKey(ch.key))];
      default:
        return [];
    }
  });

const NodeRenderer = ({
  nodeKey,
  position,
  selected,
  draggable,
}: BaseDiagram.NodeProps): ReactElement | null => {
  const key = Key.use<arc.Key>("Arc.Diagram.NodeRenderer");
  const config = useSelectNodeConfig({ key, nodeKey });
  const { dispatch } = useDispatch();
  const type = config?.type;
  const handleChange = useCallback(
    (params: record.Unknown) =>
      dispatch({
        key,
        actions: [arc.setNodeConfig({ key: nodeKey, config: { ...params, type } })],
      }),
    [key, nodeKey, dispatch, type],
  );
  if (config == null) return null;
  const C = Stage.REGISTRY[config.type];
  if (C == null) throw new Error(`Arc function ${config.type} not found`);
  const { type: _type, ...params } = config;
  return (
    <C.Symbol
      nodeKey={nodeKey}
      position={position}
      selected={selected}
      draggable={draggable}
      onChange={handleChange}
      {...params}
    />
  );
};

export const Diagram = create({ node: Component.renderProp(NodeRenderer) });
