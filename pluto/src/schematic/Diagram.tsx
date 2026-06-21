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
import { type ReactElement, useCallback } from "react";

import { Component } from "@/component";
import { Edge } from "@/schematic/edge";
import { type ElementConfig } from "@/schematic/element";
import { Node } from "@/schematic/node";
import { useSelectElementConfig, useSingleDispatch } from "@/schematic/queries";
import { Diagram as Base } from "@/vis/diagram";

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

export const Diagram = Base.create({
  node: Component.renderProp(NodeRenderer),
  edge: Component.renderProp(EdgeRenderer),
  connectionLine: Component.renderProp(Edge.ConnectionLine),
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
