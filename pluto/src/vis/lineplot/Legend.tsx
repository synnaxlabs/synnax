// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Optional } from "@synnaxlabs/x";
import { memo, type ReactElement, useMemo } from "react";

import { Flex } from "@/flex";
import { useUniqueKey } from "@/hooks/useUniqueKey";
import { Text } from "@/text";
import { Legend as Core } from "@/vis/legend";
import { Entries, type EntryData } from "@/vis/legend/Entries";
import { type LineSpec, useContext, useGridEntry } from "@/vis/lineplot/LinePlot";

export interface LegendProps extends Omit<Core.SimpleProps, "data" | "onEntryChange"> {
  variant?: "floating" | "fixed";
  onLineChange?: (line: LineSpec) => void;
}

export const Legend = ({ variant = "floating", ...rest }: LegendProps): ReactElement =>
  variant === "floating" ? <Floating {...rest} /> : <Fixed {...rest} />;

interface FloatingProps extends Omit<LegendProps, "variant"> {}

const Floating = memo(({ onLineChange, ...rest }: FloatingProps): ReactElement => {
  const { lines } = useContext("FloatingLegend");
  const groups: Core.GroupData[] = useGroupData(lines);
  // if we only have the y1 group, use the simple legend
  if (groups.length === 1 && groups[0].key === "y1")
    return <Core.Simple data={groups[0].data} onEntryChange={onLineChange} {...rest} />;
  return <Core.Grouped data={groups} onEntryChange={onLineChange} {...rest} />;
});
Floating.displayName = "LinePlot.FloatingLegend";

interface FixedProps extends Pick<LegendProps, "onLineChange"> {}

const useGroupData = (lines: LineSpec[]): Core.GroupData[] => {
  const groups: Core.GroupData[] = useMemo(() => {
    const groups: Core.GroupData[] = [];
    const data1 = lines.filter((l) => l.key.startsWith("y1"));
    const data2 = lines.filter((l) => l.key.startsWith("y2"));
    if (data1.length > 0) groups.push({ key: "y1", name: "Y1", data: data1 });
    if (data2.length > 0) groups.push({ key: "y2", name: "Y2", data: data2 });
    return groups;
  }, [lines]);
  return groups;
};

const Fixed = ({ onLineChange }: FixedProps): ReactElement | null => {
  const { lines } = useContext("Legend");
  const groups: Core.GroupData[] = useGroupData(lines);
  const key = useUniqueKey();
  const gridStyle = useGridEntry(
    { key, size: lines.length > 0 ? 36 : 0, loc: "top", order: 5 },
    "Legend",
  );
  if (groups.length === 0) return null;
  if (groups.length === 1) {
    if (groups[0].key === "y1")
      return (
        <Flex.Box align="center" justify="start" x style={gridStyle}>
          <Entries data={groups[0].data} onEntryChange={onLineChange} background={0} />
        </Flex.Box>
      );
    return <FocusedGroup name="Y2" data={groups[0].data} onLineChange={onLineChange} />;
  }
  return (
    <Flex.Box align="center" justify="start" x style={gridStyle} gap="huge">
      <FocusedGroup name="Y1" data={groups[0].data} onLineChange={onLineChange} />
      <FocusedGroup name="Y2" data={groups[1].data} onLineChange={onLineChange} />
    </Flex.Box>
  );
};

interface FocusedGroupProps {
  name: string;
  data: Optional<EntryData, "visible">[];
  onLineChange?: (line: LineSpec) => void;
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
