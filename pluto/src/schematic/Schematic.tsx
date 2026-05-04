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
import { Node } from "@/schematic/node";
import { Diagram } from "@/vis/diagram";
import { type diagram } from "@/vis/diagram/aether";

export interface SchematicProps extends Omit<
  Diagram.DiagramProps,
  "dragHandleSelector"
> {
  itemKey: string;
}

export type ElementConfig = (Edge.Config | Node.Config) & {
  variant: Edge.Variant | Node.Variant;
};

const DRAG_HANDLE_SELECTOR = `.${Node.DRAG_HANDLE_CLASS}`;

export interface CreateSchematicParams {
  useConfig: (
    itemKey: string,
    nodeKey: string,
  ) => [ElementConfig, (key: string, props: Partial<ElementConfig>) => void];
}

const AUTO_RENDER_INTERVAL = TimeSpan.seconds(1).milliseconds;

export const create = ({ useConfig }: CreateSchematicParams): FC<SchematicProps> => {
  const NodeRenderer = ({ nodeKey, ...rest }: Diagram.NodeProps): ReactElement => {
    const itemKey = Key.use<string>("Schematic.NodeRenderer");
    const [config, setConfig] = useConfig(itemKey, nodeKey);
    const { variant } = config;
    const handleChange = useCallback(
      (next: Partial<ElementConfig>) => setConfig(nodeKey, next),
      [nodeKey, setConfig],
    );
    const Spec = Node.resolveSpec(variant);
    return (
      <Spec.Node
        nodeKey={nodeKey}
        onConfigChange={handleChange}
        config={config}
        {...rest}
      />
    );
  };

  const EdgeRenderer = ({ edgeKey, ...rest }: diagram.EdgeProps): ReactElement => {
    const itemKey = Key.use<string>("Schematic.EdgeRenderer");
    const [config, setConfig] = useConfig(itemKey, edgeKey);
    const { variant } = config;
    const handleChange = useCallback(
      (props: Partial<ElementConfig>) => setConfig(edgeKey, props),
      [edgeKey, setConfig],
    );
    const E = Edge.resolve(variant);
    return (
      <E
        {...rest}
        edgeKey={edgeKey}
        onChange={handleChange}
        // eslint is not smart enough to know this type assertion is necessary
        // eslint-disable-next-line @typescript-eslint/no-unnecessary-type-assertion
        config={config as Edge.Config}
      />
    );
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
        dragHandleSelector={DRAG_HANDLE_SELECTOR}
        autoRenderInterval={AUTO_RENDER_INTERVAL}
        {...rest}
      />
    </Key.Provider>
  );
  return Schematic;
};
