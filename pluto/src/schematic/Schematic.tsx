// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/Schematic.css";

import { schematic } from "@synnaxlabs/client";
import { box, type record, TimeSpan, xy } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useRef } from "react";
import { z } from "zod";

import { Component } from "@/component";
import { CSS } from "@/css";
import { Haul } from "@/haul";
import { useSyncedRef } from "@/hooks";
import { Provider, useKey } from "@/schematic/Context";
import { Edge } from "@/schematic/edge";
import { Node } from "@/schematic/node";
import {
  useAddNode,
  useDispatch,
  useRetrieve,
  useSelectConfig,
} from "@/schematic/queries";
import { Diagram } from "@/vis/diagram";
import { type diagram } from "@/vis/diagram/aether";

export interface SchematicProps extends Omit<
  Diagram.DiagramProps,
  | "dragHandleSelector"
  | "nodes"
  | "edges"
  | "onNodesChange"
  | "onEdgesChange"
  | "selected"
  | "onSelectionChange"
> {
  resourceKey: string;
  selected?: string[];
  onSelectionChange?: (selected: string[]) => void;
}

export const elementConfigZ = z.discriminatedUnion("variant", [
  ...Node.configZ.options,
  ...Edge.configZ.options,
]);
export type ElementConfig = z.infer<typeof elementConfigZ>;

const AUTO_RENDER_INTERVAL = TimeSpan.seconds(1).milliseconds;
export const HAUL_TYPE = "schematic-element";

const EdgeRenderer = (props: diagram.EdgeProps): ReactElement | null => {
  const { edgeKey } = props;
  const schematicKey = useKey();
  const config = useSelectConfig({ key: schematicKey, configKey: edgeKey });
  const configRef = useSyncedRef(config);
  const { update: dispatch } = useDispatch();
  const onChange = useCallback(
    (next: Partial<Edge.Config>) =>
      dispatch({
        key: schematicKey,
        actions: schematic.setConfig({
          key: edgeKey,
          config: { ...configRef.current, ...next } as record.Unknown,
        }),
      }),
    [edgeKey, schematicKey, dispatch],
  );
  if (config == null) return null;
  const variant = (config as { variant?: string }).variant;
  if (variant == null) return null;
  const E = Edge.resolve(variant);
  return (
    <E
      // eslint-disable-next-line @typescript-eslint/no-unnecessary-type-assertion
      config={config as Edge.Config}
      onChange={onChange}
      {...props}
    />
  );
};

const SchematicDiagram = Diagram.create({
  node: Component.renderProp(Node.Node),
  edge: Component.renderProp(EdgeRenderer),
  connectionLine: Component.renderProp(Edge.ConnectionLine),
});

const nodeChangeToAction = (change: Diagram.NodeChange): schematic.Action | null => {
  switch (change.type) {
    case "position":
      return schematic.setNodePosition({ key: change.key, position: change.position });
    case "remove":
      return schematic.removeNode({ key: change.key });
    default:
      return null;
  }
};

const edgeChangeToActions = (change: Diagram.EdgeChange): schematic.Action[] => {
  switch (change.type) {
    case "add":
      return [schematic.setEdge({ edge: change.edge })];
    case "remove":
      return [schematic.removeEdge({ key: change.key })];
    default:
      return [];
  }
};

export const Schematic = ({
  className,
  resourceKey,
  viewport,
  onDoubleClick,
  onSelectionChange,
  ...props
}: SchematicProps): ReactElement => {
  const { data: doc } = useRetrieve({ key: resourceKey });
  const { update: dispatch } = useDispatch();

  const handleNodesChange = useCallback(
    (changes: Diagram.NodeChange[]) => {
      const actions = changes
        .map(nodeChangeToAction)
        .filter((a): a is schematic.Action => a != null);
      if (actions.length > 0) dispatch({ key: resourceKey, actions });
    },
    [resourceKey, dispatch],
  );

  const handleEdgesChange = useCallback(
    (changes: Diagram.EdgeChange[]) => {
      const actions = changes.flatMap(edgeChangeToActions);
      if (actions.length > 0) dispatch({ key: resourceKey, actions });
    },
    [resourceKey, dispatch],
  );

  const handleAddNode = useAddNode(resourceKey);
  const ref = useRef<HTMLDivElement>(null);
  const viewportRef = useSyncedRef(viewport);
  const calculateCursorPosition = useCallback(
    (cursor: xy.Crude) =>
      Diagram.calculateCursorPosition(
        box.construct(ref.current ?? box.ZERO),
        cursor,
        viewportRef.current,
      ),
    [],
  );

  const handleDrop = useCallback(
    ({ items, event }: Haul.OnDropProps): Haul.Item[] => {
      const valid = Haul.filterByType(HAUL_TYPE, items);
      if (event == null) return valid;
      valid.forEach(({ key, data }) => {
        const pos = xy.truncate(calculateCursorPosition(event), 0);
        handleAddNode(key.toString(), pos, data);
      });
      return valid;
    },
    [handleAddNode, calculateCursorPosition],
  );

  const dropProps = Haul.useDrop({
    type: "Schematic",
    key: resourceKey,
    canDrop: Haul.canDropOfType(HAUL_TYPE),
    onDrop: handleDrop,
  });

  const handleClearSelection = useCallback(() => onSelectionChange?.([]), []);

  Diagram.useTriggers({
    onCopy: () => {},
    onPaste: () => {},
    onSelectAll: () => {},
    onClear: handleClearSelection,
    onUndo: () => {},
    onRedo: () => {},
    region: ref,
  });

  return (
    <Provider value={resourceKey}>
      <SchematicDiagram
        ref={ref}
        className={CSS(CSS.B("schematic"), className)}
        dragHandleSelector={`.${Node.DRAG_HANDLE_CLASS}`}
        autoRenderInterval={AUTO_RENDER_INTERVAL}
        nodes={doc?.nodes ?? []}
        edges={doc?.edges ?? []}
        onNodesChange={handleNodesChange}
        onEdgesChange={handleEdgesChange}
        viewport={viewport}
        onSelectionChange={onSelectionChange}
        onDoubleClick={onDoubleClick}
        {...dropProps}
        {...props}
      />
    </Provider>
  );
};

export const REGISTRY = {
  ...Node.REGISTRY,
  ...Edge.REGISTRY,
};
