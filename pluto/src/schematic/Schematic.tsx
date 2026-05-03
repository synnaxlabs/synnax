// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/Schematic.css";

import { type color, TimeSpan } from "@synnaxlabs/x";
import { type FC, type ReactElement, useCallback } from "react";

import { Component } from "@/component";
import { CSS } from "@/css";
import { Key } from "@/key";
import { Edge } from "@/schematic/edge";
import { type connector } from "@/schematic/edge/connector";
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

export interface NodeProps extends Record<string, unknown> {
  variant: Variant;
}

export interface EdgeProps extends Record<string, unknown> {
  segments?: connector.Segment[];
  variant?: Edge.Variant;
  color?: color.Color;
}

export interface SchematicHooks {
  /** Called inside the node renderer to read node props for the given key. */
  useNodeProps: (itemKey: string, nodeKey: string) => NodeProps | undefined;
  /** Called inside the edge renderer to read edge props for the given key. */
  useEdgeProps: (itemKey: string, edgeKey: string) => EdgeProps | undefined;
  /** Returns a stable callback for persisting a partial props update. */
  useSetElementProps: (
    itemKey: string,
  ) => (key: string, props: NodeProps | EdgeProps) => void;
}

const AUTO_RENDER_INTERVAL = TimeSpan.seconds(1).milliseconds;

export const create = (hooks: SchematicHooks): FC<SchematicProps> => {
  const NodeRenderer = ({
    nodeKey,
    position,
    selected,
    draggable,
  }: Diagram.NodeProps): ReactElement | null => {
    const itemKey = Key.use<string>("Schematic.NodeRenderer");
    const props = hooks.useNodeProps(itemKey, nodeKey);
    const setElementProps = hooks.useSetElementProps(itemKey);
    const variant = props?.variant;
    const handleChange = useCallback(
      (next: object) => {
        if (variant == null) return;
        setElementProps(nodeKey, { variant, ...next });
      },
      [nodeKey, variant, setElementProps],
    );
    if (props == null || variant == null) return null;
    const C = REGISTRY[variant];
    if (C == null) throw new Error(`Symbol ${variant} not found`);
    const { variant: _, ...rest } = props;
    return (
      <C.Symbol
        nodeKey={nodeKey}
        position={position}
        selected={selected}
        draggable={draggable}
        onChange={handleChange}
        data={rest}
      />
    );
  };

  const EdgeRenderer = ({
    edgeKey,
    ...rest
  }: diagram.EdgeProps): ReactElement | null => {
    const itemKey = Key.use<string>("Schematic.EdgeRenderer");
    const edgeProps = hooks.useEdgeProps(itemKey, edgeKey);
    const setElementProps = hooks.useSetElementProps(itemKey);
    const handleSegmentsChange = useCallback(
      (segments: connector.Segment[]) => setElementProps(edgeKey, { segments }),
      [edgeKey, setElementProps],
    );
    return (
      <Edge.Edge
        {...rest}
        edgeKey={edgeKey}
        segments={edgeProps?.segments}
        variant={edgeProps?.variant}
        color={edgeProps?.color}
        onSegmentsChange={handleSegmentsChange}
      />
    );
  };

  const Base = Diagram.create({
    node: Component.renderProp(NodeRenderer),
    edge: Component.renderProp(EdgeRenderer),
    connectionLine: Component.renderProp(Edge.ConnectionLine),
  });

  const Schematic = ({
    className,
    itemKey,
    ...props
  }: SchematicProps): ReactElement => (
    <Key.Provider<string> value={itemKey}>
      <Base
        className={CSS(CSS.B("schematic"), className)}
        dragHandleSelector={`.${DRAG_HANDLE_CLASS}`}
        autoRenderInterval={AUTO_RENDER_INTERVAL}
        {...props}
      />
    </Key.Provider>
  );
  return Schematic;
};
