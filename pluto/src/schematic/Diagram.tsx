// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { useStoreApi, useUpdateNodeInternals } from "@xyflow/react";
import {
  type PropsWithChildren,
  type ReactElement,
  useEffect,
  useRef,
  useState,
} from "react";

import { Edge } from "@/schematic/edge";
import { Node } from "@/schematic/node";
import {
  useAllEdges,
  useEdge,
  useNodeConfig,
  useSingleDispatch,
} from "@/schematic/queries";
import { useKey } from "@/schematic/Suspended";
import { Diagram as Base } from "@/vis/diagram";
import { internalNodeBox, resolveEndpoint } from "@/vis/diagram/util";

// The partial merges against the latest config through a ref, not the render's
// closure, so two changes in one tick both land.
const NodeRenderer = ({ position, ...rest }: Base.NodeProps): ReactElement | null => {
  const config = useNodeConfig({ elKey: rest.nodeKey });
  const configRef = useSyncedRef(config);
  const dispatch = useSingleDispatch();
  const handleChange = (partial: Partial<object>) => {
    const current = configRef.current;
    if (current == null) return;
    dispatch(
      schematic.setChanged({
        key: rest.nodeKey,
        base: current,
        next: { ...current, ...partial },
      }),
    );
  };
  // React Flow re-measures handles only on a resize. A symbol change may not resize.
  const updateNodeInternals = useUpdateNodeInternals();
  const symbolKey = config == null ? null : Node.symbolKey(config);
  const prevSymbolKey = useRef(symbolKey);
  useEffect(() => {
    if (prevSymbolKey.current === symbolKey) return;
    prevSymbolKey.current = symbolKey;
    updateNodeInternals(rest.nodeKey);
  }, [symbolKey, rest.nodeKey, updateNodeInternals]);
  // React flow takes time to unmount, so the config can be temporarily undefined.
  if (config == null) return null;
  const Spec = Node.resolveSpec(config.variant);
  return (
    <Spec.Node
      config={config}
      onConfigChange={handleChange}
      position={Spec.needsPosition === true ? position : undefined}
      {...rest}
    />
  );
};

const EdgeRenderer = (props: Base.EdgeProps): ReactElement | null => {
  const edge = useEdge({ elKey: props.edgeKey });
  const edgeRef = useSyncedRef(edge);
  const dispatch = useSingleDispatch();
  const handleChange = (partial: Partial<schematic.Edge>) => {
    const current = edgeRef.current;
    if (current == null) return;
    dispatch(schematic.setEdge({ edge: { ...current, ...partial } }));
  };
  // React flow takes time to unmount, so the edge can be temporarily undefined.
  if (edge == null) return null;
  const Spec = Edge.resolveSpec(edge.variant);
  return <Spec.Edge edge={edge} onChange={handleChange} {...props} />;
};

const EdgeJumpProvider = ({ children }: PropsWithChildren): ReactElement => {
  const key = useKey();
  const edges = useAllEdges({ key });
  const rfStore = useStoreApi();
  const [jumpStore] = useState(Edge.Jumps.createStore);

  const recompute = () => {
    const { nodeLookup, transform } = rfStore.getState();
    const zoom = transform[2];
    const polylines: Edge.Jumps.Polyline[] = [];
    edges.forEach((edge, order) => {
      const sourceNode = nodeLookup.get(edge.source.node);
      const targetNode = nodeLookup.get(edge.target.node);
      if (sourceNode == null || targetNode == null) return;
      const source = resolveEndpoint(sourceNode.internals, edge.source.param);
      const target = resolveEndpoint(targetNode.internals, edge.target.param);
      if (source == null || target == null) return;
      const segments = Edge.Segmented.build({
        source,
        target,
        sourceBox: internalNodeBox(sourceNode),
        targetBox: internalNodeBox(targetNode),
        middleSegments: edge.segments,
      });
      const points = Edge.Segmented.segmentsToPoints(
        source.position,
        segments,
        zoom,
        true,
      );
      if (points.length < 2) return;
      const { strokeWidth } = edge;
      const width = Edge.resolveSpec(edge.variant).renderedWidth(strokeWidth);
      polylines.push({ key: edge.key, points, order, strokeWidth, width });
    });
    jumpStore.commit(Edge.Jumps.findCrossings(polylines));
  };

  // Coalesce React Flow's many per-interaction store updates into one recompute per
  // frame, skipping frames where no node geometry or zoom changed (panning, selection,
  // hover).
  const frameRef = useRef<number | null>(null);
  const geometryRef = useRef("");
  useEffect(() => {
    recompute();
    const schedule = (): void => {
      if (frameRef.current != null) return;
      frameRef.current = requestAnimationFrame(() => {
        frameRef.current = null;
        const { nodeLookup, transform } = rfStore.getState();
        let geometry = `${transform[2]}`;
        for (const node of nodeLookup.values()) {
          const { x, y } = node.internals.positionAbsolute;
          geometry += `;${x},${y},${node.measured.width},${node.measured.height}`;
        }
        if (geometry === geometryRef.current) return;
        geometryRef.current = geometry;
        recompute();
      });
    };
    const unsubscribe = rfStore.subscribe(schedule);
    return () => {
      unsubscribe();
      if (frameRef.current != null) {
        cancelAnimationFrame(frameRef.current);
        frameRef.current = null;
      }
    };
  }, [recompute, rfStore]);

  return <Edge.Jumps.Context value={jumpStore}>{children}</Edge.Jumps.Context>;
};

export const Diagram = Base.create({
  node: Component.renderProp(NodeRenderer),
  edge: Component.renderProp(EdgeRenderer),
  connectionLine: Component.renderProp(Edge.ConnectionLine),
  Provider: EdgeJumpProvider,
});

export const nodeChangesToActions = (
  changes: Base.NodeChange[],
): schematic.Action[] => {
  const actions: schematic.Action[] = [];
  changes.forEach((ch) => {
    switch (ch.type) {
      case "position":
        actions.push(schematic.setNodePosition({ key: ch.key, position: ch.position }));
        return;
      case "remove":
        actions.push(schematic.removeNode({ key: ch.key }));
    }
  });
  return actions;
};

export const edgeChangesToActions = (changes: Base.EdgeChange[]): schematic.Action[] =>
  changes.flatMap((ch) => {
    switch (ch.type) {
      case "add":
        return [schematic.addEdge({ edge: ch.edge })];
      case "remove":
        return [schematic.removeEdge({ key: ch.key })];
      default:
        return [];
    }
  });
