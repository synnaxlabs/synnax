// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type optional } from "@synnaxlabs/x";
import { memo, type ReactElement, useMemo } from "react";

import { Flex } from "@/flex";
import { useUniqueKey } from "@/hooks/useUniqueKey";
import { type LineSpec, useContext, useGridEntry } from "@/lineplot/Frame";
import { Text } from "@/text";
import { Legend as Base } from "@/vis/legend";
import { Entries, type EntriesProps, type EntryData } from "@/vis/legend/Entries";

type LineHandlers = Pick<
  EntriesProps,
  "onColorChange" | "onLabelChange" | "onVisibleChange"
>;

export interface LegendProps extends Omit<
  Base.SimpleProps,
  "data" | "onColorChange" | "onLabelChange" | "onVisibleChange"
> {
  variant?: "floating" | "fixed";
  onLineColorChange?: EntriesProps["onColorChange"];
  onLineLabelChange?: EntriesProps["onLabelChange"];
  onLineVisibleChange?: EntriesProps["onVisibleChange"];
}

export const Legend = ({ variant = "floating", ...rest }: LegendProps): ReactElement =>
  variant === "floating" ? <Floating {...rest} /> : <Fixed {...rest} />;

interface FloatingProps extends Omit<LegendProps, "variant"> {}

const Floating = memo(
  ({
    onLineColorChange,
    onLineLabelChange,
    onLineVisibleChange,
    ...rest
  }: FloatingProps): ReactElement => {
    const { lines } = useContext("LinePlot.Legend");
    const groups: Base.GroupData[] = useGroupData(lines);
    const handlers: LineHandlers = {
      onColorChange: onLineColorChange,
      onLabelChange: onLineLabelChange,
      onVisibleChange: onLineVisibleChange,
    };
    if (groups.length === 1)
      return <Base.Simple data={groups[0].data} {...handlers} {...rest} />;
    return <Base.Grouped data={groups} {...handlers} {...rest} />;
  },
);
Floating.displayName = "LinePlot.FloatingLegend";

interface FixedProps extends Pick<
  LegendProps,
  "onLineColorChange" | "onLineLabelChange" | "onLineVisibleChange"
> {}

const useGroupData = (lines: LineSpec[]): Base.GroupData[] => {
  const groups: Base.GroupData[] = useMemo(() => {
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
    groups.sort((a, b) => a.name.localeCompare(b.name));
    return groups;
  }, [lines]);
  return groups;
};

const Fixed = ({
  onLineColorChange,
  onLineLabelChange,
  onLineVisibleChange,
}: FixedProps): ReactElement | null => {
  const { lines } = useContext("LinePlot.Legend");
  const groups: Base.GroupData[] = useGroupData(lines);
  const key = useUniqueKey();
  const gridStyle = useGridEntry(
    { key, size: lines.length > 0 ? 36 : 0, loc: "top", order: 5 },
    "LinePlot.Legend",
  );
  const handlers: LineHandlers = {
    onColorChange: onLineColorChange,
    onLabelChange: onLineLabelChange,
    onVisibleChange: onLineVisibleChange,
  };
  if (groups.length === 0) return null;
  if (groups.length === 1)
    return (
      <Flex.Box align="center" justify="start" x style={gridStyle}>
        <Entries data={groups[0].data} {...handlers} background={0} />
      </Flex.Box>
    );
  return (
    <Flex.Box align="center" justify="start" x style={gridStyle} gap="huge">
      <FocusedGroup name="Y1" data={groups[0].data} {...handlers} />
      <FocusedGroup name="Y2" data={groups[1].data} {...handlers} />
    </Flex.Box>
  );
};
Fixed.displayName = "LinePlot.FixedLegend";

interface FocusedGroupProps extends LineHandlers {
  name: string;
  data: optional.Optional<EntryData, "visible">[];
}

const FocusedGroup = ({ name, data, ...handlers }: FocusedGroupProps): ReactElement => (
  <Flex.Box x>
    <Text.Text level="small">{name}</Text.Text>
    <Entries data={data} {...handlers} background={0} />
  </Flex.Box>
);
