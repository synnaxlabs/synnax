// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch, type UnknownAction } from "@reduxjs/toolkit";
import { arc, type rack, task } from "@synnaxlabs/client";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import {
  Access,
  Arc,
  Arc as Core,
  Button,
  Diagram,
  Flex,
  Haul,
  Icon,
  Menu as PMenu,
  Rack,
  Status,
  Synnax,
  Task,
  Theming,
  useSyncedRef,
  Viewport,
} from "@synnaxlabs/pluto";
import { box, deep, id, uuid, xy } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo, useRef, useState } from "react";
import { useDispatch } from "react-redux";
import { z } from "zod";

import {
  select,
  useSelect,
  useSelectNodeProps,
  useSelectVersion,
  useSelectViewportMode,
} from "@/arc/selectors";
import { createArcTask, deleteArcTask, useArcTask } from "@/arc/task";
import {
  addElement,
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
import { translateGraphToConsole, translateGraphToServer } from "@/arc/types/translate";
import { TYPE } from "@/arc/types/v0";
import { Controls as CoreControls } from "@/components";
import { createLoadRemote } from "@/hooks/useLoadRemote";
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
    [symbolKey, layoutKey, key, key, dispatch],
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

interface ControlsProps {
  arc: State;
}

const statusDetailsSchema = z.object({
  running: z.boolean(),
});

const { useRetrieve: useRetrieveStatus } = Status.createRetrieve(statusDetailsSchema);

export const Controls = ({ arc }: ControlsProps) => {
  const client = Synnax.use();
  const name = Layout.useSelectRequiredName(arc.key);
  const arcTask = useArcTask(arc.key);
  const [selectedRack, setSelectedRack] = useState<rack.Key | undefined>(
    arcTask != null ? task.rackKey(arcTask.key) : undefined,
  );
  const addStatus = Status.useAdder();

  // Get task status
  const taskStatus = useRetrieveStatus(
    { key: arcTask?.key?.toString() ?? "" },
    { addStatusOnFailure: false },
  );
  const isRunning = taskStatus.data?.details?.running ?? false;

  const { update: runCommand } = Task.useCommand();

  const handleDeploy = useCallback(async () => {
    if (client == null || selectedRack === undefined) return;

    try {
      let taskKey = arcTask?.key;
      const currentTaskRack = arcTask != null ? task.rackKey(arcTask.key) : null;

      // If task exists on different rack, delete it first
      if (taskKey != null && currentTaskRack !== selectedRack) {
        // Stop then delete the old task
        if (isRunning) {
          await runCommand([{ task: taskKey, type: "stop" }]);
        }
        await deleteArcTask(client, taskKey);
        taskKey = undefined;
      }

      // Create task if it doesn't exist
      if (taskKey == null) {
        const newTask = await createArcTask(client, arc.key, selectedRack, name);
        taskKey = newTask.key;
      }

      // Send start/stop command
      await runCommand([{ task: taskKey, type: isRunning ? "stop" : "start" }]);
    } catch (e) {
      addStatus({
        variant: "error",
        message: `Failed to ${isRunning ? "stop" : "start"} Arc task`,
        description: e instanceof Error ? e.message : String(e),
      });
    }
  }, [client, selectedRack, arcTask, arc.key, name, isRunning, runCommand, addStatus]);

  return (
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
        gap="medium"
      >
        <Flex.Box x gap="small" align="center" grow>
          <Rack.SelectSingle
            value={selectedRack}
            onChange={setSelectedRack}
            allowNone
            style={{ minWidth: 150 }}
          />
          <Status.Summary
            variant="disabled"
            message="Not deployed"
            status={taskStatus.data}
          />
        </Flex.Box>
        <Button.Button
          onClick={handleDeploy}
          variant="filled"
          disabled={selectedRack === undefined}
        >
          {isRunning ? <Icon.Pause /> : <Icon.Play />}
          {isRunning ? "Stop" : "Start"}
        </Button.Button>
      </Flex.Box>
    </Flex.Box>
  );
};

export const Loaded: Layout.Renderer = ({ layoutKey, visible }) => {
  const windowKey = useSelectWindowKey() as string;
  const state = useSelect(layoutKey);

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
  const viewportRef = useSyncedRef(state.graph.viewport);
  const hasEditPermission = Access.useUpdateGranted(arc.ontologyID(layoutKey));
  const canEdit = hasEditPermission && state.graph.editable;

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
      <StageRenderer layoutKey={layoutKey} dispatch={undoableDispatch} {...props} />
    ),
    [layoutKey, undoableDispatch],
  );

  const ref = useRef<HTMLDivElement>(null);

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
      if (ref.current == null || event == null) return valid;
      valid.forEach(({ key, data }) => {
        const spec = Core.Stage.REGISTRY[key];
        if (spec == null) return;
        const pos = xy.truncate(calculateCursorPosition(event), 0);
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

  const handleDoubleClick = useCallback(() => {
    if (!state.graph.editable) return;
    dispatch(
      Layout.setNavDrawerVisible({
        windowKey,
        key: "visualization",
        value: true,
      }),
    );
  }, [windowKey, state.graph.editable, dispatch]);

  const viewportMode = useSelectViewportMode();

  const handleViewportModeChange = useCallback(
    (mode: Viewport.Mode) => dispatch(setViewportMode({ mode })),
    [layoutKey, dispatch],
  );

  const handleCopySelection = useCallback(
    (cursor: xy.XY) =>
      dispatch(copySelection({ pos: calculateCursorPosition(cursor) })),
    [dispatch, calculateCursorPosition],
  );

  const handlePasteSelection = useCallback(
    (cursor: xy.XY) =>
      dispatch(
        pasteSelection({
          pos: calculateCursorPosition(cursor),
          key: layoutKey,
        }),
      ),
    [dispatch, calculateCursorPosition, layoutKey],
  );

  const handleSelectAll = useCallback(
    () => dispatch(selectAll({ key: layoutKey })),
    [dispatch, layoutKey],
  );

  const handleClearSelection = useCallback(
    () => dispatch(clearSelection({ key: layoutKey })),
    [dispatch, layoutKey],
  );

  Diagram.useTriggers({
    onCopy: handleCopySelection,
    onPaste: handlePasteSelection,
    onSelectAll: handleSelectAll,
    onClear: handleClearSelection,
    onUndo: undo,
    onRedo: redo,
    region: ref,
  });

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
        edges={state.graph.edges}
        nodes={state.graph.nodes}
        // Turns out that setting the zoom value to 1 here doesn't have any negative
        // effects on the arc sizing and ensures that we position all the lines
        // in the correct place.
        viewport={{ ...state.graph.viewport, zoom: 1 }}
        onEdgesChange={handleEdgesChange}
        onNodesChange={handleNodesChange}
        onEditableChange={handleEditableChange}
        editable={canEdit}
        triggers={triggers}
        onDoubleClick={handleDoubleClick}
        fitViewOnResize={state.graph.fitViewOnResize}
        setFitViewOnResize={handleSetFitViewOnResize}
        visible={visible}
        {...dropProps}
      >
        <Diagram.NodeRenderer>{elRenderer}</Diagram.NodeRenderer>
        <Diagram.Background />
        <CoreControls x>
          <Diagram.FitViewControl />
          {hasEditPermission && <Diagram.ToggleEditControl />}
        </CoreControls>
      </Core.Arc>
      <Controls arc={state} />
    </div>
  );
};

export const LAYOUT_TYPE = "arc_editor";
export type LayoutType = typeof LAYOUT_TYPE;

export const SELECTABLE: Selector.Selectable = {
  key: LAYOUT_TYPE,
  title: "Arc Automation",
  icon: <Icon.Arc />,
  useVisible: () => Access.useUpdateGranted(arc.TYPE_ONTOLOGY_ID),
  create: async ({ layoutKey, rename }) => {
    const name = await rename({}, { icon: "Arc", name: "Arc.Create" });
    if (name == null) return null;
    return create({ key: layoutKey, name });
  },
};

export type CreateArg = Partial<State> & Partial<Layout.BaseState>;

export const create =
  (initial: CreateArg = {}): Layout.Creator =>
  ({ dispatch }) => {
    const { name = "Arc Editor", location = "mosaic", window, tab, ...rest } = initial;
    const key = arc.keyZ.safeParse(initial.key).data ?? uuid.create();
    dispatch(internalCreate({ ...deep.copy(ZERO_STATE), ...rest, key }));
    return {
      key,
      location,
      name,
      icon: "Arc",
      type: LAYOUT_TYPE,
      window: { navTop: true, showTitle: true },
      tab,
    };
  };

export const useLoadRemote = createLoadRemote<arc.Arc>({
  useRetrieve: Core.useRetrieveObservable,
  targetVersion: ZERO_STATE.version,
  useSelectVersion,
  actionCreator: (v) =>
    internalCreate({
      version: "0.0.0",
      key: v.key,
      type: TYPE,
      remoteCreated: false,
      graph: translateGraphToConsole(v.graph),
      text: { raw: "" },
    }),
});

export const Editor: Layout.Renderer = ({ layoutKey, ...rest }) => {
  const arc = useLoadRemote(layoutKey);
  if (arc == null) return null;
  return <Loaded layoutKey={layoutKey} {...rest} />;
};
