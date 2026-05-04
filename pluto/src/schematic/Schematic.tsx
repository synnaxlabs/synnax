// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/Schematic.css";

import { TimeSpan } from "@synnaxlabs/x";
import { type FC, type ReactElement, useCallback } from "react";

import { Component } from "@/component";
import { CSS } from "@/css";
import { Key } from "@/key";
import { Edge } from "@/schematic/edge";
import { DRAG_HANDLE_CLASS } from "@/schematic/symbol/Grid";
import { REGISTRY, type Variant } from "@/schematic/symbol/registry";
import { Diagram } from "@/vis/diagram";
import { type diagram } from "@/vis/diagram/aether";

export interface SchematicProps extends Omit<
  Diagram.DiagramProps,
  "dragHandleSelector"
> {
  itemKey: string;
}

export interface ElementConfig {
  variant: Variant;
}

export interface SchematicHooks {
  useConfig: (
    itemKey: string,
    nodeKey: string,
  ) => [ElementConfig, (key: string, props: Partial<ElementConfig>) => void];
}

const AUTO_RENDER_INTERVAL = TimeSpan.seconds(1).milliseconds;

export const create = ({ useConfig }: SchematicHooks): FC<SchematicProps> => {
  const NodeRenderer = ({
    nodeKey,
    position,
    selected,
    draggable,
  }: Diagram.NodeProps): ReactElement | null => {
    const itemKey = Key.use<string>("Schematic.NodeRenderer");
    const [config, setConfig] = useConfig(itemKey, nodeKey);
    const { variant } = config;
    const handleChange = useCallback(
      (next: Partial<ElementConfig>) => setConfig(nodeKey, { variant, ...next }),
      [nodeKey, variant, setConfig],
    );
    const Spec = REGISTRY[variant];
    return (
      <Spec.Symbol
        nodeKey={nodeKey}
        position={position}
        selected={selected}
        draggable={draggable}
        onConfigChange={handleChange}
        config={config}
      />
    );
  };

  const EdgeRenderer = ({
    edgeKey,
    ...rest
  }: diagram.EdgeProps): ReactElement | null => {
    const itemKey = Key.use<string>("Schematic.EdgeRenderer");
    const [config, setConfig] = useConfig(itemKey, edgeKey);
    const { variant } = config;
    const handleChange = useCallback(
      (props: Partial<ElementConfig>) => setConfig(edgeKey, props),
      [edgeKey, setConfig],
    );
    const E = Edge.resolve(variant);
    return <E {...rest} edgeKey={edgeKey} onChange={handleChange} config={config} />;
  };

  const Base = Diagram.create({
    node: Component.renderProp(NodeRenderer),
    edge: Component.renderProp(EdgeRenderer),
    connectionLine: Component.renderProp(Edge.ConnectionLine),
  });

  const Schematic = ({ className, itemKey, ...rest }: SchematicProps): ReactElement => (
    <Key.Provider<string> value={itemKey}>
      <Base
        className={CSS(CSS.B("schematic"), className)}
        dragHandleSelector={`.${DRAG_HANDLE_CLASS}`}
        autoRenderInterval={AUTO_RENDER_INTERVAL}
        {...rest}
      />
    </Key.Provider>
  );
  return Schematic;
};
