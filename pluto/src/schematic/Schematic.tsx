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
import { type FC, type ReactElement } from "react";
import { z } from "zod";

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

export const elementConfigZ = z.discriminatedUnion("variant", [
  ...Node.configZ.options,
  ...Edge.configZ.options,
]);
export type ElementConfig = z.infer<typeof elementConfigZ>;

const DRAG_HANDLE_SELECTOR = `.${Node.DRAG_HANDLE_CLASS}`;

export interface CreateSchematicParams {
  useConfig: (
    itemKey: string,
    nodeKey: string,
  ) => [ElementConfig, (props: Partial<ElementConfig>) => void];
}

const AUTO_RENDER_INTERVAL = TimeSpan.seconds(1).milliseconds;

export const create = ({ useConfig }: CreateSchematicParams): FC<SchematicProps> => {
  const NodeRenderer = (props: Diagram.NodeProps): ReactElement => {
    const { nodeKey } = props;
    const itemKey = Key.use<string>("Schematic.NodeRenderer");
    const [config, setConfig] = useConfig(itemKey, nodeKey);
    const N = Node.resolve(config.variant);
    // eslint-disable-next-line @typescript-eslint/no-unnecessary-type-assertion
    return <N onConfigChange={setConfig} config={config as Node.Config} {...props} />;
  };

  const EdgeRenderer = (props: diagram.EdgeProps): ReactElement => {
    const { edgeKey } = props;
    const itemKey = Key.use<string>("Schematic.EdgeRenderer");
    const [config, setConfig] = useConfig(itemKey, edgeKey);
    const E = Edge.resolve(config.variant);
    // eslint-disable-next-line @typescript-eslint/no-unnecessary-type-assertion
    return <E onChange={setConfig} config={config as Edge.Config} {...props} />;
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
