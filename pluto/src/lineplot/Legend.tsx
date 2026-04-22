// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type optional } from "@synnaxlabs/x";
import { memo, type ReactElement, useCallback, useMemo } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Divider } from "@/divider";
import { Flex } from "@/flex";
import { useUniqueKey } from "@/hooks/useUniqueKey";
import { Icon } from "@/icon";
import {
  type LineSpec,
  type SubGroup,
  useContext,
  useGridEntry,
} from "@/lineplot/LinePlot";
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
  const { lines, highlightedSubGroup } = useContext("LinePlot.Legend");
  const groups: Base.GroupData[] = useGroupData(lines);
  const subGroups = useSubGroups(lines);
  const hsgi = useHighlightedSubGroupIndex(subGroups, highlightedSubGroup);
  if (groups.length === 1)
    return (
      <Base.Simple
        data={groups[0].data}
        onEntryChange={onLineChange}
        highlightedSubGroupIndex={hsgi}
        {...rest}
      >
        <SubGroupKey subGroups={subGroups} onLineChange={onLineChange} />
      </Base.Simple>
    );
  return (
    <Base.Grouped
      data={groups}
      onEntryChange={onLineChange}
      highlightedSubGroupIndex={hsgi}
      {...rest}
    >
      <SubGroupKey subGroups={subGroups} onLineChange={onLineChange} />
    </Base.Grouped>
  );
});
Floating.displayName = "LinePlot.FloatingLegend";

interface FixedProps extends Pick<LegendProps, "onLineChange"> {}

const useGroupData = (lines: LineSpec[]): Base.GroupData[] => {
  const groups: Base.GroupData[] = useMemo(() => {
    const subGroupKeys: string[] = [];
    for (const line of lines)
      if (line.subGroup != null && !subGroupKeys.includes(line.subGroup.key))
        subGroupKeys.push(line.subGroup.key);

    const groupInfo: Record<string, LineSpec[]> = {};
    for (const line of lines) {
      const group = groupInfo[line.legendGroup];
      if (group != null) group.push(line);
      else groupInfo[line.legendGroup] = [line];
    }
    const groups = Object.entries(groupInfo).map(([key, data]) => ({
      key,
      name: key,
      data: data
        .sort((a, b) => a.key.localeCompare(b.key))
        .map((line) => ({
          ...line,
          subGroupIndex:
            line.subGroup != null
              ? subGroupKeys.indexOf(line.subGroup.key) + 1
              : undefined,
        })),
    }));
    groups.sort((a, b) => a.name.localeCompare(b.name));
    return groups;
  }, [lines]);
  return groups;
};

const Fixed = ({ onLineChange }: FixedProps): ReactElement | null => {
  const { lines, highlightedSubGroup } = useContext("LinePlot.Legend");
  const groups: Base.GroupData[] = useGroupData(lines);
  const subGroups = useSubGroups(lines);
  const hsgi = useHighlightedSubGroupIndex(subGroups, highlightedSubGroup);
  const key = useUniqueKey();
  const gridStyle = useGridEntry(
    { key, size: lines.length > 0 ? 36 : 0, loc: "top", order: 5 },
    "LinePlot.Legend",
  );
  if (groups.length === 0) return null;
  if (groups.length === 1)
    return (
      <Flex.Box align="center" justify="start" x style={gridStyle}>
        <Entries
          data={groups[0].data}
          onEntryChange={onLineChange}
          background={0}
          highlightedSubGroupIndex={hsgi}
        />
        <SubGroupKey subGroups={subGroups} onLineChange={onLineChange} />
      </Flex.Box>
    );
  return (
    <Flex.Box align="center" justify="start" x style={gridStyle} gap="huge">
      <FocusedGroup name="Y1" data={groups[0].data} onLineChange={onLineChange} />
      <FocusedGroup name="Y2" data={groups[1].data} onLineChange={onLineChange} />
      <SubGroupKey subGroups={subGroups} onLineChange={onLineChange} />
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

interface SubGroupInfo {
  key: string;
  name: string;
  index: number;
}

const useSubGroups = (lines: LineSpec[]): SubGroupInfo[] =>
  useMemo(() => {
    const seen = new Map<string, SubGroup>();
    for (const line of lines)
      if (line.subGroup != null && !seen.has(line.subGroup.key))
        seen.set(line.subGroup.key, line.subGroup);

    let i = 1;
    return Array.from(seen, ([key, { name }]) => ({ key, name, index: i++ }));
  }, [lines]);

const useHighlightedSubGroupIndex = (
  subGroups: SubGroupInfo[],
  highlightedSubGroup: string | null,
): number | undefined =>
  useMemo(() => {
    if (highlightedSubGroup == null) return undefined;
    const sg = subGroups.find((s) => s.key === highlightedSubGroup);
    return sg?.index;
  }, [subGroups, highlightedSubGroup]);

interface SubGroupKeyProps {
  subGroups: SubGroupInfo[];
  onLineChange?: (line: optional.Optional<LineSpec, "legendGroup">) => void;
}

const SubGroupKey = ({
  subGroups,
  onLineChange,
}: SubGroupKeyProps): ReactElement | null => {
  const { highlightedSubGroup, setHighlightedSubGroup, lines } =
    useContext("SubGroupKey");

  const toggleSubGroupVisibility = useCallback(
    (subGroupKey: string) => {
      if (onLineChange == null) return;
      const subGroupLines = lines.filter((l) => l.subGroup?.key === subGroupKey);
      const allVisible = subGroupLines.every((l) => l.visible);
      subGroupLines.forEach((l) => onLineChange({ ...l, visible: !allVisible }));
    },
    [lines, onLineChange],
  );

  if (subGroups.length === 0) return null;
  return (
    <>
      <Divider.Divider x padded />
      <Flex.Box y empty>
        {subGroups.map(({ key, name, index }) => {
          const subGroupLines = lines.filter((l) => l.subGroup?.key === key);
          const allVisible = subGroupLines.every((l) => l.visible);
          return (
            <Flex.Box
              key={key}
              x
              align="center"
              justify="between"
              gap="small"
              style={{ cursor: "pointer" }}
              onMouseEnter={() => setHighlightedSubGroup(key)}
              onMouseLeave={() => setHighlightedSubGroup(null)}
            >
              <Flex.Box x align="center" gap="small">
                <Text.Text
                  level="small"
                  weight={600}
                  color={highlightedSubGroup === key ? 10 : 9}
                >
                  {index}
                </Text.Text>
                <Text.Text level="small" color={highlightedSubGroup === key ? 10 : 8}>
                  {name}
                </Text.Text>
              </Flex.Box>
              {onLineChange != null && (
                <Button.Button
                  className={CSS.B("visible-toggle")}
                  onClick={() => toggleSubGroupVisibility(key)}
                  size="tiny"
                  variant="text"
                >
                  {allVisible ? <Icon.Visible /> : <Icon.Hidden />}
                </Button.Button>
              )}
            </Flex.Box>
          );
        })}
      </Flex.Box>
    </>
  );
};
