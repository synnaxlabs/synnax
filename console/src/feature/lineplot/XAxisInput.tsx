// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { LinePlot } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { XAxisChannelSelect } from "@/feature/lineplot/SelectAxis";
import { CSS } from "@/platform/css";
import { Range } from "@/platform/range";

const SelectMode = (props: Select.ButtonsProps<lineplot.XAxisMode>): ReactElement => (
  <Select.Buttons<lineplot.XAxisMode> {...props}>
    <Select.Item itemKey="samples" size="small">
      Samples
    </Select.Item>
    <Select.Item itemKey="spectrum" size="small">
      Spectrum
    </Select.Item>
  </Select.Buttons>
);

const SelectWindow = (
  props: Select.ButtonsProps<lineplot.WindowFunction>,
): ReactElement => (
  <Select.Buttons<lineplot.WindowFunction> {...props}>
    <Select.Item itemKey="hann" size="small">
      Hann
    </Select.Item>
    <Select.Item itemKey="rectangular" size="small">
      Rectangular
    </Select.Item>
    <Select.Item itemKey="flat_top" size="small">
      Flat top
    </Select.Item>
  </Select.Buttons>
);

const SelectScale = (
  props: Select.ButtonsProps<lineplot.MagnitudeScale>,
): ReactElement => (
  <Select.Buttons<lineplot.MagnitudeScale> {...props}>
    <Select.Item itemKey="linear" size="small">
      Linear
    </Select.Item>
    <Select.Item itemKey="decibel" size="small">
      dB
    </Select.Item>
  </Select.Buttons>
);

const POINT_LIMIT_BOUNDS = { lower: 2, upper: 1048576 };
const POINT_LIMIT_DRAG_SCALE = { x: 10, y: 10 };

interface SpectrumControlsProps {
  axisKey: lineplot.XAxisKey;
  axis: lineplot.Axis;
}

const SpectrumControls = ({ axisKey, axis }: SpectrumControlsProps): ReactElement => {
  const dispatch = LinePlot.useSingleDispatch();
  const setSpectrum = (spectrum: Partial<lineplot.Spectrum>) =>
    dispatch(
      lineplot.setAxisSpectrum({
        key: axisKey,
        spectrum: { ...axis.spectrum, ...spectrum },
      }),
    );
  return (
    <>
      <Input.Item x label="Window">
        <SelectWindow
          value={axis.spectrum.window}
          onChange={(window: lineplot.WindowFunction) => setSpectrum({ window })}
        />
      </Input.Item>
      <Input.Item x label="Scale">
        <SelectScale
          value={axis.spectrum.scale}
          onChange={(scale: lineplot.MagnitudeScale) => setSpectrum({ scale })}
        />
      </Input.Item>
      <Input.Item x label="Point limit" className={CSS.BE("line-plot", "window-field")}>
        <Input.Numeric
          value={axis.spectrum.pointLimit}
          onChange={(pointLimit: number) => setSpectrum({ pointLimit })}
          bounds={POINT_LIMIT_BOUNDS}
          dragScale={POINT_LIMIT_DRAG_SCALE}
        />
      </Input.Item>
    </>
  );
};

export interface XAxisInputProps {
  axisKey: lineplot.XAxisKey;
}

/**
 * Edits an x axis: whether it plots samples or their spectrum, then the channel the
 * samples plot against or the spectrum's settings. A triggered window always plots
 * samples, so it hides the mode.
 */
export const XAxisInput = ({ axisKey }: XAxisInputProps): ReactElement => {
  const axis = LinePlot.useAxis({ axisKey });
  const ranges = LinePlot.useRanges();
  const dispatch = LinePlot.useSingleDispatch();
  const triggered =
    ranges.custom?.variant === "triggered" &&
    ranges[axisKey].includes(Range.CUSTOM_KEY);
  const showMode = !triggered;
  const spectral = showMode && axis.mode === "spectrum";
  const handleModeChange = (mode: lineplot.XAxisMode) =>
    dispatch(lineplot.setAxisMode({ key: axisKey, mode }));
  return (
    <Flex.Box x align="center" className={CSS.BE("line-plot", "toolbar", "data-x-row")}>
      {showMode && (
        <Input.Item x label="Mode">
          <SelectMode value={axis.mode} onChange={handleModeChange} />
        </Input.Item>
      )}
      {spectral ? (
        <SpectrumControls axisKey={axisKey} axis={axis} />
      ) : (
        <XAxisChannelSelect
          axisKey={axisKey}
          className={CSS.BE("line-plot", "toolbar", "data-x")}
        />
      )}
    </Flex.Box>
  );
};
