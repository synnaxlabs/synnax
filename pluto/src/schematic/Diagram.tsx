// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { type location, type record, type xy } from "@synnaxlabs/x";
import { type InternalNode, useStoreApi } from "@xyflow/react";
import {
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useEffect,
  useMemo,
  useRef,
} from "react";

import { Component } from "@/component";
import { Key } from "@/key";
import { Edge } from "@/schematic/edge";
import { Node } from "@/schematic/node";
import {
  useDispatch,
  useSelectAllEdges,
  useSelectConfigs,
  useSelectElementConfig,
} from "@/schematic/queries";
import { Diagram as Base } from "@/vis/diagram";
import { internalNodeBox } from "@/vis/diagram/util";

interface Endpoint {
  position: xy.XY;
  orientation: location.Outer;
}

// Mirror of React Flow's getHandlePosition: the connection point sits on the handle's
// edge for its side, not at the handle center, so the reconstructed polyline matches the
// one the edge actually draws.
const resolveEndpoint = (node: InternalNode, handleKey: string): Endpoint | null => {
  const bounds = node.internals.handleBounds;
  const handles = [...(bounds?.source ?? []), ...(bounds?.target ?? [])];
  const handle = handles.find((h) => h.id === handleKey) ?? handles[0];
  if (handle == null) return null;
  const abs = node.internals.positionAbsolute;
  const x = abs.x + handle.x;
  const y = abs.y + handle.y;
  const { width: w, height: h } = handle;
  const orientation = handle.position as location.Outer;
  let position: xy.XY;
  switch (orientation) {
    case "top":
      position = { x: x + w / 2, y };
      break;
    case "right":
      position = { x: x + w, y: y + h / 2 };
      break;
    case "bottom":
      position = { x: x + w / 2, y: y + h };
      break;
    default:
      position = { x, y: y + h / 2 };
  }
  return { position, orientation };
};

const NodeRenderer = ({ position, ...rest }: Base.NodeProps): ReactElement | null => {
  const { nodeKey } = rest;
  const key = Key.use<string>("Schematic.Node.Renderer");
  const config = useSelectElementConfig({ key, elKey: nodeKey });
  const { dispatch } = useDispatch();
  const handleChange = useCallback(
    (config: Partial<Node.Config>) =>
      dispatch({ key, actions: schematic.setConfig({ key: nodeKey, config }) }),
    [nodeKey, key, dispatch],
  );
  // React flow can take time to unmount the node, meaning that we need to tolerate
  // temporarily undefined configs.
  if (config == null) return null;
  const Spec = Node.resolveSpec(config.variant);
  return (
    <Spec.Node
      config={config as Node.Config}
      onConfigChange={handleChange}
      position={Spec.needsPosition === true ? position : undefined}
      {...rest}
    />
  );
};

const EdgeRenderer = (props: Base.EdgeProps): ReactElement | null => {
  const { edgeKey } = props;
  const key = Key.use<string>("Schematic.Edge.Renderer");
  const config = useSelectElementConfig({ key, elKey: edgeKey });
  const { dispatch } = useDispatch();
  const handleChange = useCallback(
    (config: Partial<Edge.Config>) =>
      dispatch({ key, actions: schematic.setConfig({ key: edgeKey, config }) }),
    [edgeKey, key, dispatch],
  );
  // React flow can take time to unmount the edge, meaning that we need to tolerate
  // temporarily undefined configs.
  if (config == null) return null;
  const Spec = Edge.resolveSpec(config.variant);
  return (
    <Spec.Edge config={config as Edge.Config} onChange={handleChange} {...props} />
  );
};

const EdgeJumpProvider = ({ children }: PropsWithChildren): ReactElement => {
  const key = Key.use<string>("Schematic.EdgeJumps");
  const edges = useSelectAllEdges({ key });
  const edgeKeys = useMemo(() => edges.map((e) => e.key), [edges]);
  const configs = useSelectConfigs({ key, keys: edgeKeys });
  const store = useStoreApi();
  const jumpsRef = useRef<Edge.Jumps.Store | null>(null);
  jumpsRef.current ??= Edge.Jumps.create();
  const jumps = jumpsRef.current;

  const recompute = useCallback(() => {
    const { nodeLookup, transform } = store.getState();
    const zoom = transform[2];
    const polylines: Edge.Jumps.Polyline[] = [];
    edges.forEach((edge, order) => {
      const sourceNode = nodeLookup.get(edge.source.node);
      const targetNode = nodeLookup.get(edge.target.node);
      if (sourceNode == null || targetNode == null) return;
      const source = resolveEndpoint(sourceNode, edge.source.param);
      const target = resolveEndpoint(targetNode, edge.target.param);
      if (source == null || target == null) return;
      const cfg = configs.get(edge.key) as { segments?: Edge.Segmented.Segment[] };
      const middleSegments = cfg?.segments ?? [];
      const segments =
        middleSegments.length === 0
          ? Edge.Segmented.createConnector({
              sourcePos: source.position,
              targetPos: target.position,
              sourceOrientation: source.orientation,
              targetOrientation: target.orientation,
              sourceBox: internalNodeBox(sourceNode),
              targetBox: internalNodeBox(targetNode),
            })
          : Edge.Segmented.stitchEdge({
              sourceOrientation: source.orientation,
              targetOrientation: target.orientation,
              sourcePos: source.position,
              targetPos: target.position,
              middleSegments,
            });
      const points = Edge.Segmented.segmentsToPoints(
        source.position,
        segments,
        zoom,
        true,
      );
      if (points.length >= 2) polylines.push({ key: edge.key, points, order });
    });
    jumps.commit(Edge.Jumps.findCrossings(polylines));
  }, [edges, configs, store, jumps]);

  // React Flow emits many store updates per interaction; coalesce them so the crossings
  // are recomputed at most once per frame.
  const frameRef = useRef<number | null>(null);
  useEffect(() => {
    recompute();
    const schedule = (): void => {
      if (frameRef.current != null) return;
      frameRef.current = requestAnimationFrame(() => {
        frameRef.current = null;
        recompute();
      });
    };
    const unsubscribe = store.subscribe(schedule);
    return () => {
      unsubscribe();
      if (frameRef.current != null) cancelAnimationFrame(frameRef.current);
    };
  }, [recompute, store]);

  return <Edge.Jumps.Context value={jumps}>{children}</Edge.Jumps.Context>;
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
      case "dimensions":
        // onResize drives the symbol's size during a drag; skip the competing write.
        if (ch.resizing) return;
        actions.push(
          schematic.setNodeMeasured({ key: ch.key, measured: ch.dimensions }),
        );
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
        return [
          schematic.addEdge({ edge: ch.edge }),
          schematic.setConfig({
            key: ch.edge.key,
            // TODO: Remove this once schematic configs are strongly typed.
            config: Edge.REGISTRY.pipe.defaultConfig() as unknown as record.Unknown,
          }),
        ];
      case "remove":
        return [schematic.removeEdge({ key: ch.key })];
      default:
        return [];
    }
  });
