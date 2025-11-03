// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Flex, Icon, Text, Triggers, Viewport } from "@synnaxlabs/pluto";
import { type location } from "@synnaxlabs/x";
import { type ReactElement, useMemo } from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import {
  useSelectControlState,
  useSelectMeasureMode,
  useSelectViewportMode,
} from "@/lineplot/selectors";
import {
  type ClickMode,
  setControlState,
  setMeasureMode,
  setViewport,
  setViewportMode,
} from "@/lineplot/slice";

const TOOLTIP_LOCATION: location.XY = {
  x: "left",
  y: "bottom",
};

const style = { zIndex: 500 };

export interface NavControlsProps {
  layoutKey: string;
}

export const NavControls = ({ layoutKey }: NavControlsProps): ReactElement => {
  const control = useSelectControlState(layoutKey);
  const { layoutKey: vis } = Layout.useSelectActiveMosaicTabState();
  const mode = useSelectViewportMode(layoutKey);
  const measureMode = useSelectMeasureMode(layoutKey);
  const dispatch = useDispatch();

  const handleModeChange = (mode: Viewport.Mode): void => {
    dispatch(setViewportMode({ key: layoutKey, mode }));
  };

  const handleClickModeChange = (clickMode: ClickMode | null): void => {
    dispatch(setControlState({ key: layoutKey, state: { clickMode } }));
  };

  const handleTooltipChange = (tooltip: boolean): void => {
    dispatch(setControlState({ key: layoutKey, state: { enableTooltip: tooltip } }));
  };

  const handleZoomReset = (): void => {
    if (vis != null) dispatch(setViewport({ key: vis }));
  };

  const handleHoldChange = (hold: boolean): void => {
    dispatch(setControlState({ key: layoutKey, state: { hold } }));
  };

  const triggers = useMemo(() => Viewport.DEFAULT_TRIGGERS[mode], [mode]);

  return (
    <Flex.Box className={CSS.BE("line-plot", "nav-controls")} gap="small" style={style}>
      <Flex.Box x>
        <Viewport.SelectMode
          value={mode}
          onChange={handleModeChange}
          triggers={triggers}
          tooltipLocation={TOOLTIP_LOCATION}
        />
        <Button.Button
          onClick={handleZoomReset}
          variant="outlined"
          tooltipLocation={TOOLTIP_LOCATION}
          tooltip={
            <Text.Text level="small">
              Reset Zoom
              <Triggers.Text trigger={triggers.zoomReset[0]} el="span" />
            </Text.Text>
          }
          size="small"
        >
          <Icon.Expand />
        </Button.Button>
        <Button.Toggle
          value={control.enableTooltip}
          onChange={handleTooltipChange}
          checkedVariant="filled"
          size="small"
          uncheckedVariant="outlined"
          tooltip={<Text.Text level="small">Show Tooltip on Hover</Text.Text>}
          tooltipLocation={TOOLTIP_LOCATION}
        >
          <Icon.Tooltip />
        </Button.Toggle>
        <Button.Toggle
          value={control.clickMode != null}
          tooltip={<Text.Text level="small">Slope</Text.Text>}
          tooltipLocation={TOOLTIP_LOCATION}
          onChange={() =>
            handleClickModeChange(control.clickMode != null ? null : "measure")
          }
          size="small"
        >
          <Icon.Rule />
        </Button.Toggle>
        <Button.Toggle
          className={CSS.BE("control", "pause")}
          value={control.hold}
          onChange={handleHoldChange}
          uncheckedVariant="outlined"
          tooltipLocation={TOOLTIP_LOCATION}
          size="small"
          tooltip={
            <Flex.Box x align="center" gap="small">
              <Text.Text level="small">
                {control.hold ? "Resume live plotting" : "Pause live plotting"}
              </Text.Text>
              <Triggers.Text level="small" trigger={["H"]} />
            </Flex.Box>
          }
        >
          {control.hold ? <Icon.Play /> : <Icon.Pause />}
        </Button.Toggle>
      </Flex.Box>
      {control.clickMode === "measure" && (
        <Flex.Box x gap="small" className={CSS.BE("control", "measure")}>
          <Button.Toggle
            size="small"
            value={measureMode === "one"}
            onChange={() => dispatch(setMeasureMode({ key: layoutKey, mode: "one" }))}
          >
            1
          </Button.Toggle>
          <Button.Toggle
            size="small"
            value={measureMode === "two"}
            onChange={() => dispatch(setMeasureMode({ key: layoutKey, mode: "two" }))}
          >
            2
          </Button.Toggle>
        </Flex.Box>
      )}
    </Flex.Box>
  );
};
