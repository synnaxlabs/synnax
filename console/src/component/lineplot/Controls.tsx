// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Button,
  Flex,
  Icon,
  LinePlot,
  Text,
  Triggers,
  Viewport,
} from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { memo, type ReactElement, useMemo } from "react";

import { CSS } from "@/component/css";
import { Vis } from "@/component/vis";
import { HOLD_TRIGGER } from "@/service/lineplot/useTriggerHold";
import { Session } from "@/session";

export interface ControlsProps {
  hasAnnotations: boolean;
}

export const Controls = memo(({ hasAnnotations }: ControlsProps): ReactElement => {
  const key = LinePlot.useKey();
  const { enableTooltip, clickMode, hold } = Session.LinePlot.useSelectControlState();
  const annotationsVisible = Session.LinePlot.useSelectAnnotationsVisible();
  const mode = Session.LinePlot.useSelectViewportMode();
  const measureMode = Session.LinePlot.useSelectMeasureMode();
  const dispatch = Session.useDispatch();

  const handleModeChange = (mode: Viewport.Mode): void => {
    dispatch(Session.LinePlot.setViewportMode({ key, mode }));
  };

  const handleTooltipChange = (enabled: boolean): void => {
    dispatch(Session.LinePlot.setControlEnableTooltip({ key, enabled }));
  };

  const handleZoomReset = (): void => {
    dispatch(Session.LinePlot.setViewport({ key }));
  };

  const handleHoldChange = (hold: boolean): void => {
    dispatch(Session.LinePlot.setControlHold({ key, hold }));
  };

  const handleAnnotationsVisibilityChange = (visible: boolean): void => {
    dispatch(Session.LinePlot.setRangeAnnotationsVisible({ key, visible }));
  };

  const handleToggleMeasure = (): void => {
    dispatch(Session.LinePlot.toggleControlClickMode({ key, mode: "measure" }));
  };

  const handleSelectFirstPoint = (): void => {
    dispatch(Session.LinePlot.setMeasureMode({ key, mode: "one" }));
  };

  const handleSelectSecondPoint = (): void => {
    dispatch(Session.LinePlot.setMeasureMode({ key, mode: "two" }));
  };

  const triggers = useMemo(() => Viewport.DEFAULT_TRIGGERS[mode], [mode]);

  return (
    <Vis.Controls
      className={CSS(
        annotationsVisible &&
          hasAnnotations &&
          CSS.BM("controls", "annotations-visible"),
      )}
    >
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
            <Text.Text level="small" color={11}>
              Reset zoom
              <Triggers.Text trigger={triggers.zoomReset[0]} el="span" />
            </Text.Text>
          }
          size="small"
        >
          <Icon.Expand />
        </Button.Button>
        <Button.Toggle
          value={enableTooltip}
          onChange={handleTooltipChange}
          size="small"
          tooltip="Show tooltip on hover"
          tooltipLocation={location.BOTTOM_LEFT}
        >
          <Icon.Tooltip />
        </Button.Toggle>
        {hasAnnotations && (
          <Button.Toggle
            value={annotationsVisible}
            onChange={handleAnnotationsVisibilityChange}
            size="small"
            tooltip={`${annotationsVisible ? "Hide" : "Show"} range annotations`}
            tooltipLocation={location.BOTTOM_LEFT}
          >
            <Icon.Range />
          </Button.Toggle>
        )}
        <Button.Toggle
          value={clickMode != null}
          tooltip={`${clickMode != null ? "Close" : "Open"} measure tool`}
          tooltipLocation={location.BOTTOM_LEFT}
          onChange={handleToggleMeasure}
          size="small"
        >
          <Icon.Rule />
        </Button.Toggle>
        <Button.Toggle
          value={hold}
          onChange={handleHoldChange}
          tooltipLocation={location.BOTTOM_LEFT}
          size="small"
          tooltip={
            <Text.Text level="small" color={11}>
              {`${hold ? "Resume" : "Pause"} live plotting`}
              <Triggers.Text trigger={HOLD_TRIGGER} level="small"></Triggers.Text>
            </Text.Text>
          }
        >
          {hold ? <Icon.Play /> : <Icon.Pause />}
        </Button.Toggle>
      </Flex.Box>
      {clickMode === "measure" && (
        <Flex.Box x pack className={CSS.BE("control", "measure")}>
          <Button.Toggle
            size="small"
            value={measureMode === "one"}
            tooltip="Select first point"
            tooltipLocation={location.BOTTOM_LEFT}
            onChange={handleSelectFirstPoint}
          >
            1
          </Button.Toggle>
          <Button.Toggle
            size="small"
            tooltipLocation={location.BOTTOM_LEFT}
            value={measureMode === "two"}
            tooltip="Select second point"
            onChange={handleSelectSecondPoint}
          >
            2
          </Button.Toggle>
        </Flex.Box>
      )}
    </Vis.Controls>
  );
});
Controls.displayName = "Controls";
