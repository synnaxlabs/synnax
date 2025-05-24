// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Dispatch,
  type PayloadAction,
  type UnknownAction,
} from "@reduxjs/toolkit";
import { slate } from "@synnaxlabs/client";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Icon } from "@synnaxlabs/media";
import {
  Diagram,
  Haul,
  Menu as PMenu,
  Slate as Core,
  Theming,
  Triggers,
  usePrevious,
  useSyncedRef,
  Viewport,
} from "@synnaxlabs/pluto";
import { box, deep, id, xy } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect, useMemo, useRef } from "react";
import { v4 as uuid } from "uuid";

import { useLoadRemote } from "@/hooks/useLoadRemote";
import { useUndoableDispatch } from "@/hooks/useUndoableDispatch";
import { Layout } from "@/layout";
import { type Selector } from "@/selector";
import {
  select,
  selectHasPermission,
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
  internalCreate,
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
import { type RootState } from "@/store";
import { Workspace } from "@/workspace";

interface SyncPayload {
  key?: string;
}

export const HAUL_TYPE = "slate-element";

const useSyncComponent = (
  layoutKey: string,
  dispatch?: Dispatch<PayloadAction<SyncPayload>>,
): Dispatch<PayloadAction<SyncPayload>> =>
  Workspace.useSyncComponent<SyncPayload>(
    "slate",
    layoutKey,
    async (ws, store, client) => {
      // const storeState = store.getState();
      // if (!selectHasPermission(storeState)) return;
      // const data = select(storeState, layoutKey);
      // if (data == null) return;
      // const layout = Layout.selectRequired(storeState, layoutKey);
      // if (data.snapshot) {
      //   await client.workspaces.slate.rename(layoutKey, layout.name);
      //   return;
      // }
      // const setData = { ...data, key: undefined };
      // if (!data.remoteCreated) store.dispatch(setRemoteCreated({ key: layoutKey }));
      // await client.workspaces.slate.create(ws, {
      //   key: layoutKey,
      //   name: layout.name,
      //   data: setData,
      // });
    },
    dispatch,
  );

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
  const key = props?.key;
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

  const C = Core.REGISTRY[key as Core.Variant];

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
  <PMenu.Menu level="small" iconSpacing="small">
    <Layout.MenuItems layoutKey={layoutKey} />
  </PMenu.Menu>
);

export const Loaded: Layout.Renderer = ({ layoutKey, visible }) => {
  const windowKey = useSelectWindowKey() as string;
  const { name } = Layout.useSelectRequired(layoutKey);
  const slate = useSelect(layoutKey);

  const dispatch = useSyncComponent(layoutKey);
  const selector = useCallback(
    (state: RootState) => select(state, layoutKey),
    [layoutKey],
  );
  const [undoableDispatch_, undo, redo] = useUndoableDispatch<RootState, State>(
    selector,
    internalCreate,
    30, // roughly the right time needed to prevent actions that get dispatch automatically by Diagram.tsx, like setNodes immediately following addElement
  );
  const undoableDispatch = useSyncComponent(layoutKey, undoableDispatch_);

  const theme = Theming.use();
  const viewportRef = useSyncedRef(slate.viewport);

  const prevName = usePrevious(name);
  useEffect(() => {
    if (prevName !== name) dispatch(Layout.rename({ key: layoutKey, name }));
  }, [name, prevName, layoutKey, dispatch]);

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
        const spec = Core.REGISTRY[key as Core.Variant];
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
      <Diagram.Diagram
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
      </Diagram.Diagram>
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
      const { key, graph } = await client.slates.retrieve(layoutKey);
      return { key, graph } as State;
    },
    actionCreator: internalCreate,
  });
  if (!loaded) return null;
  return <Loaded layoutKey={layoutKey} {...rest} />;
};

export const LAYOUT_TYPE = "slate";
export type LayoutType = typeof LAYOUT_TYPE;

export const SELECTABLE: Selector.Selectable = {
  key: LAYOUT_TYPE,
  title: "slate",
  icon: <Icon.Slate />,
  create: async ({ layoutKey }) => create({ key: layoutKey }),
};

export type CreateArg = Partial<State> & Partial<Layout.BaseState>;

export const create =
  (initial: CreateArg = {}): Layout.Creator =>
  ({ dispatch, store }) => {
    const canEditSlate = selectHasPermission(store.getState());
    const { name = "slate", location = "mosaic", window, tab, ...rest } = initial;
    if (!canEditSlate && tab?.editable) tab.editable = false;
    const key = slate.keyZ.safeParse(initial.key).data ?? uuid();
    dispatch(internalCreate({ ...deep.copy(ZERO_STATE), ...rest, key }));
    return {
      key,
      location,
      name,
      icon: "Slate",
      type: LAYOUT_TYPE,
      window: { navTop: true, showTitle: true },
      tab,
    };
  };
