// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { Text } from "@synnaxlabs/lyra/text";
import { LinePlot } from "@synnaxlabs/pluto";
import { type lineplot } from "@synnaxlabs/pluto/ether";
import { color, math } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { Range } from "@/platform/range";
import { Session } from "@/session";

interface Column {
  key: string;
  label: string;
  format: (m: lineplot.Measurement) => string;
}

const EMPTY = "—";

const SIGNIFICANT = new Intl.NumberFormat("en-US", {
  maximumSignificantDigits: 4,
  useGrouping: false,
});

const value = (v: number | null, units?: string): string => {
  if (v == null) return EMPTY;
  const n = SIGNIFICANT.format(v);
  return units == null ? n : `${n} ${units}`;
};

/**
 * Formats a sample statistic to four significant digits of the line's peak to peak, so
 * every value in a row shares one resolution and float noise around zero reads as 0.
 */
const sample = (v: number | null, peakToPeak: number | null): string => {
  if (v == null) return EMPTY;
  if (peakToPeak == null || peakToPeak === 0) return value(v);
  const decimals = Math.min(Math.max(3 - Math.floor(Math.log10(peakToPeak)), 0), 12);
  const fixed = v.toFixed(decimals);
  return Number(fixed) === 0 ? (0).toFixed(decimals) : fixed;
};

const SECOND_UNITS: Array<[number, string]> = [
  [1, "s"],
  [1e-3, "ms"],
  [1e-6, "µs"],
  [1e-9, "ns"],
];

/** Formats seconds in the largest unit that keeps the value at or above one. */
const seconds = (v: number | null): string => {
  if (v == null) return EMPTY;
  const [scale, unit] =
    SECOND_UNITS.find(([scale]) => Math.abs(v) >= scale) ?? SECOND_UNITS.at(-1)!;
  return `${(v / scale).toPrecision(3)} ${unit}`;
};

const SAMPLE_COLUMNS: Column[] = [
  { key: "last", label: "Last", format: (m) => sample(m.last, m.peakToPeak) },
  { key: "min", label: "Min", format: (m) => sample(m.min, m.peakToPeak) },
  { key: "max", label: "Max", format: (m) => sample(m.max, m.peakToPeak) },
  { key: "mean", label: "Mean", format: (m) => sample(m.mean, m.peakToPeak) },
  { key: "stdDev", label: "Std. dev.", format: (m) => sample(m.stdDev, m.peakToPeak) },
];

const FREQUENCY_COLUMN: Column = {
  key: "frequency",
  label: "Frequency",
  format: (m) => value(m.frequency, "Hz"),
};

const WAVEFORM_COLUMNS: Column[] = [
  {
    key: "peakToPeak",
    label: "Peak to peak",
    format: (m) => sample(m.peakToPeak, m.peakToPeak),
  },
  { key: "rms", label: "RMS", format: (m) => sample(m.rms, m.peakToPeak) },
  FREQUENCY_COLUMN,
  { key: "riseTime", label: "Rise time", format: (m) => seconds(m.riseTime) },
];

const SPECTRUM_COLUMNS: Column[] = [
  { key: "peakX", label: "Peak frequency", format: (m) => value(m.peakX, "Hz") },
  { key: "max", label: "Peak magnitude", format: (m) => value(m.max) },
];

const COUNT = new Intl.NumberFormat("en-US");

const COUNT_COLUMN: Column = {
  key: "count",
  label: "Samples",
  format: (m) => COUNT.format(m.count),
};

/** Reads the x1 axis to pick the statistics that make sense for what it plots. */
const useColumns = (): Column[] => {
  const x1 = LinePlot.useXAxis({ axisKey: "x1" });
  const ranges = LinePlot.useRanges();
  const triggered =
    ranges.x1.includes(Range.CUSTOM_KEY) && ranges.custom?.variant === "triggered";
  if (triggered) return [...SAMPLE_COLUMNS, ...WAVEFORM_COLUMNS];
  if (x1.mode === "spectrum") return SPECTRUM_COLUMNS;
  if (x1.type === "time") return [...SAMPLE_COLUMNS, FREQUENCY_COLUMN];
  return SAMPLE_COLUMNS;
};

export const Measure = (): ReactElement => {
  const measurements = Session.LinePlot.useSelectMeasurements();
  const columns = [...useColumns(), COUNT_COLUMN];
  const lines = LinePlot.useLines();
  const colors = new Map(lines.map((l) => [l.key, color.cssString(l.color)]));
  if (measurements.length === 0)
    return <Empty.Action message="No measurements yet" action="Plot a channel" />;
  return (
    <Flex.Box y className={CSS.BE("line-plot", "toolbar", "measure")} full="x">
      <table className={CSS.BE("line-plot", "measure-table")}>
        <thead>
          <tr>
            <th>
              <Text.Text level="small" weight={500} color={9}>
                Line
              </Text.Text>
            </th>
            {columns.map((c) => (
              <th key={c.key}>
                <Text.Text level="small" weight={500} color={9}>
                  {c.label}
                </Text.Text>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {measurements.map((m) => (
            <tr key={m.line}>
              <td>
                <Flex.Box x align="center" gap="small">
                  <span
                    className={CSS.BE("line-plot", "measure-swatch")}
                    style={{ backgroundColor: colors.get(m.line) }}
                  />
                  <Text.Text level="small" overflow="ellipsis">
                    {m.label ?? m.line}
                  </Text.Text>
                </Flex.Box>
              </td>
              {columns.map((c) => (
                <td key={c.key}>
                  <Text.Text level="small">{c.format(m)}</Text.Text>
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </Flex.Box>
  );
};
