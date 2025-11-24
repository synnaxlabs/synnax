// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import {
  Button,
  Control,
  Diagram,
  Flex,
  Icon,
  type Legend,
  Menu as PMenu,
  Schematic as Core,
  Schematic,
  Text,
} from "@synnaxlabs/pluto";
import { location, uuid, xy } from "@synnaxlabs/x";
import { useCallback, useState } from "react";

import { Layout } from "@/layout";
import { type Selector } from "@/selector";
import { Workspace } from "@/workspace";

export const HAUL_TYPE = "schematic-element";

export const ContextMenu: Layout.ContextMenuRenderer = ({ layoutKey }) => (
  <PMenu.Menu level="small" gap="small">
    <Layout.MenuItems layoutKey={layoutKey} />
  </PMenu.Menu>
);

export const Loaded: Layout.Renderer = ({ layoutKey, visible }) => {
  const { name } = Layout.useSelectRequired(layoutKey);
  const [controlTrigger, setControlTrigger] = useState<number>(0);
  const acquireControl = useCallback(
    () => setControlTrigger(controlTrigger + 1),
    [controlTrigger],
  );
  const [controlStatus, setControlStatus] = useState<Control.Status>("released");
  const [editable, setEditable] = useState<boolean>(false);
  const handleEditableChange = useCallback((cbk: boolean) => setEditable(cbk), []);
  const [viewport, setViewport] = useState<Diagram.Viewport>({
    position: xy.ZERO,
    zoom: 1,
  });
  const [legendPosition, setLegendPosition] = useState<Legend.StickyXY>({
    x: 50,
    y: 50,
    units: { x: "px", y: "px" },
    root: { x: "left", y: "top" },
  });

  return (
    <div style={{ width: "inherit", height: "inherit", position: "relative" }}>
      <Control.Controller
        name={name}
        authority={255}
        acquireTrigger={controlTrigger}
        onStatusChange={setControlStatus}
      >
        <Core.Schematic
          schematicKey={layoutKey}
          onViewportChange={setViewport}
          viewportMode="pan"
          onViewportModeChange={() => {}}
          // Turns out that setting the zoom value to 1 here doesn't have any negative
          // effects on the schematic sizing and ensures that we position all the lines
          // in the correct place.
          viewport={viewport}
          onEditableChange={handleEditableChange}
          editable={editable}
          visible={visible}
          fitViewOnResize={false}
          setFitViewOnResize={() => {}}
        >
          <Diagram.Background />
          <Diagram.Controls>
            <Diagram.SelectViewportModeControl />
            <Diagram.FitViewControl />
            <Flex.Box x pack>
              <Diagram.ToggleEditControl disabled={controlStatus === "acquired"} />
              <Button.Toggle
                value={controlStatus === "acquired"}
                onChange={acquireControl}
                tooltipLocation={location.BOTTOM_LEFT}
                uncheckedVariant="outlined"
                checkedVariant="filled"
                size="small"
                tooltip={
                  <Text.Text level="small">
                    {controlStatus === "acquired"
                      ? "Release control"
                      : "Acquire control"}
                  </Text.Text>
                }
              >
                <Icon.Circle />
              </Button.Toggle>
            </Flex.Box>
          </Diagram.Controls>
        </Core.Schematic>
        <Control.Legend
          position={legendPosition}
          onPositionChange={handleLegendPositionChange}
          allowVisibleChange={false}
        />
      </Control.Controller>
    </div>
  );
};

export const LAYOUT_TYPE = "schematic";
export type LayoutType = typeof LAYOUT_TYPE;

export const SELECTABLE: Selector.Selectable = {
  key: LAYOUT_TYPE,
  title: "Schematic",
  icon: <Icon.Schematic />,
  create: async ({ layoutKey }) => open(layoutKey, "New Schematic"),
};

export const useCreateEmpty = (): ((
  params: Partial<Schematic.UseCreateArgs>,
) => void) => {
  const placeLayout = Layout.usePlacer();
  const workspace = Workspace.useSelectRequiredActiveKey();
  const { update } = Schematic.useCreate({
    afterSuccess: async ({ data: { key, name } }) => {
      placeLayout(open(key, name));
    },
  });
  return (params: Partial<Schematic.UseCreateArgs>) => {
    const key = schematic.keyZ.safeParse(params.key).data ?? uuid.create();
    return update({ workspace, name: "New Schematic", ...params, key });
  };
};

export const open = (key: string, name: string): Layout.PlacerArgs => ({
  key,
  name,
  type: LAYOUT_TYPE,
  location: "mosaic",
  icon: "Schematic",
});
