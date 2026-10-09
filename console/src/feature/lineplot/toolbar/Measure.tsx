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
import { math, TimeSpan } from "@synnaxlabs/x";
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

const value = (v: number | null, units?: string): string => {
  if (v == null) return EMPTY;
  const rounded = math.smartRound(v);
  return units == null || units.length === 0 ? `${rounded}` : `${rounded} ${units}`;
};

const seconds = (v: number | null): string =>
  v == null ? EMPTY : new TimeSpan(v * 1e9).toString();

const SAMPLE_COLUMNS: Column[] = [
  { key: "min", label: "Min", format: (m) => value(m.min, m.units) },
  { key: "max", label: "Max", format: (m) => value(m.max, m.units) },
  {
    key: "peakToPeak",
    label: "Peak to peak",
    format: (m) => value(m.peakToPeak, m.units),
  },
  { key: "mean", label: "Mean", format: (m) => value(m.mean, m.units) },
  { key: "rms", label: "RMS", format: (m) => value(m.rms, m.units) },
];

const TIME_COLUMNS: Column[] = [
  { key: "frequency", label: "Frequency", format: (m) => value(m.frequency, "Hz") },
  { key: "riseTime", label: "Rise time", format: (m) => seconds(m.riseTime) },
];

const SPECTRUM_COLUMNS: Column[] = [
  { key: "max", label: "Peak", format: (m) => value(m.max, m.units) },
  { key: "peakX", label: "Peak frequency", format: (m) => value(m.peakX, "Hz") },
  { key: "mean", label: "Mean", format: (m) => value(m.mean, m.units) },
];

/** Reads the x1 axis to pick the statistics that make sense for what it plots. */
const useColumns = (): Column[] => {
  const x1 = LinePlot.useXAxis({ axisKey: "x1" });
  const ranges = LinePlot.useRanges();
  if (x1.mode === "spectrum") return SPECTRUM_COLUMNS;
  const triggered =
    ranges.x1.includes(Range.CUSTOM_KEY) && ranges.custom?.variant === "triggered";
  if (x1.type === "time" || triggered) return [...SAMPLE_COLUMNS, ...TIME_COLUMNS];
  return SAMPLE_COLUMNS;
};

export const Measure = (): ReactElement => {
  const measurements = Session.LinePlot.useSelectMeasurements();
  const columns = useColumns();
  if (measurements.length === 0)
    return <Empty.Action message="No measurements yet" action="Plot a channel" />;
  return (
    <Flex.Box y className={CSS.BE("line-plot", "toolbar", "measure")} full="x">
      <table className={CSS.BE("line-plot", "measure-table")}>
        <thead>
          <tr>
            <th>
              <Text.Text level="small" weight={500}>
                Line
              </Text.Text>
            </th>
            {columns.map((c) => (
              <th key={c.key}>
                <Text.Text level="small" weight={500}>
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
                <Text.Text level="small">{m.label ?? m.line}</Text.Text>
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
