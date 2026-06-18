// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { type record } from "@synnaxlabs/x";
import { useStoreApi } from "@xyflow/react";
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
import { internalNodeBox, resolveEndpoint } from "@/vis/diagram/util";

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
      const source = resolveEndpoint(sourceNode.internals, edge.source.param);
      const target = resolveEndpoint(targetNode.internals, edge.target.param);
      if (source == null || target == null) return;
      const cfg = configs.get(edge.key) as Edge.Config | undefined;
      const segments = Edge.Segmented.build({
        sourcePos: source.position,
        targetPos: target.position,
        sourceOrientation: source.orientation,
        targetOrientation: target.orientation,
        sourceBox: internalNodeBox(sourceNode),
        targetBox: internalNodeBox(targetNode),
        middleSegments: cfg?.segments ?? [],
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

  // Coalesce React Flow's many per-interaction store updates into one recompute per frame,
  // skipping frames where no node geometry or zoom changed (panning, selection, hover).
  const frameRef = useRef<number | null>(null);
  const geometryRef = useRef("");
  useEffect(() => {
    recompute();
    const schedule = (): void => {
      if (frameRef.current != null) return;
      frameRef.current = requestAnimationFrame(() => {
        frameRef.current = null;
        const { nodeLookup, transform } = store.getState();
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
