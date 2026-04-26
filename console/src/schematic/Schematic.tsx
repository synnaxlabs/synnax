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
import { location, uuid } from "@synnaxlabs/x";
import { memo, type ReactElement, useCallback, useMemo, useState } from "react";
import { useDispatch } from "react-redux";

import { ContextMenu as CMenu, Controls as BaseControls } from "@/components";
import { Layout } from "@/layout";
import { Controller } from "@/schematic/Controller";
import {
  useSelectControlStatus,
  useSelectEditable,
  useSelectFitViewOnResize,
  useSelectLegend,
  useSelectSelected,
  useSelectViewport,
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

interface ControlsProps {
  hasUpdatePermission: boolean;
  control: Control.Status;
  snapshot: boolean;
}

const Controls = memo(
  ({ hasUpdatePermission, control, snapshot }: ControlsProps): ReactElement => (
    <BaseControls x>
      <Diagram.Controls.SelectViewportMode />
      <Diagram.Controls.FitView />
      <Flex.Box x pack>
        {hasUpdatePermission && (
          <Diagram.Controls.ToggleEdit disabled={control === "acquired"} />
        )}
        {!snapshot && <ControlToggleButton control={control} />}
      </Flex.Box>
    </BaseControls>
  ),
);
Controls.displayName = "Controls";

export const ContextMenu: Layout.ContextMenuRenderer = ({ layoutKey }) => (
  <CMenu.Menu>
    <Layout.MenuItems layoutKey={layoutKey} />
  </CMenu.Menu>
);

export const Loaded: Layout.Renderer = ({ layoutKey, visible }) => {
  const windowKey = useSelectWindowKey() as string;
  const { name } = Layout.useSelectRequired(layoutKey);

  const { data: doc } = Base.useRetrieve({ key: layoutKey });
  const dispatch = useDispatch();
  useAutoUpload(layoutKey, name);

  const hasUpdatePermission =
    Access.useUpdateGranted(schematic.ontologyID(layoutKey)) &&
    !(doc?.snapshot ?? false);
  const editableState = useSelectEditable(layoutKey);
  const editable = hasUpdatePermission && editableState;
  const fitViewOnResize = useSelectFitViewOnResize(layoutKey);
  const selected = useSelectSelected(layoutKey);
  const control = useSelectControlStatus(layoutKey);
  const legend = useSelectLegend(layoutKey);
  const viewport = useSelectViewport(layoutKey);

  const handleSelectionChange = useCallback(
    (next: string[]) => dispatch(setSelected({ key: layoutKey, selected: next })),
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

  const handleViewportChange = useCallback(
    (vp: Diagram.Viewport) => dispatch(setViewport({ key: layoutKey, viewport: vp })),
    [dispatch, layoutKey],
  );

  const [mode, setMode] = useState<Viewport.Mode>("select");
  const triggers = useMemo(() => Viewport.DEFAULT_TRIGGERS[mode], [mode]);
  const handleDoubleClick = useCallback(() => {
    if (!editable) return;
    dispatch(
      Layout.setNavDrawerVisible({ windowKey, key: "visualization", value: true }),
    );
  }, [windowKey, editable, dispatch]);

  const snapshot = doc?.snapshot ?? false;

  return (
    <Controller resourceKey={layoutKey} authority={doc?.authority ?? 1}>
      <Base.Schematic
        resourceKey={layoutKey}
        selected={selected}
        onSelectionChange={handleSelectionChange}
        viewportMode={mode}
        onViewportModeChange={setMode}
        viewport={viewport}
        onViewportChange={handleViewportChange}
        editable={editable}
        onEditableChange={handleEditableChange}
        setFitViewOnResize={handleFitViewOnResizeChange}
        triggers={triggers}
        onDoubleClick={handleDoubleClick}
        fitViewOnResize={fitViewOnResize}
        visible={visible}
      >
        <Diagram.Background />
        <Controls
          hasUpdatePermission={hasUpdatePermission}
          control={control}
          snapshot={snapshot}
        />
      </Base.Schematic>
      {legend.visible && (
        <Control.Legend
          position={legend.position}
          colors={doc?.legend?.colors ?? {}}
          allowVisibleChange={false}
        />
      )}
    </Controller>
  );
};

export const SchematicComponent: Layout.Renderer = (props) => <Loaded {...props} />;

export const LAYOUT_TYPE = "schematic";
export type LayoutType = typeof LAYOUT_TYPE;

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
    const key = initial.key ?? uuid.create();
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
