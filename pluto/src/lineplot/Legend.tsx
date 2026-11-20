// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { compare } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { type LineSpec, useContext } from "@/lineplot/Frame";
import { Legend as Base } from "@/vis/legend";
import type { EntriesProps } from "@/vis/legend/Entries";

type LineHandlers = Pick<
  EntriesProps,
  "onColorChange" | "onLabelChange" | "onVisibleChange"
>;

export interface LegendProps extends Omit<
  Base.SimpleProps,
  "data" | "onColorChange" | "onLabelChange" | "onVisibleChange"
> {
  onLineColorChange?: EntriesProps["onColorChange"];
  onLineLabelChange?: EntriesProps["onLabelChange"];
  onLineVisibleChange?: EntriesProps["onVisibleChange"];
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

export const Legend = ({
  onLineColorChange,
  onLineLabelChange,
  onLineVisibleChange,
  ...rest
}: LegendProps): ReactElement => {
  const { lines } = useContext("LinePlot.Legend");
  const groups = groupLines(lines);
  const handlers: LineHandlers = {
    onColorChange: onLineColorChange,
    onLabelChange: onLineLabelChange,
    onVisibleChange: onLineVisibleChange,
  };
  if (groups.length === 1)
    return <Base.Simple data={groups[0].data} {...handlers} {...rest} />;
  return <Base.Grouped data={groups} {...handlers} {...rest} />;
};
Legend.displayName = "LinePlot.Legend";
