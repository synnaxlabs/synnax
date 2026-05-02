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
  Icon,
  Schematic as Base,
  Viewport,
} from "@synnaxlabs/pluto";
import { deep, location, type sticky, uuid } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo, useState } from "react";
import { useDispatch } from "react-redux";

import { ContextMenu as CContextMenu, Controls } from "@/components";
import { Layout } from "@/layout";
import { Controller } from "@/schematic/Controller";
import {
  useSelectLegendVisible,
  useSelectRequired,
  useSelectRequiredViewportMode,
  useSelectSelected,
} from "@/schematic/selectors";
import {
  internalCreate,
  setEditable,
  setFitViewOnResize,
  setSelected,
  setViewport,
  setViewportMode,
  type State,
  ZERO_STATE,
} from "@/schematic/slice";
import { Selector } from "@/selector";

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

export const ContextMenu: Layout.ContextMenuRenderer = ({ layoutKey }) => (
  <CContextMenu.Menu>
    <Layout.MenuItems layoutKey={layoutKey} />
  </CContextMenu.Menu>
);

export const Loaded: Layout.Renderer = ({ layoutKey, visible }) => {
  const windowKey = useSelectWindowKey() as string;
  const { data: doc } = Base.useRetrieve({ key: layoutKey });
  const dispatch = useDispatch();
  const state = useSelectRequired(layoutKey);
  const legendVisible = useSelectLegendVisible(layoutKey);
  const selected = useSelectSelected(layoutKey);

  const hasUpdatePermission =
    Access.useUpdateGranted(schematic.ontologyID(layoutKey)) &&
    !(doc?.snapshot ?? false);
  const canEdit = hasUpdatePermission && state.editable;
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

  const handleViewportModeChange = useCallback(
    (mode: Viewport.Mode) => dispatch(setViewportMode({ key: layoutKey, mode })),
    [dispatch, layoutKey],
  );

  const mode = useSelectRequiredViewportMode(layoutKey);
  const triggers = useMemo(() => Viewport.DEFAULT_TRIGGERS[mode], [mode]);

  const handleDoubleClick = useCallback(() => {
    if (!state.editable) return;
    dispatch(
      Layout.setNavDrawerVisible({
        windowKey,
        key: "visualization",
        value: true,
      }),
    );
  }, [windowKey, state.editable, dispatch]);

  const [legendPosition, setLegendPosition] = useState<sticky.XY>(
    state.legend.position,
  );

  return (
    <Controller resourceKey={layoutKey} authority={state.authority}>
      <Base.Schematic
        resourceKey={layoutKey}
        selected={selected}
        onSelectionChange={handleSelectionChange}
        viewportMode={mode}
        onViewportModeChange={handleViewportModeChange}
        viewport={{ ...state.viewport, zoom: 1 }}
        onViewportChange={handleViewportChange}
        editable={canEdit}
        onEditableChange={handleEditableChange}
        setFitViewOnResize={handleFitViewOnResizeChange}
        triggers={triggers}
        onDoubleClick={handleDoubleClick}
        fitViewOnResize={state.fitViewOnResize}
        visible={visible}
      >
        <Diagram.Background />
        <Controls x>
          <Diagram.Controls.SelectViewportMode />
          <Diagram.Controls.FitView />
          <Flex.Box x pack>
            {hasUpdatePermission && (
              <Diagram.Controls.ToggleEdit disabled={state.control === "acquired"} />
            )}
            {!snapshot && <ControlToggleButton control={state.control} />}
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

export type CreateArg = Partial<State> & Partial<Layout.BaseState>;

export const create =
  (initial: CreateArg = {}): Layout.Creator =>
  ({ dispatch }) => {
    const { name = "Schematic", location = "mosaic", tab, ...rest } = initial;
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
