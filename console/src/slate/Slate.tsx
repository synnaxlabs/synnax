// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch, type UnknownAction } from "@reduxjs/toolkit";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import {
  Diagram,
  Haul,
  Menu as PMenu,
  Slate as Core,
  Theming,
  Triggers,
  useSyncedRef,
  Viewport,
} from "@synnaxlabs/pluto";
import { box, id, xy } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo, useRef } from "react";
import { useDispatch } from "react-redux";

import { useLoadRemote } from "@/hooks/useLoadRemote";
import { useUndoableDispatch } from "@/hooks/useUndoableDispatch";
import { Layout } from "@/layout";
import {
  select,
  useSelect,
  useSelectHasPermission,
  useSelectNodeProps,
  useSelectVersion,
  useSelectViewportMode,
} from "@/slate/selectors";
import {
  addElement,
  calculatePos,
  clearSelection,
  copySelection,
  create,
  pasteSelection,
  selectAll,
  setEdges,
  setEditable,
  setElementProps,
  setFitViewOnResize,
  setNodes,
  setViewport,
  type State,
  ZERO_STATE,
} from "@/slate/slice";
import { translateSlateForward } from "@/slate/types/translate";
import { type RootState } from "@/store";

export const HAUL_TYPE = "slate-element";

interface SymbolRendererProps extends Diagram.SymbolProps {
  layoutKey: string;
  dispatch: Dispatch<UnknownAction>;
}

const SymbolRenderer = ({
  symbolKey,
  position,
  selected,
  layoutKey,
  draggable,
  dispatch,
}: SymbolRendererProps): ReactElement | null => {
  const props = useSelectNodeProps(layoutKey, symbolKey);
  const key = props?.key ?? "";
  const handleChange = useCallback(
    (props: object) => {
      if (key == null) return;
      dispatch(
        setElementProps({ layoutKey, key: symbolKey, props: { key, ...props } }),
      );
    },
    [symbolKey, layoutKey, key, dispatch],
  );

  if (props == null) return null;

  const C = Core.SYMBOLS[key];

  if (C == null) throw new Error(`Symbol ${key} not found`);

  // Just here to make sure we don't spread the key into the symbol.
  const { key: _, ...rest } = props;

  return (
    <C.Symbol
      key={key}
      id={symbolKey}
      symbolKey={symbolKey}
      position={position}
      selected={selected}
      draggable={draggable}
      onChange={handleChange}
      {...rest}
    />
  );
};

export const ContextMenu: Layout.ContextMenuRenderer = ({ layoutKey }) => (
  <PMenu.Menu level="small" gap="small">
    <Layout.MenuItems layoutKey={layoutKey} />
  </PMenu.Menu>
);

export const Loaded: Layout.Renderer = ({ layoutKey, visible }) => {
  const windowKey = useSelectWindowKey() as string;
  const slate = useSelect(layoutKey);

  const dispatch = useDispatch();
  const selector = useCallback(
    (state: RootState) => select(state, layoutKey),
    [layoutKey],
  );
  const [undoableDispatch, undo, redo] = useUndoableDispatch<RootState, State>(
    selector,
    create,
    30, // roughly the right time needed to prevent actions that get dispatch automatically by Diagram.tsx, like setNodes immediately following addElement
  );

  const theme = Theming.use();
  const viewportRef = useSyncedRef(slate.viewport);

  const canBeEditable = useSelectHasPermission();
  if (!canBeEditable && slate.editable)
    dispatch(setEditable({ key: layoutKey, editable: false }));

  const handleEdgesChange: Diagram.DiagramProps["onEdgesChange"] = useCallback(
    (edges) => undoableDispatch(setEdges({ key: layoutKey, edges })),
    [layoutKey, undoableDispatch],
  );

  const handleNodesChange: Diagram.DiagramProps["onNodesChange"] = useCallback(
    (nodes, changes) => {
      // @ts-expect-error - Sometimes, the nodes do have dragging
      if (nodes.some((n) => n.dragging) || changes.some((c) => c.type === "select"))
        // don't remember dragging a node or selecting an element
        dispatch(setNodes({ key: layoutKey, nodes }));
      else undoableDispatch(setNodes({ key: layoutKey, nodes }));
    },
    [layoutKey, dispatch, undoableDispatch],
  );

  const handleViewportChange: Diagram.DiagramProps["onViewportChange"] = useCallback(
    (vp) => dispatch(setViewport({ key: layoutKey, viewport: vp })),
    [layoutKey, dispatch],
  );

  const handleEditableChange: Diagram.DiagramProps["onEditableChange"] = useCallback(
    (cbk) => dispatch(setEditable({ key: layoutKey, editable: cbk })),
    [layoutKey, dispatch],
  );

  const handleSetFitViewOnResize = useCallback(
    (v: boolean) =>
      dispatch(setFitViewOnResize({ key: layoutKey, fitViewOnResize: v })),
    [layoutKey, dispatch],
  );

  const elRenderer = useCallback(
    (props: Diagram.SymbolProps) => (
      <SymbolRenderer layoutKey={layoutKey} dispatch={undoableDispatch} {...props} />
    ),
    [layoutKey, undoableDispatch],
  );

  const ref = useRef<HTMLDivElement>(null);

  const handleDrop = useCallback(
    ({ items, event }: Haul.OnDropProps): Haul.Item[] => {
      const valid = Haul.filterByType(HAUL_TYPE, items);
      if (ref.current == null || event == null) return valid;
      const region = box.construct(ref.current);
      valid.forEach(({ key, data }) => {
        const spec = Core.SYMBOLS[key];
        if (spec == null) return;
        const pos = xy.truncate(
          calculatePos(
            region,
            { x: event.clientX, y: event.clientY },
            viewportRef.current,
          ),
          0,
        );
        undoableDispatch(
          addElement({
            key: layoutKey,
            elKey: id.create(),
            node: { position: pos, zIndex: spec.zIndex },
            props: { key, ...spec.defaultProps(theme), ...(data ?? {}) },
          }),
        );
      });
      return valid;
    },
    [theme, undoableDispatch, layoutKey],
  );

  const dropProps = Haul.useDrop({
    type: "slate",
    key: layoutKey,
    canDrop: Haul.canDropOfType(HAUL_TYPE),
    onDrop: handleDrop,
  });

  const mode = useSelectViewportMode();
  const triggers = useMemo(() => Viewport.DEFAULT_TRIGGERS[mode], [mode]);

  Triggers.use({
    triggers: [
      ["Control", "V"],
      ["Control", "C"],
      ["Escape"],
      ["Control", "Z"],
      ["Control", "Shift", "Z"],
      ["Control", "A"],
    ],
    loose: true,
    region: ref,
    callback: useCallback(
      ({ triggers, cursor, stage }: Triggers.UseEvent) => {
        if (ref.current == null || stage !== "start") return;
        const region = box.construct(ref.current);
        const copy = triggers.some((t) => t.includes("C"));
        const isClear = triggers.some((t) => t.includes("Escape"));
        const isAll = triggers.some((t) => t.includes("A"));
        const isUndo =
          triggers.some((t) => t.includes("Z")) &&
          triggers.some((t) => t.includes("Control")) &&
          !triggers.some((t) => t.includes("Shift"));
        const isRedo =
          triggers.some((t) => t.includes("Z")) &&
          triggers.some((t) => t.includes("Control")) &&
          triggers.some((t) => t.includes("Shift"));
        const pos = calculatePos(region, cursor, viewportRef.current);
        if (copy) dispatch(copySelection({ pos }));
        else if (isClear) dispatch(clearSelection({ key: layoutKey }));
        else if (isUndo) undo();
        else if (isRedo) redo();
        else if (isAll) dispatch(selectAll({ key: layoutKey }));
        else undoableDispatch(pasteSelection({ pos, key: layoutKey }));
      },
      [layoutKey, undoableDispatch, undo, redo, dispatch],
    ),
  });

  const handleDoubleClick = useCallback(() => {
    if (!slate.editable) return;
    dispatch(
      Layout.setNavDrawerVisible({ windowKey, key: "visualization", value: true }),
    );
  }, [windowKey, slate.editable, dispatch]);

  const canEditSlate = useSelectHasPermission();

  return (
    <div
      ref={ref}
      onDoubleClick={handleDoubleClick}
      style={{ width: "inherit", height: "inherit", position: "relative" }}
    >
      <Core.Slate
        onViewportChange={handleViewportChange}
        edges={slate.edges}
        nodes={slate.nodes}
        // Turns out that setting the zoom value to 1 here doesn't have any negative
        // effects on the slate sizing and ensures that we position all the lines
        // in the correct place.
        viewport={{ ...slate.viewport, zoom: 1 }}
        onEdgesChange={handleEdgesChange}
        onNodesChange={handleNodesChange}
        onEditableChange={handleEditableChange}
        editable={slate.editable}
        triggers={triggers}
        onDoubleClick={handleDoubleClick}
        fitViewOnResize={slate.fitViewOnResize}
        setFitViewOnResize={handleSetFitViewOnResize}
        visible={visible}
        {...dropProps}
      >
        <Diagram.NodeRenderer>{elRenderer}</Diagram.NodeRenderer>
        <Diagram.Background />
        <Diagram.Controls>
          {canEditSlate && <Diagram.ToggleEditControl />}
          <Diagram.FitViewControl />
        </Diagram.Controls>
      </Core.Slate>
    </div>
  );
};

export const Slate: Layout.Renderer = ({ layoutKey, ...rest }) => {
  const loaded = useLoadRemote({
    name: "slate",
    targetVersion: ZERO_STATE.version,
    layoutKey,
    useSelectVersion,
    fetcher: async (client, layoutKey) => {
      try {
        const { key, graph } = await client.slates.retrieve(layoutKey);
        return translateSlateForward({ key, graph });
      } catch (__) {
        return { ...ZERO_STATE, key: layoutKey };
      }
    },
    actionCreator: create,
  });
  if (!loaded) return null;
  return <Loaded layoutKey={layoutKey} {...rest} />;
};
