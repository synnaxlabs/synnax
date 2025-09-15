// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch, type UnknownAction } from "@reduxjs/toolkit";
import { arc } from "@synnaxlabs/client";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import {
  Arc,
  Arc as Core,
  Button,
  Diagram,
  Flex,
  Haul,
  Icon,
  Menu as PMenu,
  Status,
  Theming,
  Triggers,
  useSyncedRef,
  Viewport,
} from "@synnaxlabs/pluto";
import { box, deep, id, uuid, xy } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo, useRef } from "react";
import { useDispatch } from "react-redux";
import { tr } from "zod/v4/locales";

import {
  select,
  useSelect,
  useSelectHasPermission,
  useSelectNodeProps,
  useSelectVersion,
  useSelectViewportMode,
} from "@/arc/selectors";
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
  setViewportMode,
  type State,
  ZERO_STATE,
} from "@/arc/slice";
import {
  translateGraphToconsole as translateGraphToConsole,
  translateGraphToServer,
} from "@/arc/types/translate";
import { TYPE } from "@/arc/types/v0";
import { useLoadRemote } from "@/hooks/useLoadRemote";
import { useUndoableDispatch } from "@/hooks/useUndoableDispatch";
import { Layout } from "@/layout";
import { type Selector } from "@/selector";
import { type RootState } from "@/store";

export const HAUL_TYPE = "arc-element";

interface SymbolRendererProps extends Diagram.SymbolProps {
  layoutKey: string;
  dispatch: Dispatch<UnknownAction>;
}

const StageRenderer = ({
  symbolKey,
  position,
  selected,
  draggable,
  dispatch,
  layoutKey,
}: SymbolRendererProps): ReactElement | null => {
  const props = useSelectNodeProps(layoutKey, symbolKey);
  const key = props?.key ?? "";
  const handleChange = useCallback(
    (props: object) => {
      if (key == null) return;
      dispatch(
        setElementProps({
          layoutKey,
          key: symbolKey,
          props: { key, ...props },
        }),
      );
    },
    [symbolKey, layoutKey, key, dispatch],
  );

  if (props == null) return null;

  const C = Core.Stage.REGISTRY[key];

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

interface StatusChipProps {
  layoutKey: string;
}

const StatusChip = ({ layoutKey }: StatusChipProps) => {
  const status = Status.useRetrieve({ key: layoutKey });
  return (
    <Status.Summary
      variant="disabled"
      message="Arc not deployed"
      {...status.data}
    />
  );
};

export const Loaded: Layout.Renderer = ({ layoutKey, visible }) => {
  const windowKey = useSelectWindowKey() as string;
  const arc = useSelect(layoutKey);
  const name = Layout.useSelectRequiredName(layoutKey);

  const dispatch = useDispatch();
  const selector = useCallback(
    (state: RootState) => select(state, layoutKey),
    [layoutKey],
  );
  const [undoableDispatch, undo, redo] = useUndoableDispatch<RootState, State>(
    selector,
    internalCreate,
    30, // roughly the right time needed to prevent actions that get dispatch automatically by Diagram.tsx, like setNodes immediately following addElement
  );

  const theme = Theming.use();
  const viewportRef = useSyncedRef(arc.graph.viewport);

  const canBeEditable = useSelectHasPermission();
  if (!canBeEditable && arc.graph.editable)
    dispatch(setEditable({ key: layoutKey, editable: false }));

  const handleEdgesChange: Diagram.DiagramProps["onEdgesChange"] = useCallback(
    (edges) => undoableDispatch(setEdges({ key: layoutKey, edges })),
    [layoutKey, undoableDispatch],
  );

  const handleNodesChange: Diagram.DiagramProps["onNodesChange"] = useCallback(
    (nodes, changes) => {
      if (
        // @ts-expect-error - Sometimes, the nodes do have dragging
        nodes.some((n) => n.dragging) ||
        changes.some((c) => c.type === "select")
      )
        // don't remember dragging a node or selecting an element
        dispatch(setNodes({ key: layoutKey, nodes }));
      else undoableDispatch(setNodes({ key: layoutKey, nodes }));
    },
    [layoutKey, dispatch, undoableDispatch],
  );

  const handleViewportChange: Diagram.DiagramProps["onViewportChange"] =
    useCallback(
      (vp) => dispatch(setViewport({ key: layoutKey, viewport: vp })),
      [layoutKey, dispatch],
    );

  const handleEditableChange: Diagram.DiagramProps["onEditableChange"] =
    useCallback(
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
      <StageRenderer
        layoutKey={layoutKey}
        dispatch={undoableDispatch}
        {...props}
      />
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
        const spec = Core.Stage.REGISTRY[key];
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
    type: "arc",
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
    if (!arc.graph.editable) return;
    dispatch(
      Layout.setNavDrawerVisible({
        windowKey,
        key: "visualization",
        value: true,
      }),
    );
  }, [windowKey, arc.graph.editable, dispatch]);

  const canEditArc = useSelectHasPermission();

  const viewportMode = useSelectViewportMode();

  const handleViewportModeChange = useCallback(
    (mode: Viewport.Mode) => dispatch(setViewportMode({ mode })),
    [layoutKey, dispatch],
  );

  const { update: create } = Arc.useCreate();

  return (
    <div
      ref={ref}
      onDoubleClick={handleDoubleClick}
      style={{ width: "inherit", height: "inherit", position: "relative" }}
    >
      <Core.Arc
        viewportMode={viewportMode}
        onViewportModeChange={handleViewportModeChange}
        onViewportChange={handleViewportChange}
        edges={arc.graph.edges}
        nodes={arc.graph.nodes}
        // Turns out that setting the zoom value to 1 here doesn't have any negative
        // effects on the arc sizing and ensures that we position all the lines
        // in the correct place.
        viewport={{ ...arc.graph.viewport, zoom: 1 }}
        onEdgesChange={handleEdgesChange}
        onNodesChange={handleNodesChange}
        onEditableChange={handleEditableChange}
        editable={arc.graph.editable}
        triggers={triggers}
        onDoubleClick={handleDoubleClick}
        fitViewOnResize={arc.graph.fitViewOnResize}
        setFitViewOnResize={handleSetFitViewOnResize}
        visible={visible}
        {...dropProps}
      >
        <Diagram.NodeRenderer>{elRenderer}</Diagram.NodeRenderer>
        <Diagram.Background />
        <Diagram.Controls>
          {canEditArc && <Diagram.ToggleEditControl />}
          <Diagram.FitViewControl />
        </Diagram.Controls>
      </Core.Arc>
      <Flex.Box
        style={{
          padding: "2rem",
          position: "absolute",
          bottom: 0,
          right: 0,
          width: 500,
        }}
        justify="end"
        grow
      >
        <Flex.Box
          x
          background={1}
          style={{ padding: "2rem" }}
          bordered
          borderColor={5}
          grow
          rounded={2}
          justify="between"
        >
          <StatusChip layoutKey={layoutKey} />
          <Button.Button
            onClick={() => {
              create({
                key: arc.key,
                name,
                graph: translateGraphToServer(arc.graph),
                text: { contents: "" },
              });
            }}
          >
            <Icon.Play />
            Deploy
          </Button.Button>
        </Flex.Box>
      </Flex.Box>
    </div>
  );
};

export const EDIT_LAYOUT_TYPE = "arc_editor";
export type EditLayoutType = typeof EDIT_LAYOUT_TYPE;

export const SELECTABLE: Selector.Selectable = {
  key: EDIT_LAYOUT_TYPE,
  title: "Arc Automation",
  icon: <Icon.Arc />,
  create: async ({ layoutKey, rename }) => {
    const name = await rename({}, { icon: "Arc", name: "Arc.Create" });
    if (name == null) return null;
    return createEditor({ key: layoutKey, name });
  },
};

export type CreateArg = Partial<State> & Partial<Layout.BaseState>;

export const createEditor =
  (initial: CreateArg = {}): Layout.Creator =>
  ({ dispatch }) => {
    const {
      name = "Arc Editor",
      location = "mosaic",
      window,
      tab,
      ...rest
    } = initial;
    const key = arc.keyZ.safeParse(initial.key).data ?? uuid.create();
    dispatch(internalCreate({ ...deep.copy(ZERO_STATE), ...rest, key }));
    return {
      key,
      location,
      name,
      icon: "Arc",
      type: EDIT_LAYOUT_TYPE,
      window: { navTop: true, showTitle: true },
      tab,
    };
  };

export const Editor: Layout.Renderer = ({ layoutKey, ...rest }) => {
  const loaded = useLoadRemote({
    name: "arc",
    targetVersion: ZERO_STATE.version,
    layoutKey,
    useSelectVersion,
    fetcher: async (client, layoutKey) => {
      try {
        const arc = await client.arcs.retrieve({ key: layoutKey });
        const graph = translateGraphToConsole(arc.graph);
        console.log(graph);
        const state: State = {
          version: "0.0.0",
          key: arc.key,
          type: TYPE,
          remoteCreated: false,
          graph,
        };
        return state;
      } catch (__) {
        return { ...ZERO_STATE, key: layoutKey };
      }
    },
    actionCreator: internalCreate,
  });
  if (!loaded) return null;
  return <Loaded layoutKey={layoutKey} {...rest} />;
};
