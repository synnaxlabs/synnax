// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/Schematic.css";

import { type schematic } from "@synnaxlabs/client";
import { CSS } from "@synnaxlabs/lyra/css";
import { Haul } from "@synnaxlabs/lyra/haul";
import { useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { Key } from "@synnaxlabs/lyra/key";
import { box } from "@synnaxlabs/x/box";
import { telem } from "@synnaxlabs/x/telem";
import { xy } from "@synnaxlabs/x/xy";
import { type ReactElement, useCallback, useRef } from "react";

import { Diagram as BaseDiagram } from "@/diagram";
import { useClipboard } from "@/schematic/clipboard";
import {
  Diagram,
  edgeChangesToActions,
  nodeChangesToActions,
} from "@/schematic/Diagram";
import { canDropHaulItem, filterHaulItems } from "@/schematic/haul";
import { Node } from "@/schematic/node";
import {
  useAddNode,
  useDispatch,
  useSelectAllEdges,
  useSelectAllNodes,
} from "@/schematic/queries";

export interface SchematicProps extends Omit<
  BaseDiagram.DiagramProps,
  | "dragHandleSelector"
  | "nodes"
  | "edges"
  | "onNodesChange"
  | "onEdgesChange"
  | "onChange"
> {
  enableTriggers?: boolean | (() => boolean);
  resourceKey: schematic.Key;
}
const AUTO_RENDER_INTERVAL = telem.TimeSpan.seconds(1).milliseconds;
const DRAG_HANDLE_SELECTOR = `.${Node.DRAG_HANDLE_CLASS}`;

export const Schematic = ({
  className,
  resourceKey: key,
  viewport,
  onDoubleClick,
  onSelectionChange,
  selected,
  enableTriggers,
  ...props
}: SchematicProps): ReactElement => {
  const nodes = useSelectAllNodes({ key });
  const nodesRef = useSyncedRef(nodes);
  const edges = useSelectAllEdges({ key });
  const edgesRef = useSyncedRef(edges);
  const { update: dispatch } = useDispatch();
  const handleNodesChange = useCallback(
    (changes: BaseDiagram.NodeChange[]) => {
      const actions = nodeChangesToActions(changes);
      if (actions.length > 0) dispatch({ key, actions });
    },
    [key, dispatch],
  );

  const handleEdgesChange = useCallback(
    (changes: BaseDiagram.EdgeChange[]) => {
      const actions = edgeChangesToActions(changes);
      if (actions.length > 0) dispatch({ key, actions });
    },
    [key, dispatch],
  );

  const handleAddNode = useAddNode(key);
  const ref = useRef<HTMLDivElement>(null);
  const viewportRef = useSyncedRef(viewport);
  const calculateCursorPosition = useCallback((cursor: xy.Crude) => {
    if (ref.current == null) return xy.ZERO;
    return BaseDiagram.calculateCursorPosition(
      box.construct(ref.current),
      cursor,
      viewportRef.current,
    );
  }, []);

  const handleDrop = useCallback(
    ({ items, event }: Haul.OnDropProps): Haul.Item[] => {
      const valid = filterHaulItems(items);
      if (event == null) return valid;
      const position = xy.truncate(calculateCursorPosition(event), 0);
      valid.forEach(({ data }) => handleAddNode({ ...data, position }));
      return valid;
    },
    [handleAddNode, calculateCursorPosition],
  );

  const dropProps = Haul.useDrop({
    type: "Schematic",
    key,
    canDrop: canDropHaulItem,
    onDrop: handleDrop,
  });

  const handleClearSelection = useCallback(
    () => onSelectionChange?.([]),
    [onSelectionChange],
  );
  const handleSelectAll = useCallback(() => {
    onSelectionChange?.([
      ...nodesRef.current.map((n) => n.key),
      ...edgesRef.current.map((e) => e.key),
    ]);
  }, [onSelectionChange]);

  const { onCopy, onPaste } = useClipboard({
    key,
    selected,
    onPaste: onSelectionChange,
  });

  BaseDiagram.useTriggers({
    onSelectAll: handleSelectAll,
    onClearSelection: handleClearSelection,
    enabled: enableTriggers,
  });

  return (
    <Key.Provider value={key}>
      <Diagram
        ref={ref}
        className={CSS(CSS.B("schematic"), className)}
        dragHandleSelector={DRAG_HANDLE_SELECTOR}
        autoRenderInterval={AUTO_RENDER_INTERVAL}
        onNodesChange={handleNodesChange}
        onEdgesChange={handleEdgesChange}
        viewport={viewport}
        onSelectionChange={onSelectionChange}
        onDoubleClick={onDoubleClick}
        onCopy={onCopy}
        onPaste={onPaste}
        nodes={nodes}
        edges={edges}
        selected={selected}
        {...dropProps}
        {...props}
      />
    </Key.Provider>
  );
};
