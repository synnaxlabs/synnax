// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import {
  Access,
  Button,
  Control,
  Diagram,
  Flex,
  Flux,
  Icon,
  type Pluto,
  Schematic as Base,
  Status,
  Synnax,
  Viewport,
} from "@synnaxlabs/pluto";
import { location, type sticky, uuid } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo, useRef, useState } from "react";
import { useDispatch, useStore } from "react-redux";

import { ContextMenu as CContextMenu, Controls } from "@/components";
import { Layout } from "@/layout";
import { Controller } from "@/schematic/Controller";
import {
  selectOptional,
  useSelectEditable,
  useSelectLegend,
  useSelectLegendVisible,
  useSelectSelected,
} from "@/schematic/selectors";
import {
  internalCreate,
  setEditable,
  setFitViewOnResize,
  setSelected,
  setViewport,
} from "@/schematic/slice";
import { useAutoUpload } from "@/schematic/useUpload";
import { Selector } from "@/selector";
import { type RootState } from "@/store";

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

type SchematicRetriever = (key: string) => Promise<schematic.Schematic>;

const navigateToLinkedSchematic = async (
  retrieve: SchematicRetriever,
  page: string,
  placeLayout: Layout.Placer,
): Promise<void> => {
  const s = await retrieve(page);
  placeLayout(create({ key: s.key, name: s.name }));
};

type NodeClickHandler = (nodeId: string, dblClick: boolean) => void;

const useHandleNodeClickAction = (layoutKey: string): NodeClickHandler => {
  const store = useStore<RootState>();
  const client = Synnax.use();
  const fluxStore = Flux.useStore<Pluto.FluxStore>();
  const retrieve: SchematicRetriever | null = useMemo(
    () =>
      client != null
        ? (key: string) =>
            Base.retrieveSingle({ store: fluxStore, client, query: { key } })
        : null,
    [fluxStore, client],
  );
  const handleError = Status.useErrorHandler();
  const placeLayout = Layout.usePlacer();

  return useCallback(
    (nodeId: string, dblClick: boolean) => {
      const storeState = store.getState();
      const ui = selectOptional(storeState, layoutKey);
      if (ui == null || ui.editable || retrieve == null) return;
      const props = fluxStore.schematics.get(layoutKey)?.props?.[nodeId] as
        | Record<string, unknown>
        | undefined;
      if (
        props?.variant !== "offPageReference" ||
        typeof props.page !== "string" ||
        props.page.length === 0
      )
        return;
      const dblClickNav = props.dblClickNav !== false;
      if (dblClick !== dblClickNav) return;
      const { page } = props;
      const labelObj = props.label as { label?: string } | undefined;
      const label = labelObj?.label;
      const name = label != null && label.length > 0 ? label : "Referenced schematic";
      handleError(
        () => navigateToLinkedSchematic(retrieve, page, placeLayout),
        `Schematic "${name}" not found`,
      );
    },
    [store, layoutKey, retrieve, placeLayout, handleError, fluxStore],
  );
};

export const ContextMenu: Layout.ContextMenuRenderer = ({ layoutKey }) => (
  <CContextMenu.Menu>
    <Layout.MenuItems layoutKey={layoutKey} />
  </CContextMenu.Menu>
);

export const Loaded: Layout.Renderer = ({ layoutKey, visible }) => {
  const windowKey = useSelectWindowKey() as string;
  const { name } = Layout.useSelectRequired(layoutKey);
  const { data: doc } = Base.useRetrieve({ key: layoutKey });
  const dispatch = useDispatch();
  const editable = useSelectEditable(layoutKey);
  const legend = useSelectLegend(layoutKey);
  const legendVisible = useSelectLegendVisible(layoutKey);
  const selected = useSelectSelected(layoutKey);

  useAutoUpload(layoutKey, name);

  const authority = Base.useSelectAuthority({ key: layoutKey }) ?? 1;
  const control =
    (doc?.snapshot ?? false) ? "released" : ("released" as Control.Status);
  const hasUpdatePermission =
    Access.useUpdateGranted(schematic.ontologyID(layoutKey)) &&
    !(doc?.snapshot ?? false);
  const canEdit = hasUpdatePermission && editable;
  const snapshot = doc?.snapshot ?? false;

  const handleSelectionChange = useCallback(
    (next: string[]) => dispatch(setSelected({ key: layoutKey, selected: next })),
    [dispatch, layoutKey],
  );

  const handleViewportChange = useCallback(
    (vp: Diagram.Viewport) => dispatch(setViewport({ key: layoutKey, viewport: vp })),
    [dispatch, layoutKey],
  );

  const handleEditableChange = useCallback(
    (v: boolean) => dispatch(setEditable({ key: layoutKey, editable: v })),
    [dispatch, layoutKey],
  );

  const handleFitViewOnResizeChange = useCallback(
    (v: boolean) =>
      dispatch(setFitViewOnResize({ key: layoutKey, fitViewOnResize: v })),
    [dispatch, layoutKey],
  );

  const [mode, setMode] = useState<Viewport.Mode>("select");
  const triggers = useMemo(() => Viewport.DEFAULT_TRIGGERS[mode], [mode]);

  const handleDoubleClick = useCallback(() => {
    if (!editable) return;
    dispatch(
      Layout.setNavDrawerVisible({
        windowKey,
        key: "visualization",
        value: true,
      }),
    );
  }, [windowKey, editable, dispatch]);

  const handleNodeClickAction = useHandleNodeClickAction(layoutKey);
  const handleNodeClick = useCallback(
    (_event: React.MouseEvent, node: { id: string }) =>
      handleNodeClickAction(node.id, false),
    [handleNodeClickAction],
  );
  const handleNodeDoubleClick = useCallback(
    (_event: React.MouseEvent, node: { id: string }) =>
      handleNodeClickAction(node.id, true),
    [handleNodeClickAction],
  );

  const [legendPosition, setLegendPosition] = useState<sticky.XY>(legend.position);
  const ref = useRef<HTMLDivElement>(null);

  return (
    <Controller resourceKey={layoutKey} authority={authority}>
      <Base.Schematic
        ref={ref}
        resourceKey={layoutKey}
        selected={selected}
        onSelectionChange={handleSelectionChange}
        viewportMode={mode}
        onViewportModeChange={setMode}
        viewport={{ position: { x: 0, y: 0 }, zoom: 1 }}
        onViewportChange={handleViewportChange}
        editable={canEdit}
        onEditableChange={handleEditableChange}
        setFitViewOnResize={handleFitViewOnResizeChange}
        triggers={triggers}
        onDoubleClick={handleDoubleClick}
        onNodeClick={handleNodeClick}
        onNodeDoubleClick={handleNodeDoubleClick}
        fitViewOnResize={false}
        visible={visible}
      >
        <Diagram.Background />
        <Controls x>
          <Diagram.Controls.SelectViewportMode />
          <Diagram.Controls.FitView />
          <Flex.Box x pack>
            {hasUpdatePermission && (
              <Diagram.Controls.ToggleEdit disabled={control === "acquired"} />
            )}
            {!snapshot && <ControlToggleButton control={control} />}
          </Flex.Box>
        </Controls>
      </Base.Schematic>
      {legendVisible && (
        <Control.Legend
          position={legendPosition}
          onPositionChange={setLegendPosition}
          colors={doc?.legend?.colors ?? {}}
          allowVisibleChange={false}
        />
      )}
    </Controller>
  );
};

export const Schematic: Layout.Renderer = ({ layoutKey, ...rest }) => (
  <Loaded layoutKey={layoutKey} {...rest} />
);

export const LAYOUT_TYPE = "schematic";
export type LayoutType = typeof LAYOUT_TYPE;

export const HAUL_TYPE = Base.HAUL_TYPE;

export const Selectable: Selector.Selectable = ({ layoutKey, onPlace }) => {
  const hasCreatePermission = Access.useCreateGranted(schematic.TYPE_ONTOLOGY_ID);
  const handleClick = useCallback(() => {
    onPlace(create({ key: layoutKey }));
  }, [onPlace, layoutKey]);

  if (!hasCreatePermission) return null;

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
Selectable.useVisible = () => Access.useCreateGranted(schematic.TYPE_ONTOLOGY_ID);

export interface CreateArg extends Partial<Layout.BaseState> {
  key?: string;
}

export const create =
  (initial: CreateArg = {}): Layout.Creator =>
  ({ dispatch }) => {
    const { name = "Schematic", location = "mosaic", tab } = initial;
    const key = schematic.keyZ.safeParse(initial.key).data ?? uuid.create();
    dispatch(internalCreate({ key }));
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
