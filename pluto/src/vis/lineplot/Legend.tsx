// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { memo, type ReactElement } from "react";

import { Flex } from "@/flex";
import { useUniqueKey } from "@/hooks/useUniqueKey";
import { Legend as Core } from "@/vis/legend";
import { Entries } from "@/vis/legend/Entries";
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
  return <Core.Simple data={lines} onEntryChange={onLineChange} {...rest} />;
});
Floating.displayName = "LinePlot.FloatingLegend";

interface FixedProps extends Pick<LegendProps, "onLineChange"> {}

const Fixed = ({ onLineChange }: FixedProps): ReactElement => {
  const { lines } = useContext("Legend");
  const key = useUniqueKey();
  const gridStyle = useGridEntry(
    { key, size: lines.length > 0 ? 36 : 0, loc: "top", order: 5 },
    "Legend",
  );
  return (
    <Flex.Box align="center" justify="start" x style={gridStyle}>
      <Entries data={lines} onEntryChange={onLineChange} background={0} />
    </Flex.Box>
  );
};
