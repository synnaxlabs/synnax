// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, dimensions, location, scale, xy } from "@synnaxlabs/x";
import { type ReactFlowInstance } from "@xyflow/react";

import { type Diagram } from ".";

export const selectNode = (key: string): HTMLDivElement => {
  const el = document.querySelector(`[data-id="${key}"]`);
  if (el == null) throw new Error(`[diagram] - cannot find node with key: ${key}`);
  return el as HTMLDivElement;
};

export const selectNodeBox = (flow: ReactFlowInstance, key: string): box.Box => {
  const n = selectNode(key);
  const flowN = flow.getNodes().find((n) => n.id === key);
  if (flowN == null) throw new Error(`[diagram] - cannot find node with key: ${key}`);
  return box.construct(
    flowN.position,
    dimensions.scale(box.dims(box.construct(n)), 1 / flow.getZoom()),
  );
};

export const selectNodeLayout = (key: string, flow: ReactFlowInstance): NodeLayout =>
  NodeLayout.fromFlow(key, flow);

export class HandleLayout {
  node_: NodeLayout | null = null;
  position: xy.XY;
  orientation: location.Outer;

  constructor(position: xy.XY, orientation: location.Outer) {
    this.position = position;
    this.orientation = orientation;
  }

  set node(node: NodeLayout) {
    this.node_ = node;
  }

  get node(): NodeLayout {
    if (this.node_ == null) throw new Error(`[schematic] - handle has no node`);
    return this.node_;
  }

  get absolutePosition(): xy.XY {
    return xy.translate(box.topLeft(this.node.box), this.position);
  }
}

export class NodeLayout {
  key: string;
  box: box.Box;
  handles: HandleLayout[];

  constructor(key: string, box: box.Box, handles: HandleLayout[]) {
    this.key = key;
    this.box = box;
    this.handles = handles;
    handles.forEach((h) => (h.node = this));
  }

  static fromFlow(key: string, flow: ReactFlowInstance): NodeLayout {
    const nodeBox = selectNodeBox(flow, key);
    // grab all child elements with the class 'react-flow__handle'
    const nodeEl = selectNode(key);
    const handleEls = nodeEl.getElementsByClassName("react-flow__handle");
    const nodeElBox = box.construct(nodeEl);
    const handles = Array.from(handleEls).map((el) => {
      const pos = box.center(box.construct(el));
      const dist = xy.translation(box.topLeft(nodeElBox), pos);
      const match = el.className.match(/react-flow__handle-(\w+)/);
      if (match == null)
        throw new Error(`[schematic] - cannot find handle orientation`);
      const orientation = location.construct(match[0]) as location.Outer;
      return new HandleLayout(dist, orientation);
    });
    return new NodeLayout(key, nodeBox, handles);
  }
}

export const calculateCursorPosition = (
  region: box.Box,
  cursor: xy.Crude,
  viewport: Diagram.Viewport,
): xy.XY => {
  const zoomXY = xy.construct(viewport.zoom);
  const s = scale.XY.translate(xy.scale(box.topLeft(region), -1))
    .translate(xy.scale(viewport.position, -1))
    .magnify(xy.reciprocal(zoomXY));
  return s.pos(xy.construct(cursor));
};
