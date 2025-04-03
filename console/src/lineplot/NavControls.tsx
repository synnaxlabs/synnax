// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  type Icon as PIcon,
  Select,
  Text,
  Triggers,
  Viewport,
} from "@synnaxlabs/pluto";
import { type location } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useMemo } from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { useSelectControlState, useSelectViewportMode } from "@/lineplot/selectors";
import {
  type ClickMode,
  setControlState,
  setViewport,
  setViewportMode,
} from "@/lineplot/slice";

const TOOLTIP_LOCATION: location.XY = {
  x: "left",
  y: "bottom",
};

export const NavControls = (): ReactElement => {
  const control = useSelectControlState();
  const vis = Layout.useSelectActiveMosaicTabKey();
  const mode = useSelectViewportMode();
  const dispatch = useDispatch();

  const handleModeChange = (mode: Viewport.Mode): void => {
    dispatch(setViewportMode({ mode }));
  };

  const handleClickModeChange = (clickMode: ClickMode): void => {
    dispatch(setControlState({ state: { clickMode } }));
  };

  const handleTooltipChange = (tooltip: boolean): void => {
    dispatch(setControlState({ state: { enableTooltip: tooltip } }));
  };

  const handleZoomReset = (): void => {
    if (vis != null) dispatch(setViewport({ key: vis }));
  };

  const handleHoldChange = (hold: boolean): void => {
    dispatch(setControlState({ state: { hold } }));
  };

  const triggers = useMemo(() => Viewport.DEFAULT_TRIGGERS[mode], [mode]);

  return (
    <Align.Space className={CSS.BE("line-plot", "nav-controls")} x size="small">
      <Viewport.SelectMode
        value={mode}
        onChange={handleModeChange}
        triggers={triggers}
        size="medium"
      />
      <Button.Icon
        onClick={handleZoomReset}
        variant="outlined"
        tooltipLocation={TOOLTIP_LOCATION}
        tooltip={
          <Align.Space x align="center">
            <Text.Text level="small">Reset Zoom</Text.Text>
            <Align.Space x empty>
              <Text.Keyboard level="small">
                <Text.Symbols.Meta />
              </Text.Keyboard>
              <Text.Keyboard level="small">Click</Text.Keyboard>
            </Align.Space>
          </Align.Space>
        }
        size="small"
      >
        <Icon.Expand />
      </Button.Icon>
      <Button.ToggleIcon
        value={control.enableTooltip}
        onChange={handleTooltipChange}
        checkedVariant="filled"
        size="small"
        uncheckedVariant="outlined"
        tooltip={
          <Align.Space x align="center">
            <Text.Text level="small">Show Tooltip on Hover</Text.Text>
          </Align.Space>
        }
        tooltipLocation={TOOLTIP_LOCATION}
      >
        <Icon.Tooltip />
      </Button.ToggleIcon>
      <Select.Button<
        ClickMode,
        { key: ClickMode; icon: PIcon.Element; tooltip: ReactNode }
      >
        value={control.clickMode}
        onChange={handleClickModeChange}
        size="small"
        bordered={false}
        entryRenderKey="icon"
        allowNone
        data={[
          {
            key: "measure",
            icon: <Icon.Rule />,
            tooltip: (
              <Align.Space x align="center">
                <Text.Text level="small">Slope</Text.Text>
              </Align.Space>
            ),
          },
        ]}
      >
        {({ title: _, entry, ...rest }) => (
          <Button.Icon
            {...rest}
            key={entry.key}
            variant={rest.selected ? "filled" : "outlined"}
            tooltip={entry.tooltip}
            tooltipLocation={TOOLTIP_LOCATION}
          >
            {entry.icon}
          </Button.Icon>
        )}
      </Select.Button>
      <Button.ToggleIcon
        className={CSS.BE("control", "pause")}
        value={control.hold}
        onChange={handleHoldChange}
        uncheckedVariant="outlined"
        tooltipLocation={TOOLTIP_LOCATION}
        size="small"
        tooltip={
          <Align.Space x align="center" size="small">
            <Text.Text level="small">
              {control.hold ? "Resume live plotting" : "Pause live plotting"}
            </Text.Text>
            <Triggers.Text level="small" trigger={["H"]}></Triggers.Text>
          </Align.Space>
        }
      >
        {control.hold ? <Icon.Play /> : <Icon.Pause />}
      </Button.ToggleIcon>
    </Align.Space>
  );
};
