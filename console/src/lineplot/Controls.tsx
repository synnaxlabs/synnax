// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/lineplot/Controls.css";

import { Button, Flex, Icon, Text, Triggers, Viewport } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { type ReactElement, useMemo } from "react";
import { useDispatch } from "react-redux";

import { Controls as Core } from "@/components";
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

export interface ControlsProps {
  layoutKey: string;
}

export const Controls = ({ layoutKey }: ControlsProps): ReactElement => {
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
    <Core>
      <Flex.Box x gap="small">
        <Viewport.SelectMode
          value={mode}
          onChange={handleModeChange}
          triggers={triggers}
          tooltipLocation={location.BOTTOM_LEFT}
        />
        <Button.Button
          onClick={handleZoomReset}
          tooltipLocation={location.BOTTOM_LEFT}
          tooltip={
            <Text.Text level="small">
              Reset zoom
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
          size="small"
          tooltip={<Text.Text level="small">Show tooltip on hover</Text.Text>}
          tooltipLocation={location.BOTTOM_LEFT}
        >
          <Icon.Tooltip />
        </Button.Toggle>
        <Button.Toggle
          value={control.clickMode != null}
          tooltip={
            <Text.Text level="small">{`${control.clickMode != null ? "Close" : "Open"} measure tool`}</Text.Text>
          }
          tooltipLocation={location.BOTTOM_LEFT}
          onChange={() =>
            handleClickModeChange(control.clickMode != null ? null : "measure")
          }
          size="small"
        >
          <Icon.Rule />
        </Button.Toggle>
        <Button.Toggle
          value={control.hold}
          onChange={handleHoldChange}
          tooltipLocation={location.BOTTOM_LEFT}
          size="small"
          tooltip={
            <Text.Text level="small">
              {`${control.hold ? "Resume" : "Pause"} live plotting`}
              <Triggers.Text trigger={["H"]} level="small"></Triggers.Text>
            </Text.Text>
          }
        >
          {control.hold ? <Icon.Play /> : <Icon.Pause />}
        </Button.Toggle>
      </Flex.Box>
      {control.clickMode === "measure" && (
        <Flex.Box x pack className={CSS.BE("control", "measure")}>
          <Button.Toggle
            size="small"
            value={measureMode === "one"}
            tooltip={<Text.Text level="small">Select first point</Text.Text>}
            tooltipLocation={location.BOTTOM_LEFT}
            onChange={() => dispatch(setMeasureMode({ key: layoutKey, mode: "one" }))}
          >
            1
          </Button.Toggle>
          <Button.Toggle
            size="small"
            tooltipLocation={location.BOTTOM_LEFT}
            value={measureMode === "two"}
            tooltip={<Text.Text level="small">Select second point</Text.Text>}
            onChange={() => dispatch(setMeasureMode({ key: layoutKey, mode: "two" }))}
          >
            2
          </Button.Toggle>
        </Flex.Box>
      )}
    </Core>
  );
};
