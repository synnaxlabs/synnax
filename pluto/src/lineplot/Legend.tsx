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
import { type LineSpec, useContext, useGridEntry } from "@/lineplot/LinePlot";
import { Text } from "@/text";
import { Legend as Base } from "@/vis/legend";
import { Entries, type EntryData } from "@/vis/legend/Entries";

export interface LegendProps extends Omit<Base.SimpleProps, "data" | "onEntryChange"> {
  variant?: "floating" | "fixed";
  onLineChange?: (line: optional.Optional<LineSpec, "legendGroup">) => void;
}

export const Legend = ({ variant = "floating", ...rest }: LegendProps): ReactElement =>
  variant === "floating" ? <Floating {...rest} /> : <Fixed {...rest} />;

interface FloatingProps extends Omit<LegendProps, "variant"> {}

const Floating = memo(({ onLineChange, ...rest }: FloatingProps): ReactElement => {
  const { lines } = useContext("LinePlot.Legend");
  const groups: Base.GroupData[] = useGroupData(lines);
  if (groups.length === 1)
    return <Base.Simple data={groups[0].data} onEntryChange={onLineChange} {...rest} />;
  return <Base.Grouped data={groups} onEntryChange={onLineChange} {...rest} />;
});
Floating.displayName = "LinePlot.FloatingLegend";

interface FixedProps extends Pick<LegendProps, "onLineChange"> {}

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

const Fixed = ({ onLineChange }: FixedProps): ReactElement | null => {
  const { lines } = useContext("LinePlot.Legend");
  const groups: Base.GroupData[] = useGroupData(lines);
  const key = useUniqueKey();
  const gridStyle = useGridEntry(
    { key, size: lines.length > 0 ? 36 : 0, loc: "top", order: 5 },
    "LinePlot.Legend",
  );
  if (groups.length === 0) return null;
  if (groups.length === 1)
    return (
      <Flex.Box align="center" justify="start" x style={gridStyle}>
        <Entries data={groups[0].data} onEntryChange={onLineChange} background={0} />
      </Flex.Box>
    );
  return (
    <Flex.Box align="center" justify="start" x style={gridStyle} gap="huge">
      <FocusedGroup name="Y1" data={groups[0].data} onLineChange={onLineChange} />
      <FocusedGroup name="Y2" data={groups[1].data} onLineChange={onLineChange} />
    </Flex.Box>
  );
};
Fixed.displayName = "LinePlot.FixedLegend";

interface FocusedGroupProps {
  name: string;
  data: optional.Optional<EntryData, "visible">[];
  onLineChange?: (line: optional.Optional<LineSpec, "legendGroup">) => void;
}

const FocusedGroup = ({
  name,
  data,
  onLineChange,
}: FocusedGroupProps): ReactElement => (
  <Flex.Box x>
    <Text.Text level="small">{name}</Text.Text>
    <Entries data={data} onEntryChange={onLineChange} background={0} />
  </Flex.Box>
);
