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
import { type connector } from "@/schematic/edge/connector";
import { ConnectionLine, Edge } from "@/schematic/edge/Edge";
import { type EdgeType } from "@/schematic/edge/paths";
import { DRAG_HANDLE_CLASS } from "@/schematic/symbol/Grid";
import { REGISTRY, type Variant } from "@/schematic/symbol/registry";
import { type diagram } from "@/vis/diagram/aether";
import { Diagram } from "@/vis/diagram";

export interface SchematicProps extends Omit<
  Diagram.DiagramProps,
  "dragHandleSelector"
> {}

export interface NodeProps extends Record<string, unknown> {
  variant: Variant;
}

export interface EdgeProps extends Record<string, unknown> {
  segments?: connector.Segment[];
  variant?: EdgeType;
  color?: string;
}

export interface SchematicHooks {
  /** Called inside the node renderer to read node props for the given key. */
  useNodeProps: (key: string) => NodeProps | undefined;
  /** Called inside the edge renderer to read edge props for the given key. */
  useEdgeProps: (key: string) => EdgeProps | undefined;
  /** Returns a stable callback for persisting a partial props update. */
  useSetElementProps: () => (key: string, props: Record<string, unknown>) => void;
}

const AUTO_RENDER_INTERVAL = TimeSpan.seconds(1).milliseconds;

export const create = (hooks: SchematicHooks): FC<SchematicProps> => {
  const NodeRenderer = ({
    nodeKey,
    position,
    selected,
    draggable,
  }: Diagram.NodeProps): ReactElement | null => {
    const props = hooks.useNodeProps(nodeKey);
    const setElementProps = hooks.useSetElementProps();
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

  const EdgeRenderer = (props: diagram.EdgeProps): ReactElement | null => {
    const edgeProps = hooks.useEdgeProps(props.edgeKey);
    const setElementProps = hooks.useSetElementProps();
    const handleSegmentsChange = useCallback(
      (segments: connector.Segment[]) => {
        setElementProps(props.edgeKey, { segments });
      },
      [props.edgeKey, setElementProps],
    );
    return (
      <Edge
        {...props}
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
    connectionLine: Component.renderProp(ConnectionLine),
  });

  const Schematic = ({ className, ...props }: SchematicProps): ReactElement => (
    <Base
      className={CSS(CSS.B("schematic"), className)}
      dragHandleSelector={`.${DRAG_HANDLE_CLASS}`}
      autoRenderInterval={AUTO_RENDER_INTERVAL}
      {...props}
    />
  );
  return Schematic;
};
