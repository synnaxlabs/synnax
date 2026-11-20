// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Text } from "@synnaxlabs/lyra/text";
import { type color, compare, TimeSpan, type TimeStamp } from "@synnaxlabs/x";
import type { ReactElement, ReactNode } from "react";

import { type LineSpec, useContext } from "@/lineplot/Frame";
import { formatRelativeTime } from "@/vis/axis/relative";
import { Legend as Base } from "@/vis/legend";
import type { EntriesProps } from "@/vis/legend/Entries";
import type { GroupData } from "@/vis/legend/Grouped";

type LineHandlers = Pick<
  EntriesProps,
  "onColorChange" | "onLabelChange" | "onVisibleChange" | "onHoverChange"
>;

/** A range that the legend lists its lines under, on a relative x axis. */
export interface LegendRange {
  key: string;
  name: string;
  color?: color.Crude;
  /** The time that the range draws at zero. A range with none does not draw. */
  zero?: TimeStamp;
  /**
   * The start of a range whose zero is a custom T0. Undefined when zero is the start.
   */
  start?: TimeStamp;
}

export interface LegendProps extends Omit<
  Base.SimpleProps,
  "data" | "onColorChange" | "onLabelChange" | "onVisibleChange"
> {
  onLineColorChange?: EntriesProps["onColorChange"];
  onLineLabelChange?: EntriesProps["onLabelChange"];
  onLineVisibleChange?: EntriesProps["onVisibleChange"];
  /** Lists lines under these ranges, in order, instead of under their axes. */
  ranges?: LegendRange[];
  hiddenRanges?: Set<string>;
  onRangeVisibleChange?: EntriesProps["onVisibleChange"];
  onRangeColorChange?: (key: string, color: color.Color) => void;
  /** Called with the key of the line or range the pointer is on, or null. */
  onHoverChange?: EntriesProps["onHoverChange"];
  footer?: ReactNode;
}

const groupLines = (lines: LineSpec[]): Base.GroupData[] => {
  const groupInfo: Record<string, LineSpec[]> = {};
  for (const line of lines) {
    const group = groupInfo[line.legendGroup];
    if (group != null) group.push(line);
    else groupInfo[line.legendGroup] = [line];
  }
  const groups = Object.entries(groupInfo).map(([key, data]) => ({
    key,
    name: key,
    data,
  }));
  groups.sort(compare.by((g) => g.name));
  return groups;
};

const ABSOLUTE_ONLY = "Shows in absolute time only";

const zeroBadge = (zero: TimeStamp, start: TimeStamp): GroupData["badge"] => {
  const offset = formatRelativeTime(
    Number(start.valueOf() - zero.valueOf()),
    Number(TimeSpan.MILLISECOND.valueOf()),
  );
  return {
    label: "T0",
    tooltip: `T0 at ${zero.toString("preciseDate", "local")}. The range starts at ${offset}.`,
  };
};

// groupRanges lists each range in order with the lines registered under its key.
const groupRanges = (
  ranges: LegendRange[],
  lines: LineSpec[],
  hidden: Set<string> | undefined,
): GroupData[] =>
  ranges.map(({ key, name, color, zero, start }) => ({
    key,
    name,
    color,
    visible: hidden?.has(key) !== true,
    badge: zero != null && start != null ? zeroBadge(zero, start) : undefined,
    note: zero == null ? ABSOLUTE_ONLY : undefined,
    data: lines
      .filter((l) => l.legendGroup === key)
      .map(({ key, label, visible, axis }) => ({
        key,
        label,
        visible,
        start:
          axis == null ? undefined : (
            <Text.Text level="small" color={9} weight={500}>
              {axis.toUpperCase()}
            </Text.Text>
          ),
      })),
  }));

export const Legend = ({
  onLineColorChange,
  onLineLabelChange,
  onLineVisibleChange,
  ranges,
  hiddenRanges,
  onRangeVisibleChange,
  onRangeColorChange,
  onHoverChange,
  footer,
  ...rest
}: LegendProps): ReactElement => {
  const { lines } = useContext("LinePlot.Legend");
  const handlers: LineHandlers = {
    onColorChange: onLineColorChange,
    onLabelChange: onLineLabelChange,
    onVisibleChange: onLineVisibleChange,
    onHoverChange,
  };
  if (ranges != null)
    return (
      <Base.Grouped
        data={groupRanges(ranges, lines, hiddenRanges)}
        onVisibleChange={onLineVisibleChange}
        onLabelChange={onLineLabelChange}
        onGroupVisibleChange={onRangeVisibleChange}
        onGroupColorChange={onRangeColorChange}
        onHoverChange={onHoverChange}
        footer={footer}
        nested
        {...rest}
      />
    );
  const groups = groupLines(lines);
  if (groups.length === 1)
    return <Base.Simple data={groups[0].data} {...handlers} {...rest} />;
  return <Base.Grouped data={groups} {...handlers} {...rest} />;
};
Legend.displayName = "LinePlot.Legend";
