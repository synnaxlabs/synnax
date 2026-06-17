// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/lineplot/Controls.css";

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
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { Controls as Base } from "@/components";
import { CSS } from "@/css";

import { Session } from "./session";

export interface ControlsProps {
  layoutKey: string;
  hasAnnotations: boolean;
}

const SelectViewportMode = (): ReactElement => {
  const viewportMode = Session.useSelectViewportMode();
  const handleViewportModeChange = Session.useSetViewportMode();
  const triggers = useMemo(
    () => Viewport.DEFAULT_TRIGGERS[viewportMode],
    [viewportMode],
  );
  return (
    <Viewport.SelectMode
      value={viewportMode}
      onChange={handleViewportModeChange}
      triggers={triggers}
      tooltipLocation={location.BOTTOM_LEFT}
    />
  );
};

const ZoomReset = (): ReactElement => {
  const key = LinePlot.useKey();
  const viewportMode = Session.useSelectViewportMode();
  const triggers = useMemo(
    () => Viewport.DEFAULT_TRIGGERS[viewportMode],
    [viewportMode],
  );
  const dispatch = useDispatch();
  const handleZoomReset = useCallback((): void => {
    dispatch(Session.resetViewport({ key }));
  }, [key]);
  return (
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
  );
};

const ToggleTooltips = (): ReactElement => {
  const enableTooltips = Session.useSelectEnableTooltip();
  const handleEnableTooltipChange = Session.useSetTooltipsVisible();
  return (
    <Button.Toggle
      value={enableTooltips}
      onChange={handleEnableTooltipChange}
      size="small"
      tooltip="Show tooltip on hover"
      tooltipLocation={location.BOTTOM_LEFT}
    >
      <Icon.Tooltip />
    </Button.Toggle>
  );
};

const ToggleRangeAnnotations = (): ReactElement => {
  const showRangeAnnotations = Session.useSelectShowRangeAnnotations();
  const handleRangeAnnotationsVisibleChange = Session.useSetShowRangeAnnotations();
  return (
    <Button.Toggle
      value={showRangeAnnotations}
      onChange={handleRangeAnnotationsVisibleChange}
      size="small"
      tooltip={`${showRangeAnnotations ? "Hide" : "Show"} range annotations`}
      tooltipLocation={location.BOTTOM_LEFT}
    >
      <Icon.Range />
    </Button.Toggle>
  );
};

const ToggleMeasureTool = (): ReactElement => {
  const clickMode = Session.useSelectClickMode();
  const handleClickModeChange = Session.useSetClickMode();
  return (
    <Button.Toggle
      value={clickMode != null}
      tooltip={`${clickMode != null ? "Close" : "Open"} measure tool`}
      tooltipLocation={location.BOTTOM_LEFT}
      onChange={() =>
        handleClickModeChange(clickMode != null ? undefined : "measure")
      }
      size="small"
    >
      <Icon.Rule />
    </Button.Toggle>
  );
};

const HOLD_TRIGGER: Triggers.Trigger = ["H"]

const ToggleHold = (): ReactElement => {
  const hold = Session.useSelectHold();
  const handleHoldChange = Session.useSetHold();
  return (
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
  );
};

const MeasureModeFirst = (): ReactElement => {
  const measureMode = Session.useSelectMeasureMode();
  const handleMeasureModeChange = Session.useSetMeasureMode();
  return (
    <Button.Toggle
      size="small"
      value={measureMode === "one"}
      tooltip="Select first point"
      tooltipLocation={location.BOTTOM_LEFT}
      onChange={() => handleMeasureModeChange("one")}
    >
      1
    </Button.Toggle>
  );
};

const MeasureModeSecond = (): ReactElement => {
  const measureMode = Session.useSelectMeasureMode();
  const handleMeasureModeChange = Session.useSetMeasureMode();
  return (
    <Button.Toggle
      size="small"
      tooltipLocation={location.BOTTOM_LEFT}
      value={measureMode === "two"}
      tooltip="Select second point"
      onChange={() => handleMeasureModeChange("two")}
    >
      2
    </Button.Toggle>
  );
};

const MeasureControls = (): ReactElement | null => {
  const clickMode = Session.useSelectClickMode();
  if (clickMode !== "measure") return null;
  return (
    <Flex.Box x pack className={CSS.BE("control", "measure")}>
      <MeasureModeFirst />
      <MeasureModeSecond />
    </Flex.Box>
  );
};

export const Controls = ({ hasAnnotations }: ControlsProps): ReactElement => {
  const showRangeAnnotations = Session.useSelectShowRangeAnnotations();
  return (
    <Base
      className={CSS(
        showRangeAnnotations &&
          hasAnnotations &&
          CSS.BM("controls", "annotations-visible"),
      )}
    >
      <Flex.Box x gap="small">
        <SelectViewportMode />
        <ZoomReset />
        <ToggleTooltips />
        {hasAnnotations && <ToggleRangeAnnotations />}
        <ToggleMeasureTool />
        <ToggleHold />
      </Flex.Box>
      <MeasureControls />
    </Base>
  );
};
