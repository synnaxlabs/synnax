// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch, type UnknownAction } from "@reduxjs/toolkit";
import { schematic } from "@synnaxlabs/client";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import {
  Access,
  Button,
  Control,
  Diagram,
  Flex,
  Haul,
  Icon,
  Menu as PMenu,
  Schematic as Base,
  Theming,
  usePrevious,
  User,
  useSyncedRef,
  Viewport,
} from "@synnaxlabs/pluto";
import { box, deep, location, type sticky, uuid, xy } from "@synnaxlabs/x";
import {
  type ReactElement,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useDispatch } from "react-redux";

import { Controls } from "@/components";
import { createLoadRemote } from "@/hooks/useLoadRemote";
import { Layout } from "@/layout";
import {
  useSelectLegendVisible,
  useSelectRequired,
  useSelectRequiredViewportMode,
  useSelectVersion,
} from "@/schematic/selectors";
import {
  internalCreate,
  setControlStatus,
  setEditable,
  setFitViewOnResize,
  setLegend,
  setRemoteCreated,
  setViewport,
  setViewportMode,
  type State,
  ZERO_STATE,
} from "@/schematic/slice";
import { stateFromRemote } from "@/schematic/remote";
import { useAddSymbol } from "@/schematic/symbols/useAddSymbol";
import { Selector } from "@/selector";
import { type RootState } from "@/store";
import { Workspace } from "@/workspace";

export const HAUL_TYPE = "schematic-element";

interface ControlToggleButtonProps {
  control: Control.Status;
}

const ControlToggleButton = ({ control }: ControlToggleButtonProps): ReactElement => {
  const { acquire, release } = Control.useContext();
  const handleChange = useCallback(
    (v: boolean) => (v ? acquire() : release()),
    [acquire, release],
  );
  return (
    <Button.Toggle
      value={control === "acquired"}
      onChange={handleChange}
      tooltipLocation={location.BOTTOM_LEFT}
      size="small"
      tooltip={`${control === "acquired" ? "Release" : "Acquire"} control`}
    >
      <Icon.Circle />
    </Button.Toggle>
  );
};

const useSyncComponent = Workspace.createSyncComponent(
  "Schematic",
  async ({ key, workspace, store, fluxStore, client }) => {
    const storeState = store.getState();
    if (
      !Access.updateGranted({ id: schematic.ontologyID(key), store: fluxStore, client })
    )
      return;
    const state = storeState.schematic.schematics[key];
    if (state == null) return;
    const layout = Layout.selectRequired(storeState, key);
    if (state.snapshot) {
      await client.schematics.rename(key, layout.name);
      return;
    }
    if (!state.remoteCreated) {
      store.dispatch(setRemoteCreated({ key }));
      // First sync: create the schematic on the server with initial state
      await client.schematics.create(workspace, {
        key,
        name: layout.name,
        nodes: state.nodes ?? [],
        edges: state.edges ?? [],
        props: state.props ?? {},
        viewport: state.viewport,
        legend: state.legend,
        editable: state.editable,
        fitViewOnResize: state.fitViewOnResize,
        authority: state.authority,
        snapshot: state.snapshot,
      });
      return;
    }
    // Subsequent syncs: only update session state fields.
    // Document mutations (nodes/edges/props) go through Flux action dispatch.
    await client.schematics.create(workspace, {
      key,
      name: layout.name,
      viewport: state.viewport,
      legend: state.legend,
      editable: state.editable,
      fitViewOnResize: state.fitViewOnResize,
      authority: state.authority,
    });
  },
);

export const ContextMenu: Layout.ContextMenuRenderer = ({ layoutKey }) => (
  <PMenu.Menu level="small" gap="small">
    <Layout.MenuItems layoutKey={layoutKey} />
  </PMenu.Menu>
);

export const Loaded: Layout.Renderer = ({ layoutKey, visible }) => {
  const windowKey = useSelectWindowKey() as string;
  const { name } = Layout.useSelectRequired(layoutKey);
  const { data: user } = User.useRetrieve({}, { addStatusOnFailure: false });
  const username = user?.username ?? "";
  const controlName = username.length > 0 ? `${name} (${username})` : name;
  const state = useSelectRequired(layoutKey);
  const legendVisible = useSelectLegendVisible(layoutKey);
  const dispatch = useDispatch();
  const syncDispatch = useSyncComponent(layoutKey);

  const viewportRef = useSyncedRef(state.viewport);

  const prevName = usePrevious(name);
  useEffect(() => {
    if (prevName !== name) syncDispatch(Layout.rename({ key: layoutKey, name }));
  }, [name, prevName, layoutKey, syncDispatch]);

  const hasEditPermission =
    Access.useUpdateGranted(schematic.ontologyID(layoutKey)) && !state.snapshot;
  const canEdit = hasEditPermission && state.editable;

  // Selection state (session, per-window)
  const [selectedNodes, setSelectedNodes] = useState<string[]>([]);
  const [selectedEdges, setSelectedEdges] = useState<string[]>([]);

  const handleSelectionChange = useCallback(
    (nodes: string[], edges: string[]) => {
      setSelectedNodes(nodes);
      setSelectedEdges(edges);
    },
    [],
  );

  const handleViewportChange = useCallback(
    (vp: Diagram.Viewport) => syncDispatch(setViewport({ key: layoutKey, viewport: vp })),
    [layoutKey, syncDispatch],
  );

  const handleEditableChange = useCallback(
    (cbk: boolean) => syncDispatch(setEditable({ key: layoutKey, editable: cbk })),
    [layoutKey, syncDispatch],
  );

  const handleSetFitViewOnResize = useCallback(
    (v: boolean) =>
      syncDispatch(setFitViewOnResize({ key: layoutKey, fitViewOnResize: v })),
    [layoutKey, syncDispatch],
  );

  const handleControlStatusChange = useCallback(
    (control: Control.Status) =>
      syncDispatch(setControlStatus({ key: layoutKey, control })),
    [layoutKey, syncDispatch],
  );

  const ref = useRef<HTMLDivElement>(null);

  const handleAddElement = useAddSymbol(layoutKey);

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
        const spec = Base.Symbol.REGISTRY[key as Base.Symbol.Variant];
        if (spec == null) return;
        const pos = xy.truncate(calculateCursorPosition(event), 0);
        handleAddElement(key.toString(), pos, data);
      });
      return valid;
    },
    [handleAddElement, calculateCursorPosition],
  );

  const dropProps = Haul.useDrop({
    type: "Schematic",
    key: layoutKey,
    canDrop: Haul.canDropOfType(HAUL_TYPE),
    onDrop: handleDrop,
  });

  const mode = useSelectRequiredViewportMode(layoutKey);
  const triggers = useMemo(() => Viewport.DEFAULT_TRIGGERS[mode], [mode]);

  const handleDoubleClick = useCallback(() => {
    if (!state.editable) return;
    syncDispatch(
      Layout.setNavDrawerVisible({
        windowKey,
        key: "visualization",
        value: true,
      }),
    );
  }, [windowKey, state.editable, syncDispatch]);

  const [legendPosition, setLegendPosition] = useState<sticky.XY>(
    state.legend.position,
  );

  const storeLegendPosition = useCallback(
    (position: sticky.XY) =>
      syncDispatch(setLegend({ key: layoutKey, legend: { position } })),
    [layoutKey, syncDispatch],
  );

  const handleLegendPositionChange = useCallback(
    (position: sticky.XY) => {
      setLegendPosition(position);
      storeLegendPosition(position);
    },
    [storeLegendPosition, setLegendPosition],
  );

  const handleLegendColorsChange = useCallback(
    (colors: Record<string, string>) =>
      syncDispatch(setLegend({ key: layoutKey, legend: { colors } })),
    [layoutKey, syncDispatch],
  );

  const handleViewportModeChange = useCallback(
    (mode: Viewport.Mode) => dispatch(setViewportMode({ key: layoutKey, mode })),
    [dispatch, layoutKey],
  );

  const handleSelectAll = useCallback(
    () => {
      // Read all node/edge keys from Flux and set them as selected
      // For now this is a no-op until we wire it properly
    },
    [layoutKey],
  );

  const handleClearSelection = useCallback(
    () => handleSelectionChange([], []),
    [handleSelectionChange],
  );

  Diagram.useTriggers({
    onCopy: () => {},
    onPaste: () => {},
    onSelectAll: handleSelectAll,
    onClear: handleClearSelection,
    onUndo: () => {},
    onRedo: () => {},
    region: ref,
  });

  return (
    <div
      ref={ref}
      onDoubleClick={handleDoubleClick}
      style={{ width: "inherit", height: "inherit", position: "relative" }}
    >
      <Control.Controller
        name={controlName}
        authority={state.authority}
        onStatusChange={handleControlStatusChange}
      >
        <Base.Schematic
          schematicKey={layoutKey}
          selectedNodes={selectedNodes}
          selectedEdges={selectedEdges}
          onSelectionChange={handleSelectionChange}
          onViewportChange={handleViewportChange}
          viewportMode={mode}
          onViewportModeChange={handleViewportModeChange}
          viewport={{ ...state.viewport, zoom: 1 }}
          onEditableChange={handleEditableChange}
          editable={canEdit}
          triggers={triggers}
          onDoubleClick={handleDoubleClick}
          fitViewOnResize={state.fitViewOnResize}
          setFitViewOnResize={handleSetFitViewOnResize}
          visible={visible}
          {...dropProps}
        >
          <Diagram.NodeRenderer>
            {(props: Diagram.SymbolProps) => (
              <SymbolRenderer layoutKey={layoutKey} {...props} />
            )}
          </Diagram.NodeRenderer>
          <Diagram.Background />
          <Controls x>
            <Diagram.SelectViewportModeControl />
            <Diagram.FitViewControl />
            <Flex.Box x pack>
              {hasEditPermission && (
                <Diagram.ToggleEditControl disabled={state.control === "acquired"} />
              )}
              {!state.snapshot && <ControlToggleButton control={state.control} />}
            </Flex.Box>
          </Controls>
        </Base.Schematic>
        {legendVisible && (
          <Control.Legend
            position={legendPosition}
            onPositionChange={handleLegendPositionChange}
            colors={state.legend.colors}
            onColorsChange={handleLegendColorsChange}
            allowVisibleChange={false}
          />
        )}
      </Control.Controller>
    </div>
  );
};

interface SymbolRendererProps extends Diagram.SymbolProps {
  layoutKey: string;
}

const SymbolRenderer = ({
  symbolKey,
  position,
  selected,
  layoutKey,
}: SymbolRendererProps): ReactElement | null => {
  const { data: doc } = Base.useRetrieve({ key: layoutKey });
  const { update: fluxDispatch } = Base.useDispatch();
  const nodeProps = doc?.props?.[symbolKey] as Record<string, unknown> | undefined;
  const key = (nodeProps?.key ?? null) as Base.Symbol.Variant | null;

  const handleChange = useCallback(
    (props: object) => {
      if (key == null) return;
      fluxDispatch({
        key: layoutKey,
        actions: schematic.setNodeProps({
          key: symbolKey,
          props: { key, ...props },
        }),
      });
    },
    [symbolKey, layoutKey, key, fluxDispatch],
  );

  if (nodeProps == null || key == null) return null;

  const C = Base.Symbol.REGISTRY[key];
  if (C == null) throw new Error(`Symbol ${key} not found`);

  const { key: _, ...rest } = nodeProps;

  return (
    <C.Symbol
      key={key}
      id={symbolKey}
      symbolKey={symbolKey}
      position={position}
      selected={selected}
      onChange={handleChange}
      {...rest}
    />
  );
};

const useLoadRemote = createLoadRemote<schematic.Schematic>({
  useRetrieve: Base.useRetrieveObservable,
  targetVersion: ZERO_STATE.version,
  useSelectVersion,
  actionCreator: (v) => internalCreate(stateFromRemote(v)),
});

export const SchematicComponent: Layout.Renderer = ({ layoutKey, ...rest }) => {
  const loaded = useLoadRemote(layoutKey);
  if (loaded == null) return null;
  return <Loaded layoutKey={layoutKey} {...rest} />;
};

export const LAYOUT_TYPE = "schematic";
export type LayoutType = typeof LAYOUT_TYPE;

export const Selectable: Selector.Selectable = ({ layoutKey, onPlace }) => {
  const visible = Access.useUpdateGranted(schematic.TYPE_ONTOLOGY_ID);
  const handleClick = useCallback(() => {
    onPlace(create({ key: layoutKey }));
  }, [onPlace, layoutKey]);

  if (!visible) return null;

  return (
    <Selector.Item
      key={LAYOUT_TYPE}
      title="Schematic"
      icon={<Icon.Schematic />}
      onClick={handleClick}
    />
  );
};
Selectable.type = LAYOUT_TYPE;
Selectable.useVisible = () => Access.useUpdateGranted(schematic.TYPE_ONTOLOGY_ID);

export type CreateArg = Partial<State> & Partial<Layout.BaseState>;

export const create =
  (initial: CreateArg = {}): Layout.Creator =>
  ({ dispatch }) => {
    const canEditSchematic = true;
    const { name = "Schematic", location = "mosaic", window, tab, ...rest } = initial;
    if (!canEditSchematic && tab?.editable) tab.editable = false;
    const key = schematic.keyZ.safeParse(initial.key).data ?? uuid.create();
    dispatch(internalCreate({ ...deep.copy(ZERO_STATE), ...rest, key }));
    return {
      key,
      location,
      name,
      icon: "Schematic",
      type: LAYOUT_TYPE,
      window: { navTop: true, showTitle: true },
      tab,
    };
  };
