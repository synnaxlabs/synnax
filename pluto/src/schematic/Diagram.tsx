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
import { useInitializerRef } from "@/hooks";
import { Edge } from "@/schematic/edge";
import { type ElementConfig } from "@/schematic/element";
import { Node } from "@/schematic/node";
import {
  useSelectAllEdges,
  useSelectConfigs,
  useSelectElementConfig,
  useSingleDispatch,
} from "@/schematic/queries";
import { useKey } from "@/schematic/Suspended";
import { Diagram as Base } from "@/vis/diagram";
import { internalNodeBox, resolveEndpoint } from "@/vis/diagram/util";

const useConfig = <T extends ElementConfig>(
  key: string,
): [T | undefined, (config: Partial<T>) => void] => {
  const config = useSelectElementConfig({ elKey: key });
  const dispatch = useSingleDispatch();
  const handleChange = useCallback(
    (config: Partial<T>) => dispatch(schematic.setConfig({ key, config })),
    [key, dispatch],
  );
  return [config as T | undefined, handleChange];
};

const NodeRenderer = ({ position, ...rest }: Base.NodeProps): ReactElement | null => {
  const [config, handleChange] = useConfig<Node.Config>(rest.nodeKey);
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
  const [config, handleChange] = useConfig(props.edgeKey);
  // React flow takes time to unmount, so the config can be temporarily undefined.
  if (config == null) return null;
  const Spec = Edge.resolveSpec(config.variant);
  return (
    <Spec.Edge config={config as Edge.Config} onChange={handleChange} {...props} />
  );
};

const EdgeJumpProvider = ({ children }: PropsWithChildren): ReactElement => {
  const key = useKey();
  const edges = useSelectAllEdges({ key });
  const edgeKeys = useMemo(() => edges.map((e) => e.key), [edges]);
  const configs = useSelectConfigs({ key, keys: edgeKeys });
  const rfStore = useStoreApi();
  const jumpStoreRef = useInitializerRef<Edge.Jumps.Store>(Edge.Jumps.createStore);
  const jumpStore = jumpStoreRef.current;

  const recompute = useCallback(() => {
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
      const cfg = configs.get(edge.key) as Edge.Config | undefined;
      const segments = Edge.Segmented.build({
        source,
        target,
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
    jumpStore.commit(Edge.Jumps.findCrossings(polylines));
  }, [edges, configs, rfStore, jumpStore]);

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
